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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"res-cms-go/internal/session"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) func() {
	dbFile := "test_handlers.db"
	var err error
	db.DB, err = gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate schemas
	err = db.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.Page{}, &models.Tag{}, &models.Comment{}, &models.SiteSetting{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed some data
	user := models.User{Username: "admin", Email: "admin@example.com", Role: "admin", IsAdmin: true}
	db.DB.Create(&user)

	post := models.Post{
		AccountID:  user.ID,
		Title: "Test Post",
		Slug:       "test-post",
		Content:    "Content",
		Status:     "published",
	}
	db.DB.Create(&post)

	cleanup := func() {
		sqlDB, _ := db.DB.DB()
		sqlDB.Close()
		os.Remove(dbFile)
		os.Remove(dbFile + "-shm")
		os.Remove(dbFile + "-wal")
	}

	return cleanup
}

func TestAPIListPostsHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	req, err := http.NewRequest("GET", "/api/v1/posts", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIListPostsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	posts := response["posts"].([]interface{})
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %v", len(posts))
	}
}

func TestAPIAdminSavePostHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	// Mock user session
	sess := &session.Session{
		UserID:   1,
		Username: "admin",
		IsAdmin:  true,
	}

	input := map[string]interface{}{
		"title":   "New Admin Post",
		"slug":    "new-admin-post",
		"content": "Admin content",
		"status":  "published",
	}
	body, _ := json.Marshal(input)

	req, err := http.NewRequest("POST", "/api/admin/posts", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// Inject user into context
	ctx := middleware.WithUser(req.Context(), sess)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIAdminSavePostHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify post was created in DB
	var entry models.Post
	if err := db.DB.Where("slug = ?", "new-admin-post").First(&entry).Error; err != nil {
		t.Errorf("post was not created in database")
	}

	if entry.Title != "New Admin Post" {
		t.Errorf("expected title New Admin Post, got %s", entry.Title)
	}
}

func TestAPIListPagesHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	db.DB.Create(&models.Page{Title: "Cat 1", Slug: "cat-1"})

	req, _ := http.NewRequest("GET", "/api/v1/pages", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIListPagesHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var pages []models.Page
	json.NewDecoder(rr.Body).Decode(&pages)
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}
}

func TestAPIListTagsHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	db.DB.Create(&models.Tag{Name: "Tag 1", Slug: "tag-1"})

	req, _ := http.NewRequest("GET", "/api/v1/tags", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIListTagsHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var tags []models.Tag
	json.NewDecoder(rr.Body).Decode(&tags)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
}

func TestAPIAdminListStatsHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/api/admin/stats", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIAdminListStatsHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var stats map[string]int64
	json.NewDecoder(rr.Body).Decode(&stats)
	if stats["posts"] != 1 {
		t.Errorf("expected 1 post in stats, got %d", stats["posts"])
	}
}

func TestAPIAdminDeletePostHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	post := models.Post{AccountID: 1, Title: "To delete", Slug: "to-delete"}
	db.DB.Create(&post)

	// Route is /api/admin/posts/{id}
	req, _ := http.NewRequest("DELETE", "/api/admin/posts/2", nil) // ID might be 2 after seeding
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIAdminDeletePostHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var count int64
	db.DB.Model(&models.Post{}).Where("id = ?", 2).Count(&count)
	if count != 0 {
		t.Errorf("post was not deleted")
	}
}

func TestAPISubmitCommentHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	input := map[string]interface{}{
		"post_id": 1,
		"author":  "Commenter",
		"email":   "comment@example.com",
		"content": "Great post!",
	}
	body, _ := json.Marshal(input)

	req, err := http.NewRequest("POST", "/api/v1/comments", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APISubmitCommentHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Verify comment was created
	var comment models.Comment
	if err := db.DB.Where("author = ?", "Commenter").First(&comment).Error; err != nil {
		t.Errorf("comment was not created in database")
	}
}
