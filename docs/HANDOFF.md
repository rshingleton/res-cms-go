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
- **Theme Engine & SSR**: 
    - **SSR Migration**: Public themes (`Classic`, `Pixel Standard`) use Go Server-Side Rendering for all core content.
    - Avoid using client-side `fetch` for basic post lists to ensure SEO and performance.
- **Media**: Images uploaded via the editor are saved to `/public/uploads/`.

## 🔑 Site Refinements (v2.2.0)
- **Widened Layout**: The main site container is set to 1400px to maximize screen usage.
- **Sidebar Clarity**: Dates are right-justified and redundant category labels have been removed.
- **Timestamp Accuracy**: Fixed backend queries and template formats to ensure correct post dates are displayed.

## 🗺 Critical Path for Next Development
1. **Media Library**: Build a dedicated file browser in the admin panel to manage existing uploads.
2. **Advanced Editor Features**: Implement search-and-replace and multi-file tabs in the Super Editor.
3. **Draft Workflow**: Implement a "Preview" mode for draft posts before they go public.

## 🚀 Running the Project
```bash
go run cmd/res-cms/main.go
```
The application listens on `:3000` by default.
