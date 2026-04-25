# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] - 2026-04-25 (Go Migration & Stability)

### Added
- **New Go Backend**: Complete rewrite of the CMS engine in Go for high performance and concurrency.
- **Quill-based Editorial Suite**:
    - Custom Image Resizer with corner handles and size indicators.
    - Server-side Upload Adapter: Images are now stored as files in `/public/uploads/` instead of Base64 strings.
    - Text-sync engine: Ensures editor content is consistently saved and retrieved without data loss.
- **Theme Editor & Operations**:
    - Added "Copy Theme" functionality to duplicate themes from the UI.
    - Integrated a Monaco-based full-screen code editor for theme files.
- **Unified Classification**: Renamed "Taxonomy" to "Pages" and unified the `models.Page` model for both pages and categories.
- **Robust Notifications**: Implemented a stable toast notification system using Axentix.

### Changed
- **Renamed Navigation**: "Appearance" is now "Theme" and "Taxonomy" is now "Pages".
- **API v2**: Optimized all JSON endpoints to handle the unified Page/Category model.
- **Admin Layout**: Refined the admin sidebar and dashboard metrics for a more professional feel.

### Fixed
- **Alpine.js Crash**: Resolved `Unexpected number` syntax errors by properly JSON-encoding Go models for the frontend.
- **Image Persistence**: Fixed 403 Forbidden errors on image uploads by adding proper authentication middleware to the API.
- **Template Errors**: Resolved `hasSuffix` function missing errors in Go templates.

## [1.0.0] - 2026-04-18 (Legacy Implementation)
- Original Perl/Mojolicious implementation (See archival docs for details).