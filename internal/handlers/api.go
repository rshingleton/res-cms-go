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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"strconv"
	"strings"
	"time"
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

	pageSlug := r.URL.Query().Get("page")
	tag := r.URL.Query().Get("tag")
	search := r.URL.Query().Get("search")

	query := db.DB.Model(&models.Post{}).Where("LOWER(status) = ?", "published")

	if pageSlug != "" {
		query = query.Joins("JOIN post_pages ON posts.id = post_pages.post_id").
			Joins("JOIN pages ON pages.id = post_pages.page_id").
			Where("pages.slug = ?", pageSlug)
	}

	if tag != "" {
		query = query.Joins("JOIN post_tags ON posts.id = post_tags.post_id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id").
			Where("tags.slug = ?", tag)
	}

	if search != "" {
		search = "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", search, search)
	}

	var total int64
	query.Count(&total)

	var entries []models.Post
	query.Preload("Author").Preload("Pages").Preload("Tags").
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

	var entry models.Post
	if err := db.DB.Preload("Author").Preload("Pages").Preload("Tags").
		Preload("Comments", "status = ?", "approved").
		Where("slug = ? AND status = ?", slug, "published").
		First(&entry).Error; err != nil {
		JSONResponse(w, http.StatusNotFound, map[string]string{"error": "post not found"})
		return
	}

	JSONResponse(w, http.StatusOK, entry)
}

// APIListPagesHandler returns all pages
func APIListPagesHandler(w http.ResponseWriter, r *http.Request) {
	var pages []models.Page
	db.DB.Order("title").Find(&pages)
	JSONResponse(w, http.StatusOK, pages)
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

	// Check if comments are allowed
	var post models.Post
	if err := db.DB.First(&post, input.PostID).Error; err != nil {
		JSONResponse(w, http.StatusNotFound, map[string]string{"error": "post not found"})
		return
	}

	var globalEnabled string
	db.DB.Model(&models.SiteSetting{}).Where("name = ?", "posts_comments_enabled").Select("value").Scan(&globalEnabled)
	if globalEnabled != "1" || !post.CommentsEnabled {
		JSONResponse(w, http.StatusForbidden, map[string]string{"error": "comments are disabled"})
		return
	}

	comment := models.Comment{
		PostID:  input.PostID,
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
	var entries []models.Post
	db.DB.Preload("Author").Preload("Pages").Preload("Tags").Order("created_at DESC").Find(&entries)
	JSONResponse(w, http.StatusOK, entries)
}

// APIAdminSavePostHandler creates or updates a post
func APIAdminSavePostHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID              uint   `json:"id"`
		Title           string `json:"title"`
		Slug            string `json:"slug"`
		Content         string `json:"content"`
		Status          string `json:"status"`
		CreatedAt       string `json:"created_at"`
		CommentsEnabled bool   `json:"comments_enabled"`
		Pages           []uint `json:"pages"`
		Tags            []uint `json:"tags"`
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
		var entry models.Post
		if err := db.DB.First(&entry, input.ID).Error; err != nil {
			JSONResponse(w, http.StatusNotFound, map[string]string{"error": "post not found"})
			return
		}
		entry.Title = input.Title
		entry.Slug = input.Slug
		entry.Content = input.Content
		entry.Status = input.Status
		entry.CommentsEnabled = input.CommentsEnabled
		if input.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, input.CreatedAt); err == nil {
				entry.CreatedAt = t
			}
		}
		if err := db.DB.Save(&entry).Error; err != nil {
			JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to update post"})
			return
		}
	} else {
		entry := models.Post{
			AccountID:       user.UserID,
			Title:           input.Title,
			Slug:            input.Slug,
			Content:         input.Content,
			Status:          input.Status,
			CommentsEnabled: input.CommentsEnabled,
		}
		if input.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, input.CreatedAt); err == nil {
				entry.CreatedAt = t
			}
		}
		if err := db.DB.Create(&entry).Error; err != nil {
			JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
			return
		}
		input.ID = entry.ID // For later use in associations
	}

	// Update associations using input.ID
	var entry models.Post
	db.DB.First(&entry, input.ID)

	db.DB.Model(&entry).Association("Pages").Clear()
	if len(input.Pages) > 0 {
		var pages []models.Page
		for _, pid := range input.Pages {
			pages = append(pages, models.Page{ID: pid})
		}
		db.DB.Model(&entry).Association("Pages").Append(&pages)
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
	db.DB.Delete(&models.Post{}, id)
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
	var postCount, commentCount, userCount, pageCount int64
	db.DB.Model(&models.Post{}).Count(&postCount)
	db.DB.Model(&models.Comment{}).Count(&commentCount)
	db.DB.Model(&models.User{}).Count(&userCount)
	db.DB.Model(&models.Page{}).Count(&pageCount)

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"posts":    postCount,
		"comments": commentCount,
		"users":    userCount,
		"pages":    pageCount,
	})
}

// APIAdminReorderPagesHandler saves drag-drop sort order for pages
func APIAdminReorderPagesHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		JSONResponse(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var order []struct {
		ID        uint `json:"id"`
		SortOrder int  `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	for _, item := range order {
		db.DB.Model(&models.Page{}).Where("id = ?", item.ID).Update("sort_order", item.SortOrder)
	}

	JSONResponse(w, http.StatusOK, map[string]string{"message": "order saved"})
}

// APIUploadImageHandler handles image uploads from the rich text editor
func APIUploadImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		JSONResponse(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	// Max 10MB
	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("upload")
	if err != nil {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	// Validate extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowed[ext] {
		JSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid file type"})
		return
	}

	// Ensure uploads dir exists
	uploadsDir := "public/uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to create uploads dir"})
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(uploadsDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		JSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to write file"})
		return
	}

	url := "/static/uploads/" + filename
	// CKEditor 5 expects: { "url": "..." }
	JSONResponse(w, http.StatusOK, map[string]string{"url": url})
}
