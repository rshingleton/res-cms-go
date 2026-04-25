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
