# ResCMS Go Architecture

## Overview
ResCMS is a modern, Go-based Content Management System. It features an API-driven architecture, a modular theme engine, and a reactive admin dashboard.

## Technology Stack
- **Backend**: Go (Golang)
- **Database**: SQLite (via GORM)
- **Frontend Logic**: Alpine.js
- **Styling**: Tailwind CSS (for Admin/Classic) & Vanilla CSS (for Pixel Theme)
- **Templates**: `html/template` (Go Standard Library)

## Core Components

### 1. Theme Engine (`internal/theme`)
The Theme Engine handles the modular delivery of the public-facing UI.
- **Location**: `/themes/[theme-name]`
- **Manifest**: `theme.json` defines metadata and color tokens.
- **Assets**: Themes manage their own CSS, JS, and Images.
- **Dynamic Loading**: Themes are loaded and validated at runtime, allowing for instant swaps and zip-based installation.

### 2. Admin UI (`internal/ui/admin`)
The administrative interface is decoupled from user themes.
- **Location**: `internal/ui/admin` (Protected internal path)
- **Reactive Design**: Built with Alpine.js and Pines UI for a seamless, SPA-like experience within a server-rendered shell.

### 3. API Handlers (`internal/handlers`)
- **API v1**: JSON endpoints for posts, categories, and tags.
- **Admin Handlers**: Secure routes for content management, user administration, and theme control.
- **Root Handlers**: Dynamic template rendering that bridges the Theme Engine and the Go backend.

### 4. Database Models (`internal/models`)
- **Entry**: Blog posts and pages.
- **Account**: User management and authentication.
- **Comment**: Interactive feedback system.
- **Category/Tag**: Taxonomies for content organization.

## Project Structure
```text
res-cms-go/
├── cmd/res-cms/          # Application entry point
├── internal/
│   ├── db/               # GORM initialization
│   ├── handlers/         # API and Web handlers
│   ├── middleware/       # Auth and Flash logic
│   ├── models/           # Data structures
│   ├── session/          # Secure session management
│   ├── theme/            # Theme Engine core
│   └── ui/               # System UI (Admin)
├── themes/               # Modular public skins
├── public/               # Shared static assets (JS libraries, etc.)
├── data/                 # SQLite database and uploads
└── docs/                 # Documentation
```

## Theme Specification
See [THEME_SPEC.md](./THEME_SPEC.md) for detailed information on creating and managing themes.