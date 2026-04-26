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

package main

import (
	"log"

	"res-cms-go/internal/config"
	"res-cms-go/internal/db"
	"res-cms-go/internal/models"
)

func main() {
	cfg, err := config.Load("rescms.yml")
	if err != nil {
		log.Printf("Warning: Could not load config, using defaults: %v", err)
		cfg = &config.Config{SQLiteDSN: "data/rescms.db"}
	}
	db.Init(cfg.SQLiteDSN, false)
	database := db.GetDB()

	log.Println("Starting Category -> Page migration...")

	// 1. Auto-migrate new models
	err = database.AutoMigrate(&models.Page{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate Page model: %v", err)
	}

	// 2. Ensure Main Page exists
	var mainPage models.Page
	result := database.Where("is_system = ?", true).First(&mainPage)
	if result.Error != nil {
		mainPage = models.Page{
			Title:    "Welcome to ResCMS",
			Slug:     "home",
			Content:  "<h2>Explore the Digital Frontier</h2><p>This is the system-managed main page for your new CMS. You can edit this content directly from the administration dashboard. Use the powerful rich-text editor to share your thoughts, projects, and insights with the world.</p><p>Check out the <strong>Recent Posts</strong> on the right to see what's new!</p>",
			IsSystem: true,
		}
		if err := database.Create(&mainPage).Error; err != nil {
			log.Fatalf("Failed to create main page: %v", err)
		}
		log.Println("Created Main system page.")
	}

	// 3. Migrate existing categories
	type OldCategory struct {
		ID   uint
		Name string
		Slug string
	}
	var oldCategories []OldCategory
	if database.Migrator().HasTable("categories") {
		database.Table("categories").Find(&oldCategories)

		for _, cat := range oldCategories {
			var page models.Page
			if err := database.Where("slug = ?", cat.Slug).First(&page).Error; err != nil {
				// Page doesn't exist, create it
				newPage := models.Page{
					ID:       cat.ID, // Preserve ID to keep many2many relationships if we migrate them directly
					Title:    cat.Name,
					Slug:     cat.Slug,
					Content:  "<h2>" + cat.Name + "</h2><p>Page automatically generated from category.</p>",
					IsSystem: false,
				}
				database.Create(&newPage)
				log.Printf("Migrated category '%s' to page.", cat.Name)
			}
		}

		// 4. Migrate entry_categories to entry_pages
		if database.Migrator().HasTable("entry_categories") {
			if !database.Migrator().HasTable("entry_pages") {
				database.Exec("CREATE TABLE entry_pages (entry_id INTEGER, page_id INTEGER, PRIMARY KEY (entry_id, page_id))")
			}
			database.Exec("INSERT INTO entry_pages (entry_id, page_id) SELECT entry_id, category_id FROM entry_categories ON CONFLICT DO NOTHING")
			log.Println("Migrated entry associations.")
		}

		// Optional: drop old table if desired, but better to keep for safety initially
		// database.Migrator().DropTable("entry_categories")
		// database.Migrator().DropTable("categories")
	}

	log.Println("Migration completed successfully.")
}
