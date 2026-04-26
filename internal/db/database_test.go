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

package db

import (
	"os"
	"res-cms-go/internal/models"
	"testing"
)

func TestInit(t *testing.T) {
	// Use a temporary file for the test database
	dbFile := "test_rescms.db"
	defer os.Remove(dbFile)
	defer os.Remove(dbFile + "-shm")
	defer os.Remove(dbFile + "-wal")

	dsn := "sqlite:" + dbFile
	err := Init(dsn, false)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	if DB == nil {
		t.Fatal("DB instance is nil after Init")
	}

	// Verify that tables are created by checking if we can count users
	var count int64
	err = DB.Model(&models.User{}).Count(&count).Error
	if err != nil {
		t.Errorf("Failed to count users: %v", err)
	}

	// Verify seeding
	if count == 0 {
		t.Error("Default admin user was not seeded")
	}

	var admin models.User
	err = DB.Where("username = ?", "admin").First(&admin).Error
	if err != nil {
		t.Errorf("Failed to find seeded admin user: %v", err)
	}

	if admin.Role != "admin" {
		t.Errorf("Expected admin role, got %s", admin.Role)
	}

	// Verify settings seeding
	var settingCount int64
	DB.Model(&models.SiteSetting{}).Count(&settingCount)
	if settingCount == 0 {
		t.Error("Default settings were not seeded")
	}

	Close()
}
