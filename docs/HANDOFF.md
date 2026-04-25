# ResCMS Go Handoff

## Project Status: STABLE (v2.0.0)

ResCMS Go has been fully migrated to a Go backend and stabilized for production editorial use. The system is lightweight, fast, and features a modular theme engine.

## 🛠 Technical Stack
- **Backend**: Go 1.22+ (using standard `http.ServeMux` with path values).
- **Database**: SQLite (`res-cms.db`).
- **Editor**: Quill with custom Image Resize and Server Upload handlers.
- **Frontend**: Alpine.js for reactivity, Tailwind CSS for Admin, Axentix for Themes.
- **Theme Engine**: Dynamic template loader with hot-reload in development.

## 🔑 Key Features & Operations
- **Admin Dashboard**: Accessible at `/manage`.
- **Theme Management**: 
    - Edit theme files directly via the Monaco-based Editor.
    - Copy existing themes via the "Copy" action in the Theme list.
- **Media**: Images uploaded via the editor are saved to `/public/uploads/`. The `/api/upload/image` endpoint is protected by Auth middleware.

## ⚠️ Important Implementation Details
- **Pages as Categories**: The system uses the `models.Page` model for both standalone pages and article classifications. Non-system pages are listed as "Pages" in the Article sidebar.
- **JSON Injection**: Use the `| js` template filter when injecting Go models into Alpine.js `x-init` or other scripts to avoid syntax errors.
- **Asset Paths**: Use `/themes/[theme-name]/` for theme assets and `/static/` for shared CMS assets.

## 🗺 Critical Path for Next Development
1. **Media Library**: Build a dedicated file browser in the admin panel to manage existing uploads.
2. **Theme Packaging**: Implement ZIP export and ZIP upload for themes.
3. **Revision History**: Add a table to track content versions and allow rollbacks.

## 🚀 Running the Project
```bash
go run cmd/res-cms/main.go
```
The application listens on `:3000` by default.