package models

import (
	"time"
)

// Entry represents a blog post
type Entry struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	AccountID  uint      `gorm:"not null" json:"account_id"`
	EntryTitle string    `gorm:"size:255;not null" json:"entry_title"`
	Slug       string    `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Content    string    `gorm:"type:text" json:"content"`
	Status     string    `gorm:"default:draft" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Associations
	Author     User       `gorm:"foreignKey:AccountID" json:"author,omitempty"`
	Pages      []Page     `gorm:"many2many:entry_pages;" json:"pages,omitempty"`
	Tags       []Tag      `gorm:"many2many:entry_tags;" json:"tags,omitempty"`
	Comments   []Comment  `gorm:"foreignKey:EntryID" json:"comments,omitempty"`
}

// TableName overrides the table name
func (Entry) TableName() string {
	return "entries"
}
