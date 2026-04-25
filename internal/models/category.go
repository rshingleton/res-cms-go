package models

import (
	"time"
)

// Category represents a post category
type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Slug      string    `gorm:"uniqueIndex;size:100;not null" json:"slug"`
	ParentID  *uint     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Parent  *Category `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Entries []Entry   `gorm:"many2many:entry_categories;" json:"entries,omitempty"`
}

// TableName overrides the table name
func (Category) TableName() string {
	return "categories"
}
