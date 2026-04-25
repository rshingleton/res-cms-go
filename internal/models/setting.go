package models

import (
	"time"
)

// SiteSetting represents a site setting key-value pair
type SiteSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the table name
func (SiteSetting) TableName() string {
	return "site_settings"
}
