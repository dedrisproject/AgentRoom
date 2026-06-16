package web

import (
	"database/sql"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dedrisproject/agentroom/internal/auth"
	"github.com/dedrisproject/agentroom/internal/i18n"
	"github.com/dedrisproject/agentroom/internal/store"
)

// Handler serves the admin web UI.
type Handler struct {
	db          *sql.DB
	authCfg     auth.Config
	adminName   string
	defaultLang string
	loginTmpl   *template.Template
	dashTmpl    *template.Template
	roomTmpl    *template.Template
	uiMux       *http.ServeMux
}

// Thread groups a root message with its replies for the UI.
type Thread struct {
	Root    *store.Message
	Replies []*store.Message
}

// pageData is the base data passed to every template.
type pageData struct {
	Lang          string
	T             func(string) string
	SwitchURL     func(string) string
	SupportedLangs []string
}

// DashboardData is passed to the dashboard template.
type DashboardData struct {
	pageData
	Rooms    []store.RoomSummary
	Blockers []*store.Message
}

// RoomData is passed to the room template.
type RoomData struct {
	pageData
	Room       *store.Room
	Agents     []*store.Agent
	Threads    []Thread
	KnownRoles []string
	KnownRepos []string
	AdminName  string
}

// LoginData is passed to the login template.
type LoginData struct {
	pageData
}

// New creates a new web Handler with compiled templates.
// Each page uses its own template set (base.html + page) to avoid block/define conflicts.
func New(db *sql.DB, authCfg auth.Config, adminName, defaultLang string) (*Handler, error) {
	sub, err := fs.Sub(FS, "templates")
	if err != nil {
		return nil, err
	}

	fns := template.FuncMap{
		"js": func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `'`, `\'`)
			return s
		},
	}

	parse := func(pages ...string) (*template.Template, error) {
		return template.New("").Funcs(fns).ParseFS(sub, pages...)
	}

	loginTmpl, err := parse("base.html", "login.html")
	if err != nil {
		return nil, err
	}
	dashTmpl, err := parse("base.html", "dashboard.html")
	if err != nil {
		return nil, err
	}
	roomTmpl, err := parse("base.html", "room.html")
	if err != nil {
		return nil, err
	}

	if defaultLang == "" {
		defaultLang = "en"
	}

	h := &Handler{
		db:          db,
		authCfg:     authCfg,
		adminName:   adminName,
		defaultLang: defaultLang,
		loginTmpl:   loginTmpl,
		dashTmpl:    dashTmpl,
		roomTmpl:    roomTmpl,
		uiMux:       http.NewServeMux(),
	}

	h.uiMux.HandleFunc("GET /login", h.loginPage)
	h.uiMux.HandleFunc("GET /", h.requireSession(h.dashboard))
	h.uiMux.HandleFunc("GET /rooms/{id}", h.requireSession(h.roomView))
	h.uiMux.HandleFunc("GET /lang", h.setLang)

	return h, nil
}

// ServeHTTP dispatches non-API, non-static requests to the UI mux.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.uiMux.ServeHTTP(w, r)
}

// RegisterRoutes registers the /static/ file server on the external mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	staticFS, _ := fs.Sub(FS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
}

// setLang sets the language cookie and redirects back.
func (h *Handler) setLang(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("l")
	if lang == "" {
		lang = h.defaultLang
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "agentroom_lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	back := r.URL.Query().Get("back")
	if back == "" {
		back = "/"
	}
	http.Redirect(w, r, back, http.StatusFound)
}

func (h *Handler) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.ValidateSession(r, h.authCfg.SessionSecret) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// detectLang returns the active language for a request.
func (h *Handler) detectLang(r *http.Request) string {
	return i18n.Detect(r, h.defaultLang)
}

func (h *Handler) makePageData(r *http.Request) pageData {
	lang := h.detectLang(r)
	return pageData{
		Lang:          lang,
		SupportedLangs: i18n.SupportedLangs,
		T:             func(key string) string { return i18n.T(lang, key) },
		SwitchURL: func(target string) string {
			return "/lang?l=" + target + "&back=" + r.URL.Path
		},
	}
}

func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	if auth.ValidateSession(r, h.authCfg.SessionSecret) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.renderWith(w, h.loginTmpl, LoginData{pageData: h.makePageData(r)})
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	rooms, err := store.ListRooms(h.db)
	if err != nil {
		h.internalError(w, err)
		return
	}
	blockers, err := store.ListGlobalBlockers(h.db)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.renderWith(w, h.dashTmpl, DashboardData{
		pageData: h.makePageData(r),
		Rooms:    rooms,
		Blockers: blockers,
	})
}

func (h *Handler) roomView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	room, err := store.GetRoom(h.db, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	agents, err := store.ListAgents(h.db, id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	messages, err := store.GetRoomMessages(h.db, id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	knownRoles, knownRepos := collectAutocomplete(agents)
	h.renderWith(w, h.roomTmpl, RoomData{
		pageData:   h.makePageData(r),
		Room:       room,
		Agents:     agents,
		Threads:    buildThreads(messages),
		KnownRoles: knownRoles,
		KnownRepos: knownRepos,
		AdminName:  h.adminName,
	})
}

func buildThreads(messages []*store.Message) []Thread {
	roots := map[int64]*store.Message{}
	replies := map[int64][]*store.Message{}
	for _, m := range messages {
		if m.ParentID == nil {
			roots[m.ID] = m
		} else {
			replies[*m.ParentID] = append(replies[*m.ParentID], m)
		}
	}
	seen := map[int64]bool{}
	var blockers, normal []Thread
	for _, m := range messages {
		rootID := m.ID
		if m.ParentID != nil {
			rootID = *m.ParentID
		}
		if seen[rootID] {
			continue
		}
		seen[rootID] = true
		root, ok := roots[rootID]
		if !ok {
			continue
		}
		t := Thread{Root: root, Replies: replies[rootID]}
		if root.Priority == "blocker" && root.Status == "open" {
			blockers = append(blockers, t)
		} else {
			normal = append(normal, t)
		}
	}
	return append(blockers, normal...)
}

func collectAutocomplete(agents []*store.Agent) (roles, repos []string) {
	sr, sp := map[string]bool{}, map[string]bool{}
	for _, a := range agents {
		if a.Role != "" && !sr[a.Role] {
			roles = append(roles, a.Role)
			sr[a.Role] = true
		}
		if a.Repo != "" && !sp[a.Repo] {
			repos = append(repos, a.Repo)
			sp[a.Repo] = true
		}
	}
	return
}

func (h *Handler) renderWith(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		slog.Error("template render error", "error", err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "error", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}
