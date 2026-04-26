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
	err = db.AutoMigrate(&User{}, &Post{}, &Page{}, &Tag{}, &Comment{}, &SiteSetting{})
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

func TestPostCreation(t *testing.T) {
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
	entry := Post{
		AccountID: user.ID,
		Title:     "Hello World",
		Slug:      "hello-world",
		Content:   "This is a test post.",
		Status:    "published",
		Pages:     []Page{page},
		Tags:      []Tag{tag},
	}

	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Verify entry exists and associations work
	var fetchedPost Post
	err := db.Preload("Author").Preload("Pages").Preload("Tags").First(&fetchedPost, entry.ID).Error
	if err != nil {
		t.Fatalf("failed to fetch entry: %v", err)
	}

	if fetchedPost.Title != "Hello World" {
		t.Errorf("expected title Hello World, got %s", fetchedPost.Title)
	}

	if len(fetchedPost.Pages) != 1 || fetchedPost.Pages[0].Title != "Tech" {
		t.Errorf("page association failed")
	}

	if len(fetchedPost.Tags) != 1 || fetchedPost.Tags[0].Name != "Go" {
		t.Errorf("tag association failed")
	}

	if fetchedPost.Author.Username != "testuser" {
		t.Errorf("author association failed")
	}
}

func TestCommentCreation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create entry first
	entry := Post{
		AccountID: 1,
		Title:     "Post for comments",
		Slug:      "post-for-comments",
	}
	db.Create(&entry)

	comment := Comment{
		PostID:  entry.ID,
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

	if fetchedComment.Post.Title != "Post for comments" {
		t.Errorf("comment to post association failed")
	}
}
