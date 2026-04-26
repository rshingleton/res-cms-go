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
	"fmt"
	"log"
	"os"

	"res-cms-go/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection
var DB *gorm.DB

// Init initializes the database connection and runs migrations
func Init(dsn string, production bool) error {
	var err error

	// Ensure directory exists
	if dir := getDirFromDSN(dsn); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database
	config := &gorm.Config{}
	if !production {
		config.Logger = logger.Default.LogMode(logger.Warn)
	}

	path := dsn
	if len(dsn) > 7 && dsn[:7] == "sqlite:" {
		path = dsn[7:]
	}

	DB, err = gorm.Open(sqlite.Open(path), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure SQLite
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// WAL mode for better concurrency
	if err := DB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		log.Printf("Warning: could not set WAL mode: %v", err)
	}
	if err := DB.Exec("PRAGMA synchronous=NORMAL").Error; err != nil {
		log.Printf("Warning: could not set synchronous mode: %v", err)
	}

	// Set max open connections
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	// Run migrations
	if err := migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed default data
	if err := seed(); err != nil {
		return fmt.Errorf("failed to seed data: %w", err)
	}

	log.Println("Database initialized successfully")

	// Seed system Home page if not exists
	var pageCount int64
	DB.Model(&models.Page{}).Where("is_system = ?", true).Count(&pageCount)
	if pageCount == 0 {
		homePage := models.Page{
			Title:    "Welcome to ResCMS",
			Slug:     "home",
			IsSystem: true,
		}
		if err := DB.Create(&homePage).Error; err != nil {
			log.Printf("Warning: failed to create home page: %v", err)
		} else {
			log.Println("Created system Home page")
		}
	}

	return nil
}

func getDirFromDSN(dsn string) string {
	// Extract directory from DSN like "sqlite:data/rescms.db"
	if len(dsn) > 7 && dsn[:7] == "sqlite:" {
		path := dsn[7:]
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' || path[i] == '\\' {
				return path[:i]
			}
		}
	}
	return ""
}

// migrate runs database migrations
func migrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Page{},
		&models.Tag{},
		&models.Comment{},
		&models.SiteSetting{},
	)
}

// seed seeds default data
func seed() error {
	// Seed default admin user if not exists
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), 10)
		admin := models.User{
			Username: "admin",
			Password: string(hashedPassword),
			Email:    "admin@rescms.com",
			Status:   "activated",
			IsAdmin:  true,
			Role:     "admin",
		}
		if err := DB.Create(&admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		log.Println("Created default admin user")
	}

	// Seed individual settings using FirstOrCreate so they are added to existing DBs
	defaultSettings := []models.SiteSetting{
		{Name: "blog_name", Value: "ResCMS"},
		{Name: "layout_style", Value: "default"},
		{Name: "active_theme", Value: "classic"},
	}
	for _, s := range defaultSettings {
		DB.Where(models.SiteSetting{Name: s.Name}).FirstOrCreate(&s)
	}

	return nil
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
