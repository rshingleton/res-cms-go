package models

import (
	"time"
)

// User represents a system user
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Status    string    `gorm:"default:activated" json:"status"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	Role      string    `gorm:"default:user" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Entries []Entry `gorm:"foreignKey:AccountID" json:"entries,omitempty"`
}

// TableName overrides the table name
func (User) TableName() string {
	return "accounts"
}
