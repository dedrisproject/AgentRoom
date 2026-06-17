package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dedrisproject/agentroom/internal/config"
	appdb "github.com/dedrisproject/agentroom/internal/db"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const mascotRobot = `
                      ✦
                      │
                   ╭──┴──╮
                   │◉   ◉│
                   │  ◡  │
                  ╭┴─────┴╮
                ◀─┤▒▒▒▒▒▒▒├─▶
                  │▒░░░░░▒│
                  ╰─┬───┬─╯
                    ╵   ╵
`

const mascotLogo = `
   █████╗  ██████╗ ███████╗███╗   ██╗████████╗
  ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝
  ███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║
  ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║
  ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║
  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝
        ██████╗  ██████╗  ██████╗ ███╗   ███╗
       ██╔══██╗██╔═══██╗██╔═══██╗████╗ ████║
       ██████╔╝██║   ██║██║   ██║██╔████╔██║
       ██╔══██╗██║   ██║██║   ██║██║╚██╔╝██║
       ██║  ██║╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
       ╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝
`

const mascotTagline = "Shared inbox for AI coding agents"

const mascotSmall = `
   [◉‿◉]   AgentRoom %s
           Shared inbox for AI coding agents
`

// printBanner renders the startup banner: a colored robot mascot + wordmark on a
// TTY, or a compact single-line variant when output is piped (non-TTY).
func printBanner(ver string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Printf(mascotSmall, ver)
		return
	}
	fmt.Print(colorCyan + mascotRobot + colorReset)
	fmt.Print(colorWhite + mascotLogo + colorReset)
	fmt.Printf("\n  %s%s%s \u00b7 %s\n", colorCyan, mascotTagline, colorReset, ver)
}

// runInit runs the interactive setup wizard.
func runInit(ver string) {
	printBanner(ver)

	cfgPath := config.DefaultConfigPath()
	cfg := config.Defaults()
	cfg.FilePath = cfgPath

	// Check existing config
	existing, err := config.Load(cfgPath)
	if err == nil && existing.FilePath == cfgPath {
		_, statErr := os.Stat(cfgPath)
		if statErr == nil {
			color(colorCyan, "\n  Existing configuration found at "+cfgPath)
			if !askYesNo("  Reconfigure?", false) {
				fmt.Println()
				color(colorGreen, "  ✓ Keeping existing configuration.")
				printStartHint(existing)
				return
			}
		}
	}
	cfg = existing

	color(colorCyan, "\n  Setup Wizard")
	fmt.Println("  ─────────────────────────────────────────────────")
	fmt.Println("  You can re-run this wizard at any time:")
	color(colorYellow, "    agentroom init")
	fmt.Println()

	// ---- Language ----
	color(colorWhite, "  UI Language")
	fmt.Println("  Supported: en (English), it (Italiano)")
	cfg.Lang = askInput("  Language", cfg.Lang)
	if cfg.Lang != "en" && cfg.Lang != "it" {
		cfg.Lang = "en"
	}
	fmt.Println()

	// ---- Port ----
	color(colorWhite, "  HTTP Port")
	cfg.Port = askInput("  Port", cfg.Port)
	fmt.Println()

	// ---- Data directory ----
	color(colorWhite, "  Data Directory")
	fmt.Println("  The database file will be created here automatically.")
	dataDir := filepath.Dir(cfg.DB)
	dataDir = askInput("  Data directory", dataDir)
	cfg.DB = filepath.Join(dataDir, "agentroom.db")
	fmt.Printf("  → Database: %s\n\n", cfg.DB)

	// ---- Base URL ----
	color(colorWhite, "  Public Base URL  (optional)")
	fmt.Println("  Used in generated agent-room.md curl commands.")
	fmt.Println("  Leave blank to auto-detect from incoming requests.")
	cfg.BaseURL = askInput("  Base URL", cfg.BaseURL)
	fmt.Println()

	// ---- Admin name ----
	color(colorWhite, "  Admin Display Name")
	fmt.Println("  Name shown for messages sent by the admin.")
	cfg.AdminName = askInput("  Admin name", cfg.AdminName)
	fmt.Println()

	// ---- Admin password ----
	color(colorWhite, "  Admin Password")
	fmt.Println("  Leave blank to auto-generate a secure random password.")
	fmt.Println("  The password is stored as a bcrypt hash — never in plaintext.")
	adminPassword := askPassword("  Password (hidden)")
	fmt.Println()

	// ---- Summary ----
	color(colorCyan, "  Summary")
	fmt.Println("  ─────────────────────────────────────────────────")
	fmt.Printf("  Language:      %s\n", cfg.Lang)
	fmt.Printf("  Port:          %s\n", cfg.Port)
	fmt.Printf("  Database:      %s\n", cfg.DB)
	fmt.Printf("  Admin name:    %s\n", cfg.AdminName)
	fmt.Printf("  Base URL:      %s\n", orDefault(cfg.BaseURL, "(auto-detect)"))
	fmt.Printf("  Password:      %s\n", orDefault(adminPassword, "(will be auto-generated on start)"))
	fmt.Println("  ─────────────────────────────────────────────────")
	fmt.Println()

	if !askYesNo("  Apply this configuration?", true) {
		color(colorYellow, "  Cancelled. Nothing was changed.")
		fmt.Println()
		return
	}

	// ---- Apply ----
	// Create data directory
	if err := os.MkdirAll(filepath.Dir(cfg.DB), 0755); err != nil {
		fatalf("  ✗ Failed to create data directory: %v", err)
	}
	color(colorGreen, "  ✓ Data directory ready: "+filepath.Dir(cfg.DB))

	// Save config file
	if err := config.Save(cfg); err != nil {
		fatalf("  ✗ Failed to save config: %v", err)
	}
	color(colorGreen, "  ✓ Configuration saved: "+cfgPath)

	// Open DB and run migrations
	database, err := appdb.Open(cfg.DB)
	if err != nil {
		fatalf("  ✗ Failed to open database: %v", err)
	}
	defer database.Close()

	if err := appdb.Migrate(database); err != nil {
		fatalf("  ✗ Failed to initialize database: %v", err)
	}
	color(colorGreen, "  ✓ Database initialized: "+cfg.DB)

	// Set admin password
	if adminPassword == "" {
		if _, ok := appdb.GetSetting(database, "admin_password_hash"); !ok {
			adminPassword = generatePassword(16)
			fmt.Println()
			color(colorYellow, "  ┌────────────────────────────────────────────────────┐")
			color(colorYellow, "  │   Admin password: "+padRight(adminPassword, 33)+"│")
			color(colorYellow, "  │   Save this — it is shown only once!               │")
			color(colorYellow, "  └────────────────────────────────────────────────────┘")
		}
	}

	if adminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			fatalf("  ✗ Failed to hash password: %v", err)
		}
		if err := appdb.SetSetting(database, "admin_password_hash", string(hash)); err != nil {
			fatalf("  ✗ Failed to store password hash: %v", err)
		}
		color(colorGreen, "  ✓ Admin password set (bcrypt hash stored in database)")
	}

	// Generate session secret if missing
	if _, ok := appdb.GetSetting(database, "session_secret"); !ok {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			fatalf("  ✗ Failed to generate session secret: %v", err)
		}
		if err := appdb.SetSetting(database, "session_secret", hex.EncodeToString(secretBytes)); err != nil {
			fatalf("  ✗ Failed to store session secret: %v", err)
		}
		color(colorGreen, "  ✓ Session secret generated")
	}

	fmt.Println()
	color(colorGreen, "  ✓ Setup complete!")
	printStartHint(cfg)
}

func printStartHint(cfg config.Config) {
	fmt.Println()
	color(colorCyan, "  Next steps")
	fmt.Println("  ─────────────────────────────────────────────────")
	fmt.Printf("  Start:         agentroom --config %s\n", cfg.FilePath)
	fmt.Printf("  URL:           http://localhost:%s\n", cfg.Port)
	fmt.Println("  Then:          log in → create a room → add an agent")
	fmt.Println("  Re-run setup:  agentroom init")
	fmt.Println()
}

// ---- Terminal helpers ----

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[1;37m"
)

func color(c, s string) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println(c + s + colorReset)
	} else {
		fmt.Println(s)
	}
}

var stdinReader = bufio.NewReader(os.Stdin)

func askInput(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func askPassword(prompt string) string {
	fmt.Printf("%s: ", prompt)
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	line, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(line)
}

func askYesNo(prompt string, defaultYes bool) bool {
	hint := "Y/n"
	if !defaultYes {
		hint = "y/N"
	}
	fmt.Printf("%s [%s]: ", prompt, hint)
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes" || line == "si" || line == "sì"
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func fatalf(format string, args ...interface{}) {
	color(colorRed, fmt.Sprintf(format, args...))
	os.Exit(1)
}
