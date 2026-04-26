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
	"net/url"
	"os"
	"strings"

	"res-cms-go/internal/config"
	"res-cms-go/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection
var DB *gorm.DB

// Init initializes the database connection and runs migrations
func Init(cfg *config.Config) error {
	var err error
	var dialector gorm.Dialector

	dbType := strings.ToLower(cfg.Database.Type)
	if dbType == "" {
		// Fallback to SQLite using SQLiteDSN if Database.Type is not set
		dbType = "sqlite"
		if cfg.SQLiteDSN != "" {
			path := cfg.SQLiteDSN
			if strings.HasPrefix(path, "sqlite:") {
				path = path[7:]
			}
			cfg.Database.Path = path
		}
	}

	switch dbType {
	case "mysql":
		if err := ensureDatabaseExists(dbType, cfg.Database.DSN); err != nil {
			log.Printf("Warning: failed to ensure database exists: %v", err)
		}
		dialector = mysql.Open(cfg.Database.DSN)
	case "postgres", "postgresql":
		if err := ensureDatabaseExists(dbType, cfg.Database.DSN); err != nil {
			log.Printf("Warning: failed to ensure database exists: %v", err)
		}
		dialector = postgres.Open(cfg.Database.DSN)
	case "sqlite", "sqlite3":

		path := cfg.Database.Path
		if path == "" {
			path = "data/rescms.db"
		}
		// Ensure directory exists
		if dir := getDirFromPath(path); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create database directory: %w", err)
			}
		}
		dialector = sqlite.Open(path)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Open database
	gormConfig := &gorm.Config{}
	if !cfg.Production {
		gormConfig.Logger = logger.Default.LogMode(logger.Warn)
	}

	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure DB
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if dbType == "sqlite" || dbType == "sqlite3" {
		// WAL mode for better concurrency
		if cfg.Database.WALMode {
			if err := DB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
				log.Printf("Warning: could not set WAL mode: %v", err)
			}
			if err := DB.Exec("PRAGMA synchronous=NORMAL").Error; err != nil {
				log.Printf("Warning: could not set synchronous mode: %v", err)
			}
		}
	}

	// Set connection pool settings
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

	log.Printf("Database (%s) initialized successfully", dbType)

	// Seed system Home page if not exists
	var pageCount int64
	DB.Model(&models.Page{}).Where("is_system = ?", true).Count(&pageCount)
	if pageCount == 0 {
		homePage := models.Page{
			Title:    "Welcome to ResCMS",
			Slug:     "home",
			IsSystem: true,
		}
		DB.Where(models.Page{Slug: "home", IsSystem: true}).FirstOrCreate(&homePage)
	}

	return nil
}

func getDirFromPath(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

func ensureDatabaseExists(dbType, dsn string) error {
	var adminDSN string
	var dbName string
	var createSQL string

	switch dbType {
	case "mysql":
		// MySQL DSN: user:pass@tcp(host:port)/dbname?params
		parts := strings.Split(dsn, "/")
		if len(parts) < 2 {
			return fmt.Errorf("invalid mysql dsn")
		}
		dbNameParts := strings.Split(parts[1], "?")
		dbName = dbNameParts[0]
		adminDSN = parts[0] + "/?charset=utf8mb4&parseTime=True&loc=Local"
		createSQL = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)

		tempDB, err := gorm.Open(mysql.Open(adminDSN), &gorm.Config{
			Logger: logger.Discard,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to mysql server for bootstrapping: %w", err)
		}
		sqlDB, _ := tempDB.DB()
		defer sqlDB.Close()

		if err := tempDB.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

	case "postgres", "postgresql":
		// Support both Key-Value and URI formats
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			// URI format: postgres://user:pass@host:port/dbname?params
			u, err := url.Parse(dsn)
			if err != nil {
				return fmt.Errorf("failed to parse postgres uri: %w", err)
			}
			dbName = strings.TrimPrefix(u.Path, "/")
			u.Path = "/postgres" // Connect to default 'postgres' db
			adminDSN = u.String()
		} else {
			// Key-Value format: host=localhost user=user password=pass dbname=db port=5432 sslmode=disable
			dbName = getDBNameFromDSN(dsn)
			if dbName == "" {
				return fmt.Errorf("could not find dbname in postgres dsn")
			}
			// Connect to 'postgres' database to create the new one
			if strings.Contains(dsn, "dbname=") {
				adminDSN = strings.Replace(dsn, "dbname="+dbName, "dbname=postgres", 1)
			} else {
				adminDSN = dsn + " dbname=postgres"
			}
		}

		createSQL = fmt.Sprintf("CREATE DATABASE %s", dbName)

		tempDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
			Logger: logger.Discard,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to postgres server for bootstrapping (tried %s): %w", adminDSN, err)
		}
		sqlDB, _ := tempDB.DB()
		defer sqlDB.Close()

		// Check if exists first as Postgres doesn't have "CREATE DATABASE IF NOT EXISTS"
		var exists bool
		err = tempDB.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists).Error
		if err != nil {
			return fmt.Errorf("failed to check if database exists: %w", err)
		}

		if !exists {
			if err := tempDB.Exec(createSQL).Error; err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			log.Printf("Created database %s", dbName)
		}
	}

	return nil
}

func getDBNameFromDSN(dsn string) string {
	// Simple parser for postgres DSN
	parts := strings.Split(dsn, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, "dbname=") {
			return strings.TrimPrefix(p, "dbname=")
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
		&models.Plugin{},
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
		{Name: "posts_comments_enabled", Value: "1"},
		{Name: "pages_comments_enabled", Value: "0"}, // Pages usually have comments disabled by default
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
