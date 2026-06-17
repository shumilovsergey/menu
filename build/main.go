package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var buildTime = "unknown"

//go:embed web
var webFiles embed.FS

var (
	authURL      string
	authInternal string
	appURL       string
	appToken     string
	tmpl         *template.Template
	httpClient   = &http.Client{}
)

type pageData struct {
	User  *User
	Error string
	Apps  []appDef
}

// appDef is the single source of truth for an app in the grid.
// The grid (template), the status endpoint, and the open handler all derive from it.
type appDef struct {
	Slug string // url segment + status key, e.g. "blur"
	Name string // display name shown on the card
	URL  string // target base URL, e.g. "https://blur.sh-development.ru"
	Icon string // temp placeholder seed (first letter) — swap for real assets later
	Desc string // info-modal text
}

var apps = []appDef{
	{
		Slug: "wgetbash", Name: "wgetbash", URL: "https://wgetbash.sh-development.ru", Icon: "w",
		Desc: "Запуск bash-скриптов одной командой через wget.",
	},
	{
		Slug: "wgetbash", Name: "loger", URL: "https://wgetbash.sh-development.ru/?view=loger", Icon: "l",
		Desc: "Помощник для разбора логов. Чтобы в разгар инцидента не вспоминать команды, флаги и синтаксис.",
	},
	{
		Slug: "blur", Name: "blur", URL: "https://blur.sh-development.ru", Icon: "b",
		Desc: "Размытие лиц и чувствительных данных на изображениях.",
	},
	{
		Slug: "food-scaner", Name: "food scaner", URL: "https://food-scaner.sh-development.ru", Icon: "f",
		Desc: "Сканирование состава продуктов по фото этикетки.",
	},
}

// appBySlug returns the app with the given slug, or nil.
func appBySlug(slug string) *appDef {
	for i := range apps {
		if apps[i].Slug == slug {
			return &apps[i]
		}
	}
	return nil
}

// statusClient is used for short-timeout server-side reachability probes.
var statusClient = &http.Client{Timeout: 4 * time.Second}

// reachable reports whether the server can reach url (any HTTP response counts).
func reachable(url string) bool {
	resp, err := statusClient.Head(url)
	if err != nil {
		resp, err = statusClient.Get(url)
		if err != nil {
			return false
		}
	}
	resp.Body.Close()
	return true
}

func initTemplate() {
	src, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		log.Fatalf("web/index.html not found: %v", err)
	}
	tmpl = template.Must(template.New("index").Parse(string(src)))
}

// ── request logging ───────────────────────────────────────────────────────────

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(status int) {
	sw.status = status
	sw.ResponseWriter.WriteHeader(status)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

// ── app routes ────────────────────────────────────────────────────────────────

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if code := r.URL.Query().Get("code"); code != "" {
		handleCallback(w, r, code)
		return
	}
	var user *User
	if uid := sessionUserID(r); uid != 0 {
		user, _ = getUserByID(uid)
	}
	tmpl.Execute(w, pageData{User: user, Apps: apps}) //nolint:errcheck
}

// handleOpen issues a cross-app delegate redirect for /open/{slug}.
// The server re-checks reachability first, so a green status on the client is
// backed by the server half of the check at redirect time too.
func handleOpen(w http.ResponseWriter, r *http.Request) {
	uid := sessionUserID(r)
	if uid == 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	app := appBySlug(r.PathValue("slug"))
	if app == nil {
		http.NotFound(w, r)
		return
	}
	if !reachable(app.URL) {
		log.Printf("open-%s uid=%d unreachable", app.Slug, uid)
		http.Error(w, "app unreachable", http.StatusBadGateway)
		return
	}
	code, err := delegateCode(uid)
	if err != nil {
		log.Printf("open-%s uid=%d error=%v", app.Slug, uid, err)
		http.Error(w, "could not open app", http.StatusInternalServerError)
		return
	}
	log.Printf("open-%s uid=%d", app.Slug, uid)
	http.Redirect(w, r, app.URL+"/?code="+code, http.StatusFound)
}

// handleStatus returns server-side reachability of every app as {slug: bool}.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	uid := sessionUserID(r)
	if uid == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	result := make(map[string]bool, len(apps))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := range apps {
		wg.Add(1)
		go func(a appDef) {
			defer wg.Done()
			ok := reachable(a.URL)
			mu.Lock()
			result[a.Slug] = ok
			mu.Unlock()
		}(apps[i])
	}
	wg.Wait()
	log.Printf("status uid=%d result=%v", uid, result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result) //nolint:errcheck
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "--info") {
		fmt.Printf("menu built: %s\n", buildTime)
		os.Exit(0)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	godotenv.Load() //nolint:errcheck

	authURL = os.Getenv("AUTH_URL")
	authInternal = os.Getenv("AUTH_INTERNAL")
	appURL = os.Getenv("APP_URL")
	appToken = os.Getenv("APP_TOKEN")

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = "dev-secret"
	}
	jwtSecret = []byte(secretKey)

	initDB()
	initTemplate()

	webFS, _ := fs.Sub(webFiles, "web")
	fileServer := http.FileServer(http.FS(webFS))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /login", handleLogin)
	mux.HandleFunc("GET /logout", handleLogout)
	mux.HandleFunc("GET /open/{slug}", handleOpen)
	mux.HandleFunc("GET /status", handleStatus)
	mux.Handle("GET /favicon.svg", fileServer)
	mux.Handle("GET /background.webp", fileServer)
	mux.Handle("GET /shell.css", fileServer)
	mux.Handle("GET /shell.js", fileServer)
	mux.Handle("GET /app.css", fileServer)
	mux.Handle("GET /app.js", fileServer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8890"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, logMiddleware(mux)))
}
