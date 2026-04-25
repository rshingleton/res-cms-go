package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"res-cms-go/internal/config"
	"res-cms-go/internal/db"
	"res-cms-go/internal/handlers"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/session"
	"res-cms-go/internal/theme"
	"encoding/json"
	"strings"
)

var (
	listenAddr string
	configPath string
	production bool
)

func init() {
	flag.StringVar(&listenAddr, "listen", ":3009", "Address to listen on")
	flag.StringVar(&configPath, "config", "rescms.yml", "Path to config file")
	flag.BoolVar(&production, "production", false, "Run in production mode")
	flag.Parse()
}

func main() {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Warning: Could not load config file: %v, using defaults", err)
		cfg = &config.Config{
			Listen:    ":3009",
			SQLiteDSN: "data/rescms.db",
		}
	}

	// Override listen address if provided
	if listenAddr != ":3009" {
		cfg.Listen = listenAddr
	}

	// Initialize database
	dsn := cfg.SQLiteDSN
	if !strings.HasPrefix(dsn, "sqlite:") {
		dsn = "sqlite:" + dsn
	}
	if err := db.Init(dsn, production); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize session store
	if len(cfg.Secrets) == 0 {
		cfg.Secrets = []string{"mysecret"}
	}
	session.Init(cfg.Secrets)

	// Load templates
	if err := loadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Initialize Theme Engine
	handlers.ThemeEngine = theme.NewEngine("themes")
	
	// Get active theme from settings or default
	activeTheme := "classic"
	// TODO: Fetch from DB
	
	err = handlers.ThemeEngine.LoadTheme(activeTheme, template.FuncMap{
		"formatDate": func(t interface{}) string { return "2024-01-01" },
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"js": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"toUpper": func(s string) string { return strings.ToUpper(s) },
	})
	if err != nil {
		log.Printf("Warning: Failed to load theme %s: %v", activeTheme, err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve public directory
	mux.Handle("/js/", http.FileServer(http.Dir("public")))

	// Serve themes assets
	mux.Handle("/themes/", http.StripPrefix("/themes/", http.FileServer(http.Dir("themes"))))

	// Public routes
	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/page/{page}", handlers.IndexHandler)
	mux.HandleFunc("/entry/{slug}", handlers.PostHandler)
	mux.HandleFunc("/post/{slug}", handlers.PostHandler) // Support legacy path
	mux.HandleFunc("/comment/add", handlers.AddCommentHandler)
	mux.HandleFunc("GET /access/login", handlers.LoginFormHandler)
	mux.HandleFunc("POST /access/login", handlers.LoginHandler)
	mux.HandleFunc("/access/logout", handlers.LogoutHandler)
	mux.HandleFunc("/profile", handlers.ProfileHandler)
	mux.HandleFunc("/entries/account/{account}", handlers.EntriesByAccountHandler)
	mux.HandleFunc("/entries/category/{category}", handlers.PostsByCategoryHandler)
	mux.HandleFunc("/entries/tag/{tag}", handlers.PostsByTagHandler)

	// API Routes (/api/v1/ prefix)
	mux.HandleFunc("/api/v1/posts", handlers.APIListPostsHandler)
	mux.HandleFunc("/api/v1/posts/", handlers.APIGetPostHandler)
	mux.HandleFunc("/api/v1/categories", handlers.APIListCategoriesHandler)
	mux.HandleFunc("/api/v1/tags", handlers.APIListTagsHandler)
	mux.HandleFunc("/api/v1/comments/submit", handlers.APISubmitCommentHandler)
	mux.HandleFunc("/api/v1/contact", handlers.APIContactHandler)
	mux.HandleFunc("/api/v1/settings", handlers.APIGetSettingsHandler)
	mux.HandleFunc("/api/v1/session", handlers.APIGetSessionHandler)

	// Admin API Routes (Protected)
	mux.Handle("/api/admin/posts", middleware.Auth(http.HandlerFunc(handlers.APIAdminListPostsHandler)))
	mux.Handle("/api/admin/posts/save", middleware.Auth(http.HandlerFunc(handlers.APIAdminSavePostHandler)))
	mux.Handle("/api/admin/posts/", middleware.Auth(http.HandlerFunc(handlers.APIAdminDeletePostHandler)))
	mux.Handle("/api/admin/comments", middleware.Auth(http.HandlerFunc(handlers.APIAdminListCommentsHandler)))
	mux.Handle("/api/admin/comments/", middleware.Auth(http.HandlerFunc(handlers.APIAdminUpdateCommentStatusHandler)))
	mux.Handle("/api/admin/stats", middleware.Auth(http.HandlerFunc(handlers.APIAdminListStatsHandler)))

	// Admin routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/", handlers.AdminIndexHandler)
	adminMux.HandleFunc("/manage", handlers.AdminIndexHandler)
	adminMux.HandleFunc("/manage/profile", handlers.AdminProfileHandler)
	adminMux.HandleFunc("/manage/profile/update", handlers.AdminProfileUpdateHandler)
	adminMux.HandleFunc("/manage/entries", handlers.AdminListPostsHandler)
	adminMux.HandleFunc("/manage/entries/new", handlers.AdminAddPostFormHandler)
	adminMux.HandleFunc("/manage/entries/edit/{id}", handlers.AdminEditPostFormHandler)
	adminMux.HandleFunc("/manage/entries/update/{id}", handlers.AdminUpdatePostHandler)
	adminMux.HandleFunc("/manage/entries/delete/{id}", handlers.AdminDeletePostHandler)
	adminMux.HandleFunc("/manage/entries/publish/{id}", handlers.AdminPublishPostHandler)
	adminMux.HandleFunc("/manage/entries/draft/{id}", handlers.AdminDraftPostHandler)
	adminMux.HandleFunc("/manage/categories", handlers.AdminListCategoriesHandler)
	adminMux.HandleFunc("/manage/categories/new", handlers.AdminAddCategoryHandler)
	adminMux.HandleFunc("/manage/categories/edit/{id}", handlers.AdminEditCategoryFormHandler)
	adminMux.HandleFunc("/manage/categories/update/{id}", handlers.AdminUpdateCategoryHandler)
	adminMux.HandleFunc("/manage/categories/delete/{id}", handlers.AdminDeleteCategoryHandler)
	adminMux.HandleFunc("/manage/comments", handlers.AdminListCommentsHandler)
	adminMux.HandleFunc("/manage/comments/approve/{id}", handlers.AdminApproveCommentHandler)
	adminMux.HandleFunc("/manage/comments/unapprove/{id}", handlers.AdminUnapproveCommentHandler)
	adminMux.HandleFunc("/manage/comments/delete/{id}", handlers.AdminDeleteCommentHandler)
	adminMux.HandleFunc("/manage/accounts", handlers.AdminListUsersHandler)
	// /manage/accounts/new has no dedicated form page; the add-user form is embedded in the accounts list.
	adminMux.HandleFunc("/manage/accounts/edit/{id}", handlers.AdminEditUserFormHandler)
	adminMux.HandleFunc("/manage/accounts/update/{id}", handlers.AdminUpdateUserHandler)
	adminMux.HandleFunc("/manage/accounts/delete/{id}", handlers.AdminDeleteUserHandler)
	adminMux.HandleFunc("/manage/configs", handlers.AdminSettingsHandler)
	adminMux.HandleFunc("/manage/themes", handlers.AdminListThemesHandler)
	adminMux.HandleFunc("/manage/themes/upload", handlers.AdminUploadThemeHandler)
	adminMux.HandleFunc("/manage/themes/activate/{name}", handlers.AdminActivateThemeHandler)
	adminMux.HandleFunc("/manage/themes/export/{name}", handlers.AdminExportThemeHandler)

	// Wrap admin routes with auth middleware
	mux.Handle("/manage/", middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminMux.ServeHTTP(w, r)
	})))

	log.Printf("Starting ResCMS Go on %s", cfg.Listen)
	log.Printf("Production mode: %v", production)
	handlers.IsProduction = production

	// Route Trace
	log.Println("Registered Routes:")
	log.Println("  [Public] /")
	log.Println("  [Public] /entry/{slug}")
	log.Println("  [Public] /access/login")
	log.Println("  [Admin]  /manage")
	log.Println("  [Admin]  /manage/entries")
	log.Println("  [Admin]  /manage/categories")
	log.Println("  [Admin]  /manage/accounts")

	if err := http.ListenAndServe(cfg.Listen, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}

}

func loadTemplates() error {
	base := template.New("").Funcs(template.FuncMap{
		"formatDate": func(t interface{}) string { return "2024-01-01" },
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"js": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"toUpper": func(s string) string { return strings.ToUpper(s) },
	})

	// Load Admin Layout
	adminLayout := "internal/ui/admin/layouts/admin.html"
	if _, err := os.Stat(adminLayout); err == nil {
		content, _ := os.ReadFile(adminLayout)
		_, err = base.New("layouts/admin.html").Parse(string(content))
		if err != nil {
			return err
		}
	}

	// Load Admin Pages
	return filepath.Walk("internal/ui/admin/admin", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}

		name := filepath.ToSlash(strings.TrimPrefix(path, "internal/ui/admin/"))
		content, _ := os.ReadFile(path)

		t, err := base.Clone()
		if err != nil {
			return err
		}
		_, err = t.New(name).Parse(string(content))
		if err != nil {
			return err
		}

		if handlers.Templates == nil {
			handlers.Templates = make(map[string]*template.Template)
		}
		handlers.Templates[name] = t
		return nil
	})
}
