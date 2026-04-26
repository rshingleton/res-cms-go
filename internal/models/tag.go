package models

import (
	"time"
)

// Tag represents a post tag
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Slug      string    `gorm:"uniqueIndex;size:50;not null" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Posts []Post `gorm:"many2many:post_tags;" json:"posts,omitempty"`
}

// TableName overrides the table name
func (Tag) TableName() string {
	return "tags"
}
