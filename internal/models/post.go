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
	"time"
)

// Post represents a blog post
type Post struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AccountID       uint      `gorm:"not null" json:"account_id"`
	Title           string    `gorm:"size:255;not null" json:"title"`
	Slug            string    `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Content         string    `gorm:"type:text" json:"content"`
	Status          string    `gorm:"default:draft" json:"status"`
	CommentsEnabled bool      `gorm:"default:true" json:"comments_enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Associations
	Author   User      `gorm:"foreignKey:AccountID" json:"author,omitempty"`
	Pages    []Page    `gorm:"many2many:post_pages;" json:"pages,omitempty"`
	Tags     []Tag     `gorm:"many2many:post_tags;" json:"tags,omitempty"`
	Comments []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

// TableName overrides the table name
func (Post) TableName() string {
	return "posts"
}
