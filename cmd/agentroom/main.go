package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dedrisproject/agentroom/internal/api"
	"github.com/dedrisproject/agentroom/internal/auth"
	"github.com/dedrisproject/agentroom/internal/config"
	appdb "github.com/dedrisproject/agentroom/internal/db"
	"github.com/dedrisproject/agentroom/internal/web"
	"golang.org/x/crypto/bcrypt"
)

// version is set via ldflags: -X main.version=vX.Y.Z
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init", "setup":
			runInit(version)
			return
		case "--version", "-version", "version":
			fmt.Println(version)
			return
		case "--help", "-help", "-h", "help":
			printHelp()
			return
		}
	}

	// ---- Load config ----
	// Priority: flags > env vars > config file > defaults
	cfg := config.Defaults()

	// Config file: --config flag or AGENTROOM_CONFIG env
	cfgPath := ""
	for i, arg := range os.Args[1:] {
		if arg == "--config" && i+1 < len(os.Args)-1 {
			cfgPath = os.Args[i+2]
			break
		}
	}
	if cfgPath == "" {
		cfgPath = os.Getenv("AGENTROOM_CONFIG")
	}
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	if loaded, err := config.Load(cfgPath); err == nil {
		cfg = loaded
	}

	// Env var overrides
	config.LoadFromEnv(&cfg)

	// Flag overrides
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--port", "-port":
			if i+1 < len(os.Args) {
				cfg.Port = os.Args[i+1]
				i++
			}
		case "--db", "-db":
			if i+1 < len(os.Args) {
				cfg.DB = os.Args[i+1]
				i++
			}
		case "--bind", "-bind":
			if i+1 < len(os.Args) {
				cfg.Bind = os.Args[i+1]
				i++
			}
		case "--config", "-config":
			i++ // already handled above
		}
	}

	// ---- Logger ----
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	slog.Info("AgentRoom starting", "version", version, "db", cfg.DB, "port", cfg.Port, "lang", cfg.Lang)

	// ---- Database ----
	database, err := appdb.Open(cfg.DB)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		slog.Error("tip: run 'agentroom init' to set up the database")
		os.Exit(1)
	}
	defer database.Close()

	if err := appdb.Migrate(database); err != nil {
		slog.Error("failed to migrate database", "err", err)
		os.Exit(1)
	}

	// ---- First-run (fallback if init was not run) ----
	adminPassword := os.Getenv("AGENTROOM_ADMIN_PASSWORD")
	if _, ok := appdb.GetSetting(database, "admin_password_hash"); !ok {
		if adminPassword == "" {
			adminPassword = generatePassword(16)
			slog.Info(">>> AgentRoom admin password: "+adminPassword+"  (shown only once)",
				"action", "SAVE THIS PASSWORD")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash admin password", "err", err)
			os.Exit(1)
		}
		if err := appdb.SetSetting(database, "admin_password_hash", string(hash)); err != nil {
			slog.Error("failed to store admin password hash", "err", err)
			os.Exit(1)
		}
	}

	sessionSecret := ""
	if s, ok := appdb.GetSetting(database, "session_secret"); ok {
		sessionSecret = s
	} else {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			slog.Error("failed to generate session secret", "err", err)
			os.Exit(1)
		}
		sessionSecret = hex.EncodeToString(secretBytes)
		if err := appdb.SetSetting(database, "session_secret", sessionSecret); err != nil {
			slog.Error("failed to store session secret", "err", err)
			os.Exit(1)
		}
	}

	// ---- Auth config ----
	secretBytes, err := hex.DecodeString(sessionSecret)
	if err != nil {
		secretBytes = []byte(sessionSecret)
	}

	authCfg := auth.Config{
		SessionSecret: secretBytes,
		AdminName:     cfg.AdminName,
	}

	// ---- Web UI handler ----
	webHandler, err := web.New(database, authCfg, cfg.AdminName, cfg.Lang)
	if err != nil {
		slog.Error("failed to initialize web UI", "err", err)
		os.Exit(1)
	}

	// ---- Router ----
	handler := api.NewRouter(database, authCfg, cfg.AdminName, webHandler)

	// ---- HTTP server ----
	addr := cfg.Bind + ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("stopped")
}

func printHelp() {
	fmt.Printf("AgentRoom %s — shared inbox for AI coding agents\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  agentroom init              Interactive setup wizard (run first)")
	fmt.Println("  agentroom [flags]           Start the server")
	fmt.Println("  agentroom --version         Print version")
	fmt.Println()
	fmt.Println("Server flags:")
	fmt.Println("  --config <path>   Config file (default: ~/.agentroom/agentroom.conf)")
	fmt.Println("  --port <port>     HTTP port (overrides config/env)")
	fmt.Println("  --db <path>       SQLite database path")
	fmt.Println("  --bind <addr>     Bind address")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  AGENTROOM_PORT, AGENTROOM_DB, AGENTROOM_BIND")
	fmt.Println("  AGENTROOM_BASE_URL, AGENTROOM_ADMIN_AGENT_NAME")
	fmt.Println("  AGENTROOM_LANG (en|it)")
	fmt.Println("  AGENTROOM_ADMIN_PASSWORD (first-run only)")
}

func generatePassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
