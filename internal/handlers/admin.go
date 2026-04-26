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
	"html/template"
	"log"
	"net/http"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"io"
	"os"
	"path/filepath"
)

// AdminIndexHandler displays admin dashboard
func AdminIndexHandler(w http.ResponseWriter, r *http.Request) {
	// Only serve the dashboard for exact /manage or /manage/ paths.
	// Any other /manage/* sub-path that falls through to this handler should 404.
	if r.URL.Path != "/manage" && r.URL.Path != "/manage/" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// Get counts
	var postCount, commentCount, userCount, pageCount int64
	db.DB.Model(&models.Post{}).Count(&postCount)
	db.DB.Model(&models.Comment{}).Count(&commentCount)
	db.DB.Model(&models.User{}).Count(&userCount)
	db.DB.Model(&models.Page{}).Where("is_system = ?", false).Count(&pageCount)

	// Get recent posts
	var recentPosts []models.Post
	db.DB.Preload("Author").Order("created_at DESC").Limit(5).Find(&recentPosts)

	// Get pending comments
	var pendingComments []models.Comment
	db.DB.Preload("Post").Where("status = ?", "pending").Order("created_at DESC").Limit(5).Find(&pendingComments)

	data := map[string]interface{}{
		"BlogName":        getBlogName(),
		"User":            user,
		"PostCount":       postCount,
		"CommentCount":    commentCount,
		"UserCount":       userCount,
		"PageCount":       pageCount,
		"RecentPosts":     recentPosts,
		"PendingComments": pendingComments,
		"ActiveTab":       "dashboard",
	}

	if err := renderTemplate(w, r, "admin/index.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ==================== POSTS ====================

// AdminListPostsHandler lists all posts
func AdminListPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/posts" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var entries []models.Post
	db.DB.Preload("Author").Order("created_at DESC").Find(&entries)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Posts":     entries,
		"ActiveTab": "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddPostFormHandler shows add post form
func AdminAddPostFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/posts/new" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var pages []models.Page
	db.DB.Where("is_system = ?", false).Find(&pages)

	var tags []models.Tag
	db.DB.Find(&tags)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Pages":     pages,
		"Tags":      tags,
		"Post":      models.Post{},
		"IsNew":     true,
		"ActiveTab": "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddPostHandler creates a new post
func AdminAddPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/posts" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/posts", http.StatusFound)
		return
	}

	title := r.PostForm.Get("title")
	slug := r.PostForm.Get("slug")
	content := r.PostForm.Get("content")
	status := r.PostForm.Get("status")
	pageIDs := r.PostForm["pages"]
	tagIDs := r.PostForm["tags"]
	commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

	if title == "" || slug == "" {
		middleware.GenerateFlashCookie(w, "Title and slug are required")
		http.Redirect(w, r, "/manage/posts/new", http.StatusFound)
		return
	}

	post := models.Post{
		AccountID:  user.UserID,
		Title: title,
		Slug:       slug,
		Content:         content,
		Status:          status,
		CommentsEnabled: commentsEnabled,
	}

	if err := db.DB.Create(&post).Error; err != nil {
		log.Printf("Error creating post: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create post")
		http.Redirect(w, r, "/manage/posts/new", http.StatusFound)
		return
	}

	// Associate pages (acting as categories)
	if len(pageIDs) > 0 {
		var pages []models.Page
		for _, id := range pageIDs {
			if pid, err := strconv.ParseUint(id, 10, 32); err == nil {
				pages = append(pages, models.Page{ID: uint(pid)})
			}
		}
		db.DB.Model(&post).Association("Pages").Append(&pages)
	}

	// Associate tags
	if len(tagIDs) > 0 {
		var tags []models.Tag
		for _, id := range tagIDs {
			if tid, err := strconv.ParseUint(id, 10, 32); err == nil {
				tags = append(tags, models.Tag{ID: uint(tid)})
			}
		}
		db.DB.Model(&post).Association("Tags").Append(&tags)
	}

	middleware.GenerateFlashCookie(w, "Post created successfully")
	http.Redirect(w, r, "/manage/posts", http.StatusFound)
}

// AdminEditPostFormHandler shows edit post form
func AdminEditPostFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/posts/edit/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var post models.Post
	if err := db.DB.Preload("Pages").Preload("Tags").First(&post, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var pages []models.Page
	db.DB.Where("is_system = ?", false).Find(&pages)

	var tags []models.Tag
	db.DB.Find(&tags)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Post":     post,
		"Pages":     pages,
		"Tags":      tags,
		"IsNew":     false,
		"ActiveTab": "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUpdatePostHandler updates a post
func AdminUpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/posts/update/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/posts", http.StatusFound)
		return
	}

	title := r.PostForm.Get("title")
	slug := r.PostForm.Get("slug")
	content := r.PostForm.Get("content")
	status := r.PostForm.Get("status")
	pageIDs := r.PostForm["pages"]
	tagIDs := r.PostForm["tags"]
	commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

	updates := map[string]interface{}{
		"title":   title,
		"slug":    slug,
		"content":          content,
		"status":           status,
		"comments_enabled": commentsEnabled,
	}

	if err := db.DB.Model(&models.Post{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.Printf("Error updating post %d: %v", id, err)
		middleware.GenerateFlashCookie(w, "Failed to update post")
		http.Redirect(w, r, "/manage/posts/edit/"+idStr, http.StatusFound)
		return
	}

	// Update pages
	var post models.Post
	db.DB.First(&post, id)
	if len(pageIDs) > 0 {
		db.DB.Model(&post).Association("Pages").Clear()
		var pages []models.Page
		for _, id := range pageIDs {
			if pid, err := strconv.ParseUint(id, 10, 32); err == nil {
				pages = append(pages, models.Page{ID: uint(pid)})
			}
		}
		db.DB.Model(&post).Association("Pages").Append(&pages)
	}

	// Update tags
	if len(tagIDs) > 0 {
		db.DB.Model(&post).Association("Tags").Clear()
		var tags []models.Tag
		for _, id := range tagIDs {
			if tid, err := strconv.ParseUint(id, 10, 32); err == nil {
				tags = append(tags, models.Tag{ID: uint(tid)})
			}
		}
		db.DB.Model(&post).Association("Tags").Append(&tags)
	}

	middleware.GenerateFlashCookie(w, "Post updated successfully")
	http.Redirect(w, r, "/manage/posts", http.StatusFound)
}

// AdminDeletePostHandler deletes a post
func AdminDeletePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/posts/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Delete(&models.Post{}, id).Error; err != nil {
		log.Printf("Error deleting post: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete post")
	} else {
		middleware.GenerateFlashCookie(w, "Post deleted successfully")
	}

	http.Redirect(w, r, "/manage/posts", http.StatusFound)
}

// AdminPublishPostHandler publishes a post
func AdminPublishPostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/posts/publish/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Post{}).Where("id = ?", id).Update("status", "published").Error; err != nil {
		log.Printf("Error publishing post: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to publish post")
	} else {
		middleware.GenerateFlashCookie(w, "Post published successfully")
	}

	http.Redirect(w, r, "/manage/posts", http.StatusFound)
}

// AdminDraftPostHandler sets post to draft
func AdminDraftPostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/posts/draft/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Post{}).Where("id = ?", id).Update("status", "draft").Error; err != nil {
		log.Printf("Error setting draft: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to set draft")
	} else {
		middleware.GenerateFlashCookie(w, "Post set to draft")
	}

	http.Redirect(w, r, "/manage/posts", http.StatusFound)
}

// ==================== PAGES / TAXONOMY ====================

// AdminListPagesHandler lists all pages (taxonomy)
func AdminListPagesHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var pages []models.Page
	db.DB.Order("is_system DESC, title ASC").Find(&pages)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Pages":     pages,
		"Layouts":   getAvailableLayouts(),
		"ActiveTab": "pages",
	}

	if err := renderTemplate(w, r, "admin/pages/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddPageHandler creates a new page
func AdminAddPageHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/pages", http.StatusFound)
		return
	}

	title := r.PostForm.Get("name")
	slug := r.PostForm.Get("slug")
	layout := r.PostForm.Get("layout")
	commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

	if title == "" || slug == "" {
		middleware.GenerateFlashCookie(w, "Title and slug are required")
		http.Redirect(w, r, "/manage/pages", http.StatusFound)
		return
	}

	page := models.Page{
		Title:    title,
		Slug:            slug,
		Layout:          layout,
		CommentsEnabled: commentsEnabled,
		IsSystem:        false,
	}

	if err := db.DB.Create(&page).Error; err != nil {
		log.Printf("Error creating page: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create page")
	} else {
		middleware.GenerateFlashCookie(w, "Page '"+title+"' created successfully")
	}

	http.Redirect(w, r, "/manage/pages", http.StatusFound)
}

// AdminEditPageFormHandler shows edit page form
func AdminEditPageFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = strings.TrimPrefix(r.URL.Path, "/manage/pages/edit/")
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Printf("Error parsing page ID '%s': %v", idStr, err)
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var page models.Page
	if err := db.DB.First(&page, id).Error; err != nil {
		log.Printf("Page with ID %d not found", id)
		http.NotFound(w, r)
		return
	}

	log.Printf("Editing page ID %d: '%s' (Content length: %d)", id, page.Title, len(page.Content))

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Page":      page,
		"Layouts":   getAvailableLayouts(),
		"ActiveTab": "pages",
	}

	if err := renderTemplate(w, r, "admin/pages/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUpdatePageHandler updates a page
func AdminUpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = strings.TrimPrefix(r.URL.Path, "/manage/pages/update/")
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		log.Printf("Error parsing update ID '%s': %v", idStr, err)
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/pages", http.StatusFound)
		return
	}

	title := r.PostForm.Get("name")
	slug := r.PostForm.Get("slug")
	content := r.PostForm.Get("content")
	layout := r.PostForm.Get("layout")
	commentsEnabled := r.PostForm.Get("comments_enabled") == "on"

	log.Printf("Updating page ID %d: title='%s', slug='%s', content_len=%d", id, title, slug, len(content))

	updates := map[string]interface{}{
		"title":   title,
		"slug":    slug,
		"content":          content,
		"layout":           layout,
		"comments_enabled": commentsEnabled,
	}

	if err := db.DB.Model(&models.Page{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.Printf("Error updating page %d: %v", id, err)
		middleware.GenerateFlashCookie(w, "Failed to update page")
	} else {
		middleware.GenerateFlashCookie(w, "Page updated successfully")
	}

	http.Redirect(w, r, "/manage/pages", http.StatusFound)
}

// AdminDeletePageHandler deletes a page
func AdminDeletePageHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/pages/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Prevent deletion of system pages
	var page models.Page
	if err := db.DB.First(&page, id).Error; err == nil && page.IsSystem {
		middleware.GenerateFlashCookie(w, "Cannot delete system pages")
		http.Redirect(w, r, "/manage/pages", http.StatusFound)
		return
	}

	if err := db.DB.Delete(&models.Page{}, id).Error; err != nil {
		log.Printf("Error deleting page: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete page")
	} else {
		middleware.GenerateFlashCookie(w, "Page deleted successfully")
	}

	http.Redirect(w, r, "/manage/pages", http.StatusFound)
}

// ==================== COMMENTS ====================

// AdminListCommentsHandler lists all comments
func AdminListCommentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/comments" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var comments []models.Comment
	db.DB.Preload("Post").Order("created_at DESC").Find(&comments)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Comments":  comments,
		"ActiveTab": "comments",
	}

	if err := renderTemplate(w, r, "admin/comments/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminApproveCommentHandler approves a comment
func AdminApproveCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/comments/approve/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Comment{}).Where("id = ?", id).Update("status", "approved").Error; err != nil {
		log.Printf("Error approving comment: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to approve comment")
	} else {
		middleware.GenerateFlashCookie(w, "Comment approved")
	}

	http.Redirect(w, r, "/manage/comments", http.StatusFound)
}

// AdminUnapproveCommentHandler unapproves a comment
func AdminUnapproveCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/comments/unapprove/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Comment{}).Where("id = ?", id).Update("status", "pending").Error; err != nil {
		log.Printf("Error unapproving comment: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to unapprove comment")
	} else {
		middleware.GenerateFlashCookie(w, "Comment unapproved")
	}

	http.Redirect(w, r, "/manage/comments", http.StatusFound)
}

// AdminDeleteCommentHandler deletes a comment
func AdminDeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/comments/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Delete(&models.Comment{}, id).Error; err != nil {
		log.Printf("Error deleting comment: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete comment")
	} else {
		middleware.GenerateFlashCookie(w, "Comment deleted successfully")
	}

	http.Redirect(w, r, "/manage/comments", http.StatusFound)
}

// ==================== USERS ====================

// AdminListUsersHandler lists all users
func AdminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/accounts" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var users []models.User
	db.DB.Order("created_at DESC").Find(&users)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Users":     users,
		"ActiveTab": "users",
	}

	if err := renderTemplate(w, r, "admin/users/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddUserHandler creates a new user
func AdminAddUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/accounts" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/accounts", http.StatusFound)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	email := r.PostForm.Get("email")
	role := r.PostForm.Get("role")
	isAdmin := r.PostForm.Get("is_admin") == "on"

	if username == "" || password == "" {
		middleware.GenerateFlashCookie(w, "Username and password are required")
		http.Redirect(w, r, "/manage/accounts", http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create user")
		http.Redirect(w, r, "/manage/accounts", http.StatusFound)
		return
	}

	newUser := models.User{
		Username: username,
		Password: string(hash),
		Email:    email,
		Role:     role,
		IsAdmin:  isAdmin,
		Status:   "activated",
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create user")
	} else {
		middleware.GenerateFlashCookie(w, "User created successfully")
	}

	http.Redirect(w, r, "/manage/accounts", http.StatusFound)
}

// AdminEditUserFormHandler shows edit user form
func AdminEditUserFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/accounts/edit/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var editUser models.User
	if err := db.DB.First(&editUser, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	// Don't show password
	editUser.Password = ""

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"EditUser":  editUser,
		"ActiveTab": "users",
	}

	if err := renderTemplate(w, r, "admin/users/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUpdateUserHandler updates a user
func AdminUpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/accounts/update/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/manage/accounts", http.StatusFound)
		return
	}

	email := r.PostForm.Get("email")
	role := r.PostForm.Get("role")
	status := r.PostForm.Get("status")
	password := r.PostForm.Get("password")
	isAdmin := r.PostForm.Get("is_admin") == "on"

	updates := map[string]interface{}{
		"email":    email,
		"role":     role,
		"status":   status,
		"is_admin": isAdmin,
	}

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			middleware.GenerateFlashCookie(w, "Failed to update user")
			http.Redirect(w, r, "/manage/accounts/edit/"+idStr, http.StatusFound)
			return
		}
		updates["password"] = string(hash)
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.Printf("Error updating user: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to update user")
	} else {
		middleware.GenerateFlashCookie(w, "User updated successfully")
	}

	http.Redirect(w, r, "/manage/accounts", http.StatusFound)
}

// AdminDeleteUserHandler deletes a user
func AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/accounts/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Prevent deleting yourself
	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin || user.UserID == uint(id) {
		middleware.GenerateFlashCookie(w, "Cannot delete your own account")
		http.Redirect(w, r, "/manage/accounts", http.StatusFound)
		return
	}

	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		log.Printf("Error deleting user: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete user")
	} else {
		middleware.GenerateFlashCookie(w, "User deleted successfully")
	}

	http.Redirect(w, r, "/manage/accounts", http.StatusFound)
}

// ==================== SETTINGS ====================

// AdminSettingsHandler shows and updates settings
func AdminSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/configs" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			log.Printf("Error parsing form: %v", err)
			http.Redirect(w, r, "/manage/configs", http.StatusFound)
			return
		}

		settingKeys := []string{
			"blog_name", "tagline", "ga_id", "meta_desc",
			"custom_css", "custom_js", "custom_header_html", "custom_footer_html",
		}

		for _, key := range settingKeys {
			if r.PostForm.Has(key) {
				val := r.PostForm.Get(key)
				setting := models.SiteSetting{Name: key}
				db.DB.Where(models.SiteSetting{Name: key}).FirstOrCreate(&setting)
				db.DB.Model(&setting).Update("value", val)
			}
		}

		// Handle checkboxes (toggles) which might be missing if unchecked
		toggles := []string{"posts_comments_enabled", "pages_comments_enabled"}
		for _, key := range toggles {
			val := "0"
			if r.PostForm.Get(key) == "on" || r.PostForm.Get(key) == "1" {
				val = "1"
			}
			setting := models.SiteSetting{Name: key}
			db.DB.Where(models.SiteSetting{Name: key}).FirstOrCreate(&setting)
			db.DB.Model(&setting).Update("value", val)
		}

		// Handle AJAX request from Monaco editor or normal form
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		middleware.GenerateFlashCookie(w, "Settings saved successfully")
		http.Redirect(w, r, "/manage/configs", http.StatusFound)
		return
	}

	var settings []models.SiteSetting
	db.DB.Find(&settings)

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Name] = s.Value
	}

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Settings":  settingsMap,
		"ActiveTab": "settings",
	}

	if err := renderTemplate(w, r, "admin/settings.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminProfileHandler shows admin profile
func AdminProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/profile" {
		http.NotFound(w, r)
		return
	}

	profileView(w, r)
}

// AdminProfileUpdateHandler updates admin profile
func AdminProfileUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/profile/update" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	profileUpdate(w, r)
}

// AdminListThemesHandler lists available themes
func AdminListThemesHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	themes, err := os.ReadDir(ThemeEngine.ThemesPath)
	if err != nil {
		log.Printf("Error reading themes dir: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	type ThemeInfo struct {
		Name   string
		Active bool
	}
	var themeList []ThemeInfo
	for _, t := range themes {
		if t.IsDir() {
			themeList = append(themeList, ThemeInfo{
				Name:   t.Name(),
				Active: t.Name() == ThemeEngine.Active,
			})
		}
	}

	// Fetch settings for customization tab
	var settings []models.SiteSetting
	db.DB.Find(&settings)
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Name] = s.Value
	}

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Themes":    themeList,
		"Settings":  settingsMap,
		"ActiveTab": "themes",
	}

	if err := renderTemplate(w, r, "admin/themes/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUploadThemeHandler handles theme zip upload
func AdminUploadThemeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	file, _, err := r.FormFile("theme_zip")
	if err != nil {
		middleware.GenerateFlashCookie(w, "Failed to upload file")
		http.Redirect(w, r, "/manage/themes", http.StatusFound)
		return
	}
	defer file.Close()

	// Save to temp
	tmpFile, err := os.CreateTemp("", "theme-*.zip")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	themeName, err := ThemeEngine.ExtractTheme(tmpFile.Name())
	if err != nil {
		middleware.GenerateFlashCookie(w, "Invalid theme: "+err.Error())
	} else {
		middleware.GenerateFlashCookie(w, "Theme "+themeName+" installed successfully")
	}

	http.Redirect(w, r, "/manage/themes", http.StatusFound)
}

// AdminExportThemeHandler handles theme export
func AdminExportThemeHandler(w http.ResponseWriter, r *http.Request) {
	themeName := strings.TrimPrefix(r.URL.Path, "/manage/themes/export/")
	if themeName == "" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+themeName+".zip")

	if err := ThemeEngine.ExportTheme(themeName, w); err != nil {
		log.Printf("Export error: %v", err)
		// Can't set header anymore
	}
}

// AdminActivateThemeHandler switches the active theme
func AdminActivateThemeHandler(w http.ResponseWriter, r *http.Request) {
	themeName := strings.TrimPrefix(r.URL.Path, "/manage/themes/activate/")
	if themeName == "" {
		http.NotFound(w, r)
		return
	}

	err := ThemeEngine.LoadTheme(themeName, template.FuncMap{
		"formatDate": func(t interface{}) string { return "2024-01-01" },
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"js": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"toUpper":   func(s string) string { return strings.ToUpper(s) },
		"safeHTML":  func(s string) template.HTML { return template.HTML(s) },
		"hasSuffix": func(s, suffix string) bool { return strings.HasSuffix(s, suffix) },
	})

	if err == nil {
		// Upsert the active_theme setting
		setting := models.SiteSetting{Name: "active_theme", Value: themeName}
		db.DB.Where(models.SiteSetting{Name: "active_theme"}).FirstOrCreate(&setting)
		db.DB.Model(&setting).Update("value", themeName)
	}

	if err != nil {
		middleware.GenerateFlashCookie(w, "Failed to activate theme: "+err.Error())
	} else {
		middleware.GenerateFlashCookie(w, "Theme "+themeName+" activated")
	}

	http.Redirect(w, r, "/manage/themes", http.StatusFound)
}

// AdminUnifiedEditorHandler shows the unified Super Editor
func AdminUnifiedEditorHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil || !user.IsAdmin {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// 1. List all themes and their files
	type ThemeFiles struct {
		Name  string
		Files []string
	}
	var themeLibrary []ThemeFiles

	themes, _ := os.ReadDir(ThemeEngine.ThemesPath)
	for _, t := range themes {
		if t.IsDir() {
			var files []string
			themePath := filepath.Join(ThemeEngine.ThemesPath, t.Name())
			filepath.Walk(themePath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(themePath, path)
				ext := strings.ToLower(filepath.Ext(rel))
				if ext == ".html" || ext == ".js" || ext == ".css" || ext == ".json" || ext == ".scss" {
					files = append(files, rel)
				}
				return nil
			})
			themeLibrary = append(themeLibrary, ThemeFiles{Name: t.Name(), Files: files})
		}
	}

	// 2. Define global injectables
	type Injectable struct {
		Key     string
		Name    string
		Type    string
		Enabled bool
		Help    string
	}
	injectables := []Injectable{
		{Key: "custom_css", Name: "Custom CSS", Type: "css", Help: "Raw CSS injected into the <head>. No <style> tags needed."},
		{Key: "custom_js", Name: "Header JS", Type: "js", Help: "Raw JavaScript injected into the <head>. No <script> tags needed."},
		{Key: "custom_header_html", Name: "Header HTML", Type: "html", Help: "Raw HTML injected at the bottom of the <head>."},
		{Key: "custom_footer_html", Name: "Footer HTML", Type: "html", Help: "Raw HTML injected at the bottom of the <body>."},
	}

	// 3. Fetch settings for injectables status
	var settings []models.SiteSetting
	db.DB.Find(&settings)
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Name] = s.Value
	}

	for i, inj := range injectables {
		if settingsMap[inj.Key+"_enabled"] == "1" {
			injectables[i].Enabled = true
		}
	}

	// 4. Handle selected target
	target := r.URL.Query().Get("target") // global:custom_css or theme:classic:layouts/main.html
	content := ""
	targetType := ""
	targetEnabled := false

	if strings.HasPrefix(target, "global:") {
		key := strings.TrimPrefix(target, "global:")
		content = settingsMap[key]
		targetType = "global"
		targetEnabled = settingsMap[key+"_enabled"] == "1"
	} else if strings.HasPrefix(target, "theme:") {
		parts := strings.SplitN(strings.TrimPrefix(target, "theme:"), ":", 2)
		if len(parts) == 2 {
			themeName, fileName := parts[0], parts[1]
			fullPath := filepath.Join(ThemeEngine.ThemesPath, themeName, fileName)
			b, err := os.ReadFile(fullPath)
			if err == nil {
				content = string(b)
			}
			targetType = "file"
		}
	}

	data := map[string]interface{}{
		"BlogName":       getBlogName(),
		"User":           user,
		"ThemeLibrary":   themeLibrary,
		"Injectables":    injectables,
		"SelectedTarget": target,
		"FileContent":    content,
		"TargetType":     targetType,
		"TargetEnabled":  targetEnabled,
		"ActiveTab":      "themes",
	}

	if err := renderTemplate(w, r, "admin/themes/editor.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUnifiedSaveHandler saves either a theme file or a global setting
func AdminUnifiedSaveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	target := r.FormValue("target")
	content := r.FormValue("content")
	enabled := r.FormValue("enabled") // "1" or "0"

	if target == "" {
		http.Error(w, "Missing target", http.StatusBadRequest)
		return
	}

	if strings.HasPrefix(target, "global:") {
		key := strings.TrimPrefix(target, "global:")
		// Save content
		setting := models.SiteSetting{Name: key}
		db.DB.Where(models.SiteSetting{Name: key}).FirstOrCreate(&setting)
		db.DB.Model(&setting).Update("value", content)

		// Save enabled status
		statusKey := key + "_enabled"
		statusSetting := models.SiteSetting{Name: statusKey}
		db.DB.Where(models.SiteSetting{Name: statusKey}).FirstOrCreate(&statusSetting)
		db.DB.Model(&statusSetting).Update("value", enabled)
	} else if strings.HasPrefix(target, "theme:") {
		parts := strings.SplitN(strings.TrimPrefix(target, "theme:"), ":", 2)
		if len(parts) == 2 {
			themeName, fileName := parts[0], parts[1]
			fullPath := filepath.Join(ThemeEngine.ThemesPath, themeName, fileName)
			cleanPath := filepath.Clean(fullPath)
			if !strings.HasPrefix(cleanPath, filepath.Join(ThemeEngine.ThemesPath, themeName)) {
				http.Error(w, "Invalid file path", http.StatusForbidden)
				return
			}
			if err := os.WriteFile(cleanPath, []byte(content), 0644); err != nil {
				log.Printf("Write error: %v", err)
				http.Error(w, "Failed to save file", http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// AdminThemeCopyHandler duplicates a theme
func AdminThemeCopyHandler(w http.ResponseWriter, r *http.Request) {
	themeName := r.PathValue("theme")
	if themeName == "" {
		themeName = strings.TrimPrefix(r.URL.Path, "/manage/themes/copy/")
	}
	if themeName == "" {
		http.NotFound(w, r)
		return
	}

	src := filepath.Join("themes", themeName)
	var dst string

	if r.Method == http.MethodPost {
		newName := r.FormValue("new_name")
		if newName == "" {
			http.Error(w, "New name is required", http.StatusBadRequest)
			return
		}
		// Sanitize newName: only alphanumeric, dashes, underscores
		newName = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return -1
		}, newName)

		dst = filepath.Join("themes", newName)
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			middleware.GenerateFlashCookie(w, "A theme with that name already exists")
			http.Redirect(w, r, "/manage/themes", http.StatusFound)
			return
		}
	} else {
		base := themeName + "-copy"
		dst = filepath.Join("themes", base)
		for i := 1; ; i++ {
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				break
			}
			dst = filepath.Join("themes", base+"-"+strconv.Itoa(i))
		}
	}

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		newPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(newPath, info.Mode())
		}
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(newPath, input, info.Mode())
	})

	if err != nil {
		log.Printf("Copy error: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to copy theme")
	} else {
		middleware.GenerateFlashCookie(w, "Theme copied to "+filepath.Base(dst))
	}

	http.Redirect(w, r, "/manage/themes", http.StatusFound)
}

// getAvailableLayouts returns a list of available layouts from the active theme
func getAvailableLayouts() []string {
	var layouts []string
	if ThemeEngine != nil {
		for k := range ThemeEngine.Templates {
			if strings.HasSuffix(k, ".html") {
				layouts = append(layouts, k)
			}
		}
	}
	return layouts
}
