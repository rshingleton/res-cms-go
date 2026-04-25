package models

import (
	"time"
)

// Page represents a standalone page that can also act as a category/taxonomy
type Page struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Slug      string    `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Content   string    `gorm:"type:text" json:"content"`
	IsSystem  bool      `gorm:"default:false" json:"is_system"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Entries []Entry `gorm:"many2many:entry_pages;" json:"entries,omitempty"`
}

// TableName overrides the table name
func (Page) TableName() string {
	return "pages"
}
