# ResCMS Go Handoff

## Project Status: STABLE (v2.2.0)

ResCMS Go has been fully migrated to a Go backend and stabilized for production editorial use. The system features a robust, unified "Super Editor" and a modular theme engine.

## 🛠 Technical Stack
- **Backend**: Go 1.22+ (using standard `http.ServeMux` with path values).
- **Database**: SQLite (`rescms.db`).
- **Editor**: Monaco (for code), Quill (for content).
- **Frontend Logic**: Alpine.js (with Collapse plugin) for reactivity.
- **Styling**: Axentix CSS (Standardized for both Admin and Themes).
- **Theme Engine**: Dynamic template loader with automated CSS/JS/HTML injection.

## 🔑 Key Features & Operations
- **Admin Dashboard**: Accessible at `/manage`.
- **Super Editor** (`/manage/editor`):
    - **Fully Stabilized**: Monaco editor now correctly expands to fill available space across all views.
    - Edit files from any installed theme via the unified sidebar library.
    - Manage global site injectables (Header JS, Custom CSS, etc.) with automated `<script>` and `<style>` wrapping.
    - Enable/Disable global customizations instantly via UI toggles.
- **Theme Management**: 
    - Duplicate themes with custom names via modal.
    - ZIP export and upload functionality.
- **Media**: Images uploaded via the editor are saved to `/public/uploads/`.

## ⚠️ Important Implementation Details
- **Automated Injections**: Raw CSS/JS entered in the Super Editor is automatically wrapped in tags. Do not add manual `<style>` or `<script>` tags in the "Global Injections" section.
- **Pages as Categories**: The system uses `models.Page` for both standalone pages and categories.
- **JSON Injection**: Use the `| json` or `| js` template filters when injecting Go models into scripts.

## 🗺 Critical Path for Next Development
1. **Media Library**: Build a dedicated file browser in the admin panel to manage existing uploads.
2. **Revision History**: Add a table to track content versions and allow rollbacks.
3. **Advanced Editor Features**: Implement search-and-replace and multi-file tabs in the Super Editor.

## 🚀 Running the Project
```bash
go run cmd/res-cms/main.go
```
The application listens on `:3000` by default.
