package handlers

import (
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
	var postCount, commentCount, userCount, categoryCount int64
	db.DB.Model(&models.Entry{}).Count(&postCount)
	db.DB.Model(&models.Comment{}).Count(&commentCount)
	db.DB.Model(&models.User{}).Count(&userCount)
	db.DB.Model(&models.Category{}).Count(&categoryCount)

	// Get recent posts
	var recentPosts []models.Entry
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
		"CategoryCount":   categoryCount,
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
	if r.URL.Path != "/manage/entries" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var entries []models.Entry
	db.DB.Preload("Author").Order("created_at DESC").Find(&entries)

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Entries":   entries,
		"ActiveTab": "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddPostFormHandler shows add post form
func AdminAddPostFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/entries/new" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var categories []models.Category
	db.DB.Find(&categories)

	var tags []models.Tag
	db.DB.Find(&tags)

	data := map[string]interface{}{
		"BlogName":   getBlogName(),
		"User":       user,
		"Categories": categories,
		"Tags":       tags,
		"Entry":      models.Entry{},
		"IsNew":      true,
		"ActiveTab":  "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddPostHandler creates a new post
func AdminAddPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/entries" || r.Method != http.MethodPost {
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
		http.Redirect(w, r, "/manage/entries", http.StatusFound)
		return
	}

	title := r.PostForm.Get("title")
	slug := r.PostForm.Get("slug")
	content := r.PostForm.Get("content")
	status := r.PostForm.Get("status")
	categoryIDs := r.PostForm["categories"]
	tagIDs := r.PostForm["tags"]

	if title == "" || slug == "" {
		middleware.GenerateFlashCookie(w, "Title and slug are required")
		http.Redirect(w, r, "/manage/entries/new", http.StatusFound)
		return
	}

	entry := models.Entry{
		AccountID:  user.UserID,
		EntryTitle: title,
		Slug:       slug,
		Content:    content,
		Status:     status,
	}

	if err := db.DB.Create(&entry).Error; err != nil {
		log.Printf("Error creating entry: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create entry")
		http.Redirect(w, r, "/manage/entries/new", http.StatusFound)
		return
	}

	// Associate categories
	if len(categoryIDs) > 0 {
		var categories []models.Category
		for _, id := range categoryIDs {
			if cid, err := strconv.ParseUint(id, 10, 32); err == nil {
				categories = append(categories, models.Category{ID: uint(cid)})
			}
		}
		db.DB.Model(&entry).Association("Categories").Append(&categories)
	}

	// Associate tags
	if len(tagIDs) > 0 {
		var tags []models.Tag
		for _, id := range tagIDs {
			if tid, err := strconv.ParseUint(id, 10, 32); err == nil {
				tags = append(tags, models.Tag{ID: uint(tid)})
			}
		}
		db.DB.Model(&entry).Association("Tags").Append(&tags)
	}

	middleware.GenerateFlashCookie(w, "Entry created successfully")
	http.Redirect(w, r, "/manage/entries", http.StatusFound)
}

// AdminEditPostFormHandler shows edit post form
func AdminEditPostFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/entries/edit/")
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

	var entry models.Entry
	if err := db.DB.Preload("Categories").Preload("Tags").First(&entry, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var categories []models.Category
	db.DB.Find(&categories)

	var tags []models.Tag
	db.DB.Find(&tags)

	data := map[string]interface{}{
		"BlogName":   getBlogName(),
		"User":       user,
		"Entry":      entry,
		"Categories": categories,
		"Tags":       tags,
		"IsNew":      false,
		"ActiveTab":  "posts",
	}

	if err := renderTemplate(w, r, "admin/posts/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUpdatePostHandler updates a post
func AdminUpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/entries/update/")
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
		http.Redirect(w, r, "/manage/entries", http.StatusFound)
		return
	}

	title := r.PostForm.Get("title")
	slug := r.PostForm.Get("slug")
	content := r.PostForm.Get("content")
	status := r.PostForm.Get("status")
	categoryIDs := r.PostForm["categories"]
	tagIDs := r.PostForm["tags"]

	updates := map[string]interface{}{
		"entry_title": title,
		"slug":        slug,
		"content":     content,
		"status":      status,
	}

	if err := db.DB.Model(&models.Entry{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.Printf("Error updating entry: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to update entry")
		http.Redirect(w, r, "/manage/entries/edit/"+idStr, http.StatusFound)
		return
	}

	// Update categories
	var entry models.Entry
	db.DB.First(&entry, id)
	if len(categoryIDs) > 0 {
		db.DB.Model(&entry).Association("Categories").Clear()
		var categories []models.Category
		for _, id := range categoryIDs {
			if cid, err := strconv.ParseUint(id, 10, 32); err == nil {
				categories = append(categories, models.Category{ID: uint(cid)})
			}
		}
		db.DB.Model(&entry).Association("Categories").Append(&categories)
	}

	// Update tags
	if len(tagIDs) > 0 {
		db.DB.Model(&entry).Association("Tags").Clear()
		var tags []models.Tag
		for _, id := range tagIDs {
			if tid, err := strconv.ParseUint(id, 10, 32); err == nil {
				tags = append(tags, models.Tag{ID: uint(tid)})
			}
		}
		db.DB.Model(&entry).Association("Tags").Append(&tags)
	}

	middleware.GenerateFlashCookie(w, "Entry updated successfully")
	http.Redirect(w, r, "/manage/entries", http.StatusFound)
}

// AdminDeletePostHandler deletes a post
func AdminDeletePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/entries/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Delete(&models.Entry{}, id).Error; err != nil {
		log.Printf("Error deleting entry: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete entry")
	} else {
		middleware.GenerateFlashCookie(w, "Entry deleted successfully")
	}

	http.Redirect(w, r, "/manage/entries", http.StatusFound)
}

// AdminPublishPostHandler publishes a post
func AdminPublishPostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/entries/publish/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Entry{}).Where("id = ?", id).Update("status", "published").Error; err != nil {
		log.Printf("Error publishing entry: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to publish entry")
	} else {
		middleware.GenerateFlashCookie(w, "Entry published successfully")
	}

	http.Redirect(w, r, "/manage/entries", http.StatusFound)
}

// AdminDraftPostHandler sets post to draft
func AdminDraftPostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/entries/draft/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Model(&models.Entry{}).Where("id = ?", id).Update("status", "draft").Error; err != nil {
		log.Printf("Error setting draft: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to set draft")
	} else {
		middleware.GenerateFlashCookie(w, "Entry set to draft")
	}

	http.Redirect(w, r, "/manage/entries", http.StatusFound)
}

// ==================== CATEGORIES ====================

// AdminListCategoriesHandler lists all categories
func AdminListCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/categories" {
		http.NotFound(w, r)
		return
	}

	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var categories []models.Category
	db.DB.Preload("Parent").Order("name").Find(&categories)

	data := map[string]interface{}{
		"BlogName":   getBlogName(),
		"User":       user,
		"Categories": categories,
		"ActiveTab":  "categories",
	}

	if err := renderTemplate(w, r, "admin/categories/list.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminAddCategoryHandler creates a new category
func AdminAddCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/manage/categories" || r.Method != http.MethodPost {
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
		http.Redirect(w, r, "/manage/categories", http.StatusFound)
		return
	}

	name := r.PostForm.Get("name")
	slug := r.PostForm.Get("slug")
	parentIDStr := r.PostForm.Get("parent_id")

	if name == "" || slug == "" {
		middleware.GenerateFlashCookie(w, "Name and slug are required")
		http.Redirect(w, r, "/manage/categories", http.StatusFound)
		return
	}

	category := models.Category{
		Name: name,
		Slug: slug,
	}

	if parentIDStr != "" {
		if pid, err := strconv.ParseUint(parentIDStr, 10, 32); err == nil {
			category.ParentID = new(uint)
			*category.ParentID = uint(pid)
		}
	}

	if err := db.DB.Create(&category).Error; err != nil {
		log.Printf("Error creating category: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to create category")
	} else {
		middleware.GenerateFlashCookie(w, "Category created successfully")
	}

	http.Redirect(w, r, "/manage/categories", http.StatusFound)
}

// AdminEditCategoryFormHandler shows edit category form
func AdminEditCategoryFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/categories/edit/")
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

	var category models.Category
	if err := db.DB.Preload("Parent").First(&category, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var categories []models.Category
	db.DB.Where("id != ?", id).Find(&categories)

	data := map[string]interface{}{
		"BlogName":   getBlogName(),
		"User":       user,
		"Category":   category,
		"Categories": categories,
		"ActiveTab":  "categories",
	}

	if err := renderTemplate(w, r, "admin/categories/edit.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// AdminUpdateCategoryHandler updates a category
func AdminUpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/categories/update/")
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
		http.Redirect(w, r, "/manage/categories", http.StatusFound)
		return
	}

	name := r.PostForm.Get("name")
	slug := r.PostForm.Get("slug")
	parentIDStr := r.PostForm.Get("parent_id")

	updates := map[string]interface{}{
		"name": name,
		"slug": slug,
	}

	if parentIDStr != "" {
		if pid, err := strconv.ParseUint(parentIDStr, 10, 32); err == nil {
			updates["parent_id"] = pid
		}
	} else {
		updates["parent_id"] = nil
	}

	if err := db.DB.Model(&models.Category{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.Printf("Error updating category: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to update category")
	} else {
		middleware.GenerateFlashCookie(w, "Category updated successfully")
	}

	http.Redirect(w, r, "/manage/categories", http.StatusFound)
}

// AdminDeleteCategoryHandler deletes a category
func AdminDeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manage/categories/delete/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := db.DB.Delete(&models.Category{}, id).Error; err != nil {
		log.Printf("Error deleting category: %v", err)
		middleware.GenerateFlashCookie(w, "Failed to delete category")
	} else {
		middleware.GenerateFlashCookie(w, "Category deleted successfully")
	}

	http.Redirect(w, r, "/manage/categories", http.StatusFound)
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

		blogName := r.PostForm.Get("blog_name")
		layoutStyle := r.PostForm.Get("layout_style")

		if blogName != "" {
			db.DB.Model(&models.SiteSetting{}).Where("name = ?", "blog_name").Update("value", blogName)
		}
		if layoutStyle != "" {
			db.DB.Model(&models.SiteSetting{}).Where("name = ?", "layout_style").Update("value", layoutStyle)
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

	data := map[string]interface{}{
		"BlogName":  getBlogName(),
		"User":      user,
		"Themes":    themeList,
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

	// In a real app, save to DB. For now, just reload in memory.
	err := ThemeEngine.LoadTheme(themeName, template.FuncMap{
		"formatDate": func(t interface{}) string { return "2024-01-01" },
		"json": func(v interface{}) string {
			return "{}" // Placeholder or use a central helper
		},
		"js": func(v interface{}) template.JS {
			return template.JS("{}")
		},
		"toUpper": func(s string) string { return strings.ToUpper(s) },
	})

	if err != nil {
		middleware.GenerateFlashCookie(w, "Failed to activate theme: "+err.Error())
	} else {
		middleware.GenerateFlashCookie(w, "Theme "+themeName+" activated")
	}

	http.Redirect(w, r, "/manage/themes", http.StatusFound)
}
