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
	err = db.DB.AutoMigrate(&models.User{}, &models.Entry{}, &models.Category{}, &models.Tag{}, &models.Comment{}, &models.SiteSetting{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed some data
	user := models.User{Username: "admin", Email: "admin@example.com", Role: "admin", IsAdmin: true}
	db.DB.Create(&user)

	post := models.Entry{
		AccountID:  user.ID,
		EntryTitle: "Test Post",
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
	var entry models.Entry
	if err := db.DB.Where("slug = ?", "new-admin-post").First(&entry).Error; err != nil {
		t.Errorf("post was not created in database")
	}

	if entry.EntryTitle != "New Admin Post" {
		t.Errorf("expected title New Admin Post, got %s", entry.EntryTitle)
	}
}

func TestAPIListCategoriesHandler(t *testing.T) {
	cleanup := setupHandlerTestDB(t)
	defer cleanup()

	db.DB.Create(&models.Category{Name: "Cat 1", Slug: "cat-1"})

	req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(APIListCategoriesHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var cats []models.Category
	json.NewDecoder(rr.Body).Decode(&cats)
	if len(cats) != 1 {
		t.Errorf("expected 1 category, got %d", len(cats))
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

	post := models.Entry{AccountID: 1, EntryTitle: "To delete", Slug: "to-delete"}
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
	db.DB.Model(&models.Entry{}).Where("id = ?", 2).Count(&count)
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
