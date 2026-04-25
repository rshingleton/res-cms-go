package models

import (
	"time"
)

// Comment represents a post comment
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	EntryID   uint      `gorm:"not null" json:"entry_id"`
	Author    string    `gorm:"size:100;not null" json:"author"`
	Email     string    `gorm:"size:100" json:"email"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Status    string    `gorm:"default:pending" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Post Entry `gorm:"foreignKey:EntryID" json:"post,omitempty"`
}

// TableName overrides the table name
func (Comment) TableName() string {
	return "comments"
}
