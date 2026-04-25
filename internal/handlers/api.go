package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"strconv"
	"strings"
)

// JSONResponse is a helper for sending JSON
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ==================== PUBLIC API ====================

// APIListPostsHandler returns posts for public view
func APIListPostsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 10
	offset := (page - 1) * perPage

	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")
	search := r.URL.Query().Get("search")

	query := db.DB.Model(&models.Entry{}).Where("LOWER(status) = ?", "published")

	if category != "" {
		query = query.Joins("JOIN entry_categories ON entries.id = entry_categories.entry_id").
			Joins("JOIN categories ON categories.id = entry_categories.category_id").
			Where("categories.slug = ?", category)
	}

	if tag != "" {
		query = query.Joins("JOIN entry_tags ON entries.id = entry_tags.entry_id").
			Joins("JOIN tags ON tags.id = entry_tags.tag_id").
			Where("tags.slug = ?", tag)
	}

	if search != "" {
		search = "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(entry_title) LIKE ? OR LOWER(content) LIKE ?", search, search)
	}

	var total int64
	query.Count(&total)

	var entries []models.Entry
	query.Preload("Author").Preload("Categories").Preload("Tags").
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&entries)

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"posts":       entries,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": (total + int64(perPage) - 1) / int64(perPage),
	})
}

// APIGetPostHandler returns a single post by slug
func APIGetPostHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/v1/posts/")
	if slug == "" {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "slug required"})
		return
	}

	var entry models.Entry
	if err := db.DB.Preload("Author").Preload("Categories").Preload("Tags").
		Preload("Comments", "status = ?", "approved").
		Where("slug = ? AND status = ?", slug, "published").
		First(&entry).Error; err != nil {
		JSONResponse(w, http.StatusNotFound, map[string]string{"error": "post not found"})
		return
	}

	JSONResponse(w, http.StatusOK, entry)
}

// APIListCategoriesHandler returns all categories
func APIListCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	db.DB.Order("name").Find(&categories)
	JSONResponse(w, http.StatusOK, categories)
}

// APIListTagsHandler returns all tags
func APIListTagsHandler(w http.ResponseWriter, r *http.Request) {
	var tags []models.Tag
	db.DB.Order("name").Find(&tags)
	JSONResponse(w, http.StatusOK, tags)
}

// APISubmitCommentHandler handles comment creation via API
func APISubmitCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var input struct {
		PostID  uint   `json:"post_id"`
		Author  string `json:"author"`
		Email   string `json:"email"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
		return
	}

	if input.PostID == 0 || input.Author == "" || input.Content == "" {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "required fields missing"})
		return
	}

	comment := models.Comment{
		EntryID: input.PostID,
		Author:  input.Author,
		Email:   input.Email,
		Content: input.Content,
		Status:  "pending",
	}

	if err := db.DB.Create(&comment).Error; err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to save comment"})
		return
	}

	JSONResponse(w, http.StatusCreated, map[string]string{"message": "comment submitted for approval"})
}

// APIContactHandler handles contact form submissions
func APIContactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var input struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
		return
	}

	if input.Name == "" || input.Email == "" || input.Message == "" {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "all fields are required"})
		return
	}

	// In a real app, we might save this to a 'contacts' table or send an email.
	// For now, we'll just log it and return success.
	log.Printf("Contact Form Submission: %s <%s> - %s", input.Name, input.Email, input.Message)

	JSONResponse(w, http.StatusOK, map[string]string{"message": "message sent successfully"})
}

// APIGetSettingsHandler returns site settings
func APIGetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var settings []models.SiteSetting
	db.DB.Find(&settings)

	config := make(map[string]string)
	for _, s := range settings {
		config[s.Name] = s.Value
	}

	// Ensure defaults
	if _, ok := config["blog_name"]; !ok {
		config["blog_name"] = "ResCMS"
	}

	JSONResponse(w, http.StatusOK, config)
}

// APIGetSessionHandler returns current user status
func APIGetSessionHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.OptionalUser(r)
	if user == nil {
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"user":          user,
	})
}

// ==================== ADMIN API ====================

// APIAdminListPostsHandler returns all posts (including drafts)
func APIAdminListPostsHandler(w http.ResponseWriter, r *http.Request) {
	var entries []models.Entry
	db.DB.Preload("Author").Preload("Categories").Preload("Tags").Order("created_at DESC").Find(&entries)
	JSONResponse(w, http.StatusOK, entries)
}

// APIAdminSavePostHandler creates or updates a post
func APIAdminSavePostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID         uint     `json:"id"`
		Title      string   `json:"title"`
		Slug       string   `json:"slug"`
		Content    string   `json:"content"`
		Status     string   `json:"status"`
		Categories []uint   `json:"categories"`
		Tags       []uint   `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		JSONResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if input.ID > 0 {
		var entry models.Entry
		if err := db.DB.First(&entry, input.ID).Error; err != nil {
			JSONResponse(w, http.StatusNotFound, map[string]string{"error": "post not found"})
			return
		}
		entry.EntryTitle = input.Title
		entry.Slug = input.Slug
		entry.Content = input.Content
		entry.Status = input.Status
		if err := db.DB.Save(&entry).Error; err != nil {
			JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to update post"})
			return
		}
	} else {
		entry := models.Entry{
			AccountID:  user.UserID,
			EntryTitle: input.Title,
			Slug:       input.Slug,
			Content:    input.Content,
			Status:     input.Status,
		}
		if err := db.DB.Create(&entry).Error; err != nil {
			JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
			return
		}
		input.ID = entry.ID // For later use in associations
	}

	// Update associations using input.ID
	var entry models.Entry
	db.DB.First(&entry, input.ID)
	
	db.DB.Model(&entry).Association("Categories").Clear()
	if len(input.Categories) > 0 {
		var cats []models.Category
		for _, cid := range input.Categories {
			cats = append(cats, models.Category{ID: cid})
		}
		db.DB.Model(&entry).Association("Categories").Append(&cats)
	}

	db.DB.Model(&entry).Association("Tags").Clear()
	if len(input.Tags) > 0 {
		var tags []models.Tag
		for _, tid := range input.Tags {
			tags = append(tags, models.Tag{ID: tid})
		}
		db.DB.Model(&entry).Association("Tags").Append(&tags)
	}

	JSONResponse(w, http.StatusOK, entry)
}

// APIAdminDeletePostHandler deletes a post
func APIAdminDeletePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/posts/")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	db.DB.Delete(&models.Entry{}, id)
	JSONResponse(w, http.StatusOK, map[string]string{"message": "post deleted"})
}

// APIAdminListCommentsHandler returns all comments
func APIAdminListCommentsHandler(w http.ResponseWriter, r *http.Request) {
	var comments []models.Comment
	db.DB.Preload("Post").Order("created_at DESC").Find(&comments)
	JSONResponse(w, http.StatusOK, comments)
}

// APIAdminUpdateCommentStatusHandler updates comment status
func APIAdminUpdateCommentStatusHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/comments/")
	if strings.Contains(idStr, "/") {
		parts := strings.Split(idStr, "/")
		id, _ := strconv.ParseUint(parts[0], 10, 32)
		status := parts[1] // e.g. /api/admin/comments/5/approved
		db.DB.Model(&models.Comment{}).Where("id = ?", id).Update("status", status)
		JSONResponse(w, http.StatusOK, map[string]string{"message": "status updated"})
		return
	}
	JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
}

// APIAdminListStatsHandler returns dashboard stats
func APIAdminListStatsHandler(w http.ResponseWriter, r *http.Request) {
	var postCount, commentCount, userCount, categoryCount int64
	db.DB.Model(&models.Entry{}).Count(&postCount)
	db.DB.Model(&models.Comment{}).Count(&commentCount)
	db.DB.Model(&models.User{}).Count(&userCount)
	db.DB.Model(&models.Category{}).Count(&categoryCount)

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"posts":      postCount,
		"comments":   commentCount,
		"users":      userCount,
		"categories": categoryCount,
	})
}
