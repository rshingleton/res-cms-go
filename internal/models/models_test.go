package models

import (
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	dbFile := "test_models.db"
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate schemas
	err = db.AutoMigrate(&User{}, &Entry{}, &Page{}, &Tag{}, &Comment{}, &SiteSetting{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dbFile)
		os.Remove(dbFile + "-shm")
		os.Remove(dbFile + "-wal")
	}

	return db, cleanup
}

func TestEntryCreation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a user
	user := User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create pages
	page := Page{
		Title: "Tech",
		Slug:  "tech",
	}
	if err := db.Create(&page).Error; err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Create tags
	tag := Tag{
		Name: "Go",
		Slug: "go",
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	// Create entry
	entry := Entry{
		AccountID:  user.ID,
		EntryTitle: "Hello World",
		Slug:       "hello-world",
		Content:    "This is a test post.",
		Status:     "published",
		Pages:      []Page{page},
		Tags:       []Tag{tag},
	}

	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Verify entry exists and associations work
	var fetchedEntry Entry
	err := db.Preload("Author").Preload("Pages").Preload("Tags").First(&fetchedEntry, entry.ID).Error
	if err != nil {
		t.Fatalf("failed to fetch entry: %v", err)
	}

	if fetchedEntry.EntryTitle != "Hello World" {
		t.Errorf("expected title Hello World, got %s", fetchedEntry.EntryTitle)
	}

	if len(fetchedEntry.Pages) != 1 || fetchedEntry.Pages[0].Title != "Tech" {
		t.Errorf("page association failed")
	}

	if len(fetchedEntry.Tags) != 1 || fetchedEntry.Tags[0].Name != "Go" {
		t.Errorf("tag association failed")
	}

	if fetchedEntry.Author.Username != "testuser" {
		t.Errorf("author association failed")
	}
}

func TestCommentCreation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create entry first
	entry := Entry{
		AccountID:  1,
		EntryTitle: "Post for comments",
		Slug:       "post-for-comments",
	}
	db.Create(&entry)

	comment := Comment{
		EntryID: entry.ID,
		Author:  "John Doe",
		Content: "Nice post!",
		Status:  "approved",
	}

	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}

	var fetchedComment Comment
	db.Preload("Post").First(&fetchedComment, comment.ID)

	if fetchedComment.Author != "John Doe" {
		t.Errorf("expected author John Doe, got %s", fetchedComment.Author)
	}

	if fetchedComment.Post.EntryTitle != "Post for comments" {
		t.Errorf("comment to post association failed")
	}
}
