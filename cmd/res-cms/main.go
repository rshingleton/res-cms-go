/*
 * Copyright (C) 2026 Russ Shingleton <reshingleton@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"encoding/json"
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
	"res-cms-go/internal/models"
	"res-cms-go/internal/plugin"
	"res-cms-go/internal/session"
	"res-cms-go/internal/theme"
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
	if err := db.Init(cfg); err != nil {
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

	// Initialize Plugin Manager
	ctx := context.Background()
	registry := plugin.NewRegistry()
	handlers.PluginManager = plugin.NewManager(ctx, plugin.PluginsDir, registry)

	// Load enabled plugins from database
	var enabledPlugins []models.Plugin
	db.DB.Where("enabled = ?", true).Find(&enabledPlugins)
	var enabledSlugs []string
	for _, p := range enabledPlugins {
		enabledSlugs = append(enabledSlugs, p.Slug)
	}
	handlers.PluginManager.LoadAll(enabledSlugs)

	// Get active theme from settings or default
	activeTheme := "classic"
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "active_theme").Select("value").Scan(&activeTheme)
	if activeTheme == "" {
		activeTheme = "classic"
	}

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
		"toUpper":  func(s string) string { return strings.ToUpper(s) },
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	})
	if err != nil {
		log.Printf("Warning: Failed to load theme %s: %v", activeTheme, err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Static files (includes /static/uploads/ for uploaded images)
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve public directory
	mux.Handle("/js/", http.FileServer(http.Dir("public")))

	// Serve themes assets
	mux.Handle("/themes/", http.StripPrefix("/themes/", http.FileServer(http.Dir("themes"))))

	// Public routes
	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/page/{slug}", handlers.PageHandler)
	mux.HandleFunc("/post/{slug}", handlers.PostHandler)
	mux.HandleFunc("/comment/add", handlers.AddCommentHandler)
	mux.HandleFunc("GET /access/login", handlers.LoginFormHandler)
	mux.HandleFunc("POST /access/login", handlers.LoginHandler)
	mux.HandleFunc("/access/logout", handlers.LogoutHandler)
	mux.HandleFunc("/profile", handlers.ProfileHandler)
	mux.HandleFunc("/posts/account/{account}", handlers.PostsByAccountHandler)
	mux.HandleFunc("/posts/page/{page}", handlers.PostsByPageHandler)
	mux.HandleFunc("/posts/tag/{tag}", handlers.PostsByTagHandler)

	// API Routes (/api/v1/ prefix)
	mux.Handle("/api/v1/posts", middleware.APIAuth(http.HandlerFunc(handlers.APIListPostsHandler)))
	mux.Handle("/api/v1/posts/", middleware.APIAuth(http.HandlerFunc(handlers.APIGetPostHandler)))
	mux.Handle("/api/v1/pages", middleware.APIAuth(http.HandlerFunc(handlers.APIListPagesHandler)))
	mux.Handle("/api/v1/tags", middleware.APIAuth(http.HandlerFunc(handlers.APIListTagsHandler)))
	mux.Handle("/api/v1/comments/submit", middleware.APIAuth(http.HandlerFunc(handlers.APISubmitCommentHandler)))
	mux.Handle("/api/v1/contact", middleware.APIAuth(http.HandlerFunc(handlers.APIContactHandler)))
	mux.Handle("/api/v1/settings", middleware.APIAuth(http.HandlerFunc(handlers.APIGetSettingsHandler)))
	mux.Handle("/api/v1/session", middleware.APIAuth(http.HandlerFunc(handlers.APIGetSessionHandler)))

	// Register Plugin Route Hooks
	registry.ApplyRouteHooks(mux)

	// Admin API Routes (Protected)
	mux.Handle("/api/admin/posts", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminListPostsHandler)))
	mux.Handle("/api/admin/posts/save", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminSavePostHandler)))
	mux.Handle("/api/admin/posts/", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminDeletePostHandler)))
	mux.Handle("/api/admin/comments", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminListCommentsHandler)))
	mux.Handle("/api/admin/comments/", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminUpdateCommentStatusHandler)))
	mux.Handle("/api/admin/stats", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminListStatsHandler)))
	mux.Handle("/api/admin/pages/reorder", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIAdminReorderPagesHandler)))
	mux.Handle("/api/upload/image", middleware.AdminAPIAuth(http.HandlerFunc(handlers.APIUploadImageHandler)))

	// Admin routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/", handlers.AdminIndexHandler)
	adminMux.HandleFunc("/manage", handlers.AdminIndexHandler)
	adminMux.HandleFunc("/manage/profile", handlers.AdminProfileHandler)
	adminMux.HandleFunc("/manage/profile/update", handlers.AdminProfileUpdateHandler)
	adminMux.HandleFunc("/manage/posts", handlers.AdminListPostsHandler)
	adminMux.HandleFunc("/manage/posts/new", handlers.AdminAddPostFormHandler)
	adminMux.HandleFunc("/manage/posts/edit/{id}", handlers.AdminEditPostFormHandler)
	adminMux.HandleFunc("/manage/posts/update/{id}", handlers.AdminUpdatePostHandler)
	adminMux.HandleFunc("/manage/posts/delete/{id}", handlers.AdminDeletePostHandler)
	adminMux.HandleFunc("/manage/posts/publish/{id}", handlers.AdminPublishPostHandler)
	adminMux.HandleFunc("/manage/posts/draft/{id}", handlers.AdminDraftPostHandler)
	adminMux.HandleFunc("GET /manage/pages", handlers.AdminListPagesHandler)
	adminMux.HandleFunc("POST /manage/pages", handlers.AdminAddPageHandler)
	adminMux.HandleFunc("/manage/pages/edit/{id}", handlers.AdminEditPageFormHandler)
	adminMux.HandleFunc("/manage/pages/update/{id}", handlers.AdminUpdatePageHandler)
	adminMux.HandleFunc("/manage/pages/delete/{id}", handlers.AdminDeletePageHandler)
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
	adminMux.HandleFunc("/manage/editor", handlers.AdminUnifiedEditorHandler)
	adminMux.HandleFunc("/manage/editor/save", handlers.AdminUnifiedSaveHandler)
	adminMux.HandleFunc("/manage/themes/copy/{theme}", handlers.AdminThemeCopyHandler)
	adminMux.HandleFunc("/manage/themes/upload", handlers.AdminUploadThemeHandler)
	adminMux.HandleFunc("/manage/themes/activate/{name}", handlers.AdminActivateThemeHandler)
	adminMux.HandleFunc("/manage/themes/export/{name}", handlers.AdminExportThemeHandler)
	adminMux.HandleFunc("/manage/plugins", handlers.AdminListPluginsHandler)
	adminMux.HandleFunc("POST /manage/plugins/upload", handlers.AdminUploadPluginHandler)
	adminMux.HandleFunc("POST /manage/plugins/toggle/{slug}", handlers.AdminTogglePluginHandler)
	adminMux.HandleFunc("POST /manage/plugins/reload/{slug}", handlers.AdminReloadPluginHandler)
	adminMux.HandleFunc("POST /manage/plugins/delete/{slug}", handlers.AdminDeletePluginHandler)

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
	log.Println("  [Public] /post/{slug}")
	log.Println("  [Public] /access/login")
	log.Println("  [Admin]  /manage")
	log.Println("  [Admin]  /manage/posts")
	log.Println("  [Admin]  /manage/pages")
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
		"toUpper":    func(s string) string { return strings.ToUpper(s) },
		"safeHTML":   func(s string) template.HTML { return template.HTML(s) },
		"hasSuffix":  func(s, suffix string) bool { return strings.HasSuffix(s, suffix) },
		"hasPrefix":  func(s, prefix string) bool { return strings.HasPrefix(s, prefix) },
		"trimPrefix": func(s, prefix string) string { return strings.TrimPrefix(s, prefix) },
		"replace":    func(s, old, new string) string { return strings.ReplaceAll(s, old, new) },
		"title":      func(s string) string { return strings.Title(s) },
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
