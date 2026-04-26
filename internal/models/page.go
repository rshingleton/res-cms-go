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

// Page represents a standalone page that can also act as a category/taxonomy
type Page struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Slug      string    `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Content   string    `gorm:"type:text" json:"content"`
	IsSystem  bool      `gorm:"default:false" json:"is_system"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	Layout          string    `gorm:"size:255" json:"layout"`
	CommentsEnabled bool      `gorm:"default:true" json:"comments_enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Associations
	Posts []Post `gorm:"many2many:post_pages;" json:"posts,omitempty"`
}

// TableName overrides the table name
func (Page) TableName() string {
	return "pages"
}
