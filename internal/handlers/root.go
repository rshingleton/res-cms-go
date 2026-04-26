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

package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"res-cms-go/internal/session"
	"res-cms-go/internal/theme"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Templates holds individual page templates cloned from base
var Templates map[string]*template.Template

// ThemeEngine is the dynamic theme manager
var ThemeEngine *theme.Engine

// IsProduction indicates if we are in production mode
var IsProduction bool

// InitTemplates loads templates
func InitTemplates() error {
	// Implementation moved to main.go but keep the variable
	return nil
}

// IndexHandler handles the home page
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Support only "/" — pagination via ?page=N query param
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Get page number
	pageStr := strings.TrimPrefix(r.URL.Path, "/page/")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		// Fallback to query param if not in path
		page, _ = strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
	}
	perPage := 10

	// Query entries
	var entries []models.Post
	var total int64

	db.DB.Model(&models.Post{}).Where("status = ?", "published").Count(&total)

	offset := (page - 1) * perPage
	if err := db.DB.Preload("Author").Preload("Pages").Preload("Tags").
		Where("status = ?", "published").
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&entries).Error; err != nil {
		log.Printf("Error fetching entries: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	// Get layout style
	var layoutStyle string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "layout_style").Select("value").Scan(&layoutStyle)
	if layoutStyle == "" {
		layoutStyle = "default"
	}

	// Get system page for Home
	var systemPage models.Page
	db.DB.Where("is_system = ?", true).First(&systemPage)

	// Get sidebar data
	sidebar := getSidebarData()

	// Render template
	data := map[string]interface{}{
		"Posts":       entries,
		"BlogName":    blogName,
		"LayoutStyle": layoutStyle,
		"Sidebar":     sidebar,
		"Page":        page,
		"SystemPage":  systemPage,
		"TotalPages":  (total + int64(perPage) - 1) / int64(perPage),
		"User":        middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// PostHandler handles individual post pages
func PostHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	var slug string
	if strings.HasPrefix(path, "/entry/") {
		slug = strings.TrimPrefix(path, "/entry/")
	} else if strings.HasPrefix(path, "/post/") {
		slug = strings.TrimPrefix(path, "/post/")
	}

	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var entry models.Post
	if err := db.DB.Preload("Author").Preload("Pages").Preload("Tags").
		Preload("Comments", "status = ?", "approved").
		Preload("Comments.Post"). // Need to update Comment model next
		Where("slug = ? AND status = ?", slug, "published").
		First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
			return
		}
		log.Printf("Error fetching entry: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	// Get sidebar data
	sidebar := getSidebarData()

	data := map[string]interface{}{
		"Post":     entry,
		"BlogName": blogName,
		"Sidebar":  sidebar,
		"User":     middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/post.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// PageHandler renders a dynamic page by slug
func PageHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/page/")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	var page models.Page
	if err := db.DB.Where("slug = ?", slug).First(&page).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	data := map[string]interface{}{
		"Page":     page,
		"BlogName": blogName,
		"Sidebar":  getSidebarData(),
		"User":     middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/page.html", data); err != nil {
		log.Printf("Error rendering page template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AddCommentHandler handles comment submission
func AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	postIDStr := r.PostForm.Get("post_id")
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid post ID: %v", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	author := r.PostForm.Get("author")
	email := r.PostForm.Get("email")
	content := r.PostForm.Get("content")

	if author == "" || content == "" {
		middleware.GenerateFlashCookie(w, "Author and content are required")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	comment := models.Comment{
		PostID:  uint(postID),
		Author:  author,
		Email:   email,
		Content: content,
		Status:  "pending",
	}

	if err := db.DB.Create(&comment).Error; err != nil {
		log.Printf("Error creating comment: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to add comment")
	} else {
		middleware.GenerateFlashCookie(w, "Comment submitted successfully")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// PostsByPageHandler handles filtered posts by page (category)
func PostsByPageHandler(w http.ResponseWriter, r *http.Request) {
	pageSlug := strings.TrimPrefix(r.URL.Path, "/entries/page/")
	if pageSlug == "" {
		http.NotFound(w, r)
		return
	}

	var p models.Page
	if err := db.DB.Where("slug = ?", pageSlug).First(&p).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var entries []models.Post
	if err := db.DB.Joins("JOIN post_pages ON posts.id = post_pages.post_id").
		Where("post_pages.page_id = ? AND posts.status = ?", p.ID, "published").
		Order("posts.created_at DESC").
		Preload("Author").
		Find(&entries).Error; err != nil {
		log.Printf("Error fetching entries: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	data := map[string]interface{}{
		"Posts":    entries,
		"BlogName": blogName,
		"Page":     p,
		"User":     middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// PostsByTagHandler handles filtered posts by tag
func PostsByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/entries/tag/")
	if tag == "" {
		http.NotFound(w, r)
		return
	}

	var t models.Tag
	if err := db.DB.Where("slug = ?", tag).First(&t).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var entries []models.Post
	if err := db.DB.Joins("JOIN post_tags ON posts.id = post_tags.post_id").
		Where("post_tags.tag_id = ? AND posts.status = ?", t.ID, "published").
		Order("posts.created_at DESC").
		Preload("Author").
		Find(&entries).Error; err != nil {
		log.Printf("Error fetching entries: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	data := map[string]interface{}{
		"Posts":    entries,
		"BlogName": blogName,
		"Tag":      t,
		"User":     middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// LatestPostsAPIHandler returns latest posts as JSON
func LatestPostsAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var entries []models.Post
	if err := db.DB.Preload("Author").
		Where("status = ?", "published").
		Order("created_at DESC").
		Limit(5).
		Find(&entries).Error; err != nil {
		log.Printf("Error fetching entries: %v", err)
		w.Write([]byte(`[]`))
		return
	}

	// Simple JSON output
	if len(entries) > 0 {
		w.Write([]byte(`[{"title":"` + entries[0].Title + `"}]`))
	} else {
		w.Write([]byte(`[]`))
	}
}

// PostsByAccountHandler handles filtered posts by account
func PostsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	accountName := strings.TrimPrefix(r.URL.Path, "/posts/account/")
	if accountName == "" {
		http.NotFound(w, r)
		return
	}

	// Get page number
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 10

	// Find the account first
	var account models.User
	if err := db.DB.Where("username = ?", accountName).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
			return
		}
		log.Printf("Error fetching account: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var entries []models.Post
	var total int64

	db.DB.Model(&models.Post{}).Where("account_id = ? AND status = ?", account.ID, "published").Count(&total)

	offset := (page - 1) * perPage
	if err := db.DB.Preload("Author").Preload("Pages").Preload("Tags").
		Where("account_id = ? AND status = ?", account.ID, "published").
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&entries).Error; err != nil {
		log.Printf("Error fetching entries: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get blog name
	var blogName string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Select("value").Scan(&blogName)
	if blogName == "" {
		blogName = "ResCMS"
	}

	// Get sidebar data
	sidebar := getSidebarData()

	// Render template
	data := map[string]interface{}{
		"Posts":             entries,
		"BlogName":          blogName,
		"Sidebar":           sidebar,
		"Page":              page,
		"TotalPages":        (total + int64(perPage) - 1) / int64(perPage),
		"PostsForAccount":   accountName,
		"User":              middleware.OptionalUser(r),
	}

	if err := renderTemplate(w, r, "public/index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getSidebarData retrieves sidebar components
func getSidebarData() map[string]interface{} {
	// Fetch ALL pages ordered by sort_order then title for consistent navbar
	var pages []models.Page
	db.DB.Order("sort_order ASC, title ASC").Find(&pages)

	var recent []models.Post
	db.DB.Select("slug, title, created_at").
		Where("status = ?", "published").
		Order("created_at DESC").
		Limit(5).
		Find(&recent)

	var popular []models.Post
	db.DB.Raw(`
		SELECT p.slug, p.title, p.created_at, (SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) as cnt
		FROM posts p WHERE p.status = 'published'
		ORDER BY cnt DESC LIMIT 5
	`).Scan(&popular)

	var tags []models.Tag
	db.DB.Raw(`
		SELECT t.* FROM tags t
		JOIN (SELECT tag_id, COUNT(*) as cnt FROM post_tags GROUP BY tag_id) pt
		ON t.id = pt.tag_id
		ORDER BY pt.cnt DESC
	`).Scan(&tags)

	return map[string]interface{}{
		"Pages":   pages,
		"Recent":  recent,
		"Popular": popular,
		"Tags":    tags,
	}
}

// renderTemplate renders a template with layout
var renderTemplate = func(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) error {
	// Get session
	store := session.Get()
	cookie, err := r.Cookie("rescms_session")
	if err == nil {
		sess, err := store.Decode(cookie.Value)
		if err == nil {
			s, ok := store.Get(sess.ID)
			if ok {
				data["Session"] = s
				data["res_account_id"] = s.UserID
			}
		}
	}

	// Get flash from cookie
	data["Flash"] = middleware.GetFlashFromRequest(w, r)

	// Get all settings for injection
	var settings []models.SiteSetting
	db.DB.Find(&settings)
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Name] = s.Value
	}

	// Inject settings with res_ prefix
	for k, v := range settingsMap {
		data["res_"+k] = v
	}

	// Special handling for HTML/CSS/JS to avoid escaping and wrap in tags if enabled
	if settingsMap["custom_css_enabled"] == "1" && settingsMap["custom_css"] != "" {
		data["res_custom_css_style"] = template.HTML("<style>\n" + settingsMap["custom_css"] + "\n</style>")
	} else {
		data["res_custom_css_style"] = template.HTML("")
	}

	if settingsMap["custom_js_enabled"] == "1" && settingsMap["custom_js"] != "" {
		data["res_custom_js_script"] = template.HTML("<script>\n" + settingsMap["custom_js"] + "\n</script>")
	} else {
		data["res_custom_js_script"] = template.HTML("")
	}

	if settingsMap["custom_header_html_enabled"] == "1" {
		data["res_custom_header_html_parsed"] = template.HTML(settingsMap["custom_header_html"])
	} else {
		data["res_custom_header_html_parsed"] = template.HTML("")
	}

	if settingsMap["custom_footer_html_enabled"] == "1" {
		data["res_custom_footer_html_parsed"] = template.HTML(settingsMap["custom_footer_html"])
	} else {
		data["res_custom_footer_html_parsed"] = template.HTML("")
	}

	// Set default values
	if data["BlogName"] != nil {
		data["res_blog_name"] = data["BlogName"]
	} else if data["res_blog_name"] == nil {
		data["res_blog_name"] = settingsMap["blog_name"]
		if data["res_blog_name"] == "" {
			data["res_blog_name"] = "ResCMS"
		}
	}

	// If we have a theme engine and the template is in it, use theme-specific template
	if ThemeEngine != nil {
		if !IsProduction {
			ThemeEngine.Reload()
		}
		// Map public/index.html to index.html in theme
		themeTemplateName := strings.TrimPrefix(name, "public/")
		if t, ok := ThemeEngine.Templates[themeTemplateName]; ok {
			log.Printf("Executing theme template %s for %s", themeTemplateName, name)

			// Try to use the master layout as entry point
			entryPoint := themeTemplateName
			if t.Lookup("layouts/main.html") != nil {
				entryPoint = "layouts/main.html"
			}

			return t.ExecuteTemplate(w, entryPoint, data)
		}
	}

	// If the template name is in our map, it's a page that needs a layout
	if t, ok := Templates[name]; ok {
		layout := "layouts/main.html"
		if strings.HasPrefix(name, "admin/") {
			layout = "layouts/admin.html"
		}

		log.Printf("Executing template %s with layout %s", name, layout)
		err := t.ExecuteTemplate(w, layout, data)
		if err != nil {
			log.Printf("Template execution error: %v", err)
		}
		return err
	}

	// Fallback for direct template execution if needed
	return fmt.Errorf("template %s not found", name)
}
