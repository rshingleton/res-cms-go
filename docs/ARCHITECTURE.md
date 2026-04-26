# ResCMS Go Architecture

## Overview
ResCMS is a modern, Go-based Content Management System. It features an API-driven architecture, a modular theme engine, and a reactive admin dashboard with a unified code editor.

## Technology Stack
- **Backend**: Go (Golang)
- **Database**: SQLite, MySQL, PostgreSQL (via GORM)
- **Frontend Logic**: Alpine.js (with Collapse plugin)

- **Styling**: Axentix CSS (Standardized across Admin and Themes)
- **Templates**: `html/template` (Go Standard Library)
- **Editor**: Monaco (for code), Quill (for content)

## Core Components

### 1. Theme Engine (`internal/theme`)
The Theme Engine handles the modular delivery of the public-facing UI and automated customization injections.
- **Manifest**: `theme.json` defines metadata and visual tokens.
- **Dynamic Injections**: Automatically wraps and injects raw CSS/JS from site settings into theme layouts.
- **Hot-Reload**: Themes are reloaded in development mode for instant feedback.

### 2. Admin Super Editor (`/manage/editor`)
A centralized hub for all site-wide code modifications.
- **Unified Selection**: Single-page interface to browse and edit all installed themes.
- **Global Customization**: Manage injectable CSS/JS/HTML with enable/disable toggles stored in the database.
- **Automated Tagging**: Backend logic ensures raw inputs are correctly wrapped in `<style>` or `<script>` tags before delivery.

### 3. API Handlers (`internal/handlers`)
- **Admin Unified Handlers**: Single POST endpoints for managing filesystem (themes) and database (settings) updates.
- **Root Render Engine**: Optimized `renderTemplate` function that maps global settings to theme variables (`res_` prefix).

### 4. Database Models (`internal/models`)
- **Post**: Represents a blog post or primary content item.
- **Page**: Handles hierarchical grouping (categories) and static standalone pages.
- **SiteSetting**: Stores all configuration and global customizations (CSS, JS, Header/Footer HTML).

## Project Structure
```text
res-cms-go/
├── cmd/res-cms/          # Application entry point
├── internal/
│   ├── db/               # Database initialization & Seeding
│   ├── handlers/         # Unified Admin and Public handlers
│   ├── middleware/       # Auth, Flash, and Context logic
│   ├── models/           # Data structures
│   ├── theme/            # Core Engine logic
│   └── ui/               # Admin UI Templates
├── themes/               # Theme Library (classic, pixel-standard)
├── public/               # Uploads and Static assets
├── docs/                 # Documentation & Specification
└── rescms.yml            # Application Configuration
```
