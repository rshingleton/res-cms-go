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

import "time"

// Plugin stores the persistent state of an installed plugin.
// Enabled/disabled status and configuration survive server restarts via SQLite.
type Plugin struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Slug        string    `gorm:"uniqueIndex;size:100;not null" json:"slug"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Version     string    `gorm:"size:50" json:"version"`
	Author      string    `gorm:"size:255" json:"author"`
	Description string    `gorm:"type:text" json:"description"`
	License     string    `gorm:"size:100" json:"license"`
	WasmFile    string    `gorm:"size:255" json:"wasm_file"`
	Checksum    string    `gorm:"size:64" json:"checksum"`
	// Permissions is a JSON array stored as text, e.g. ["content_write","asset_inject"].
	Permissions string `gorm:"type:text" json:"permissions"`
	// Hooks is a JSON array of hook type strings the plugin subscribes to.
	Hooks   string `gorm:"type:text" json:"hooks"`
	Enabled bool   `gorm:"default:false" json:"enabled"`
	// Config is an opaque JSON blob for plugin-specific settings.
	Config    string    `gorm:"type:text" json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the GORM table name.
func (Plugin) TableName() string {
	return "plugins"
}
