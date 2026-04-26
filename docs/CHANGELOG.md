# Changelog

All notable changes to this project will be documented in this file.

## [2.3.0] - 2026-04-26 (Multi-Database Support)

### Added
- **Multi-Database Backend Support**:
    - Added support for **MySQL** and **PostgreSQL** alongside SQLite.
    - Integrated `gorm.io/driver/mysql` and `gorm.io/driver/postgres`.
    - Automated **Database Bootstrapping**: The application now attempts to create the target database automatically if it does not exist (requires appropriate user permissions like `CREATEDB` in Postgres).
- **Comprehensive Database Documentation**:
    - Created `docs/DATABASE_SETUP.md` with detailed instructions for SQLite, MySQL, and PostgreSQL setup.
    - Included specific troubleshooting for PostgreSQL 15+ schema permissions.

### Changed
- **Refactored Database Initialization**:
    - Updated `internal/db` to use a unified `config.Config` object for initialization.
    - Standardized DSN parsing for URI and Key-Value formats.
- **Configuration Schema**:
    - Updated `rescms.yml` and `internal/config` to support a structured `database` configuration block.
    - Maintained backward compatibility for legacy `sqlite_dsn` settings.

### Fixed
- **Postgres DSN Parsing**: Resolved issues where URI-based DSNs were not correctly identifying the database name during bootstrapping.
- **Idempotent Seeding**: Refactored database seeding to use `FirstOrCreate` to prevent unique constraint violations during initialization of existing databases.

## [2.2.0] - 2026-04-26 (Public Site Fixes & SSR Migration)


### Added
- **Server-Side Rendering (SSR) Migration**:
    - Refactored `Classic` and `Pixel Standard` themes from a broken client-side Alpine.js fetch pattern to native Go Server-Side Rendering.
    - This ensures posts and sidebar content are rendered immediately, improving SEO and site reliability.
- **Enhanced Site Layout**:
    - Widened the main site container to 1400px (95% width) for better usage of modern screen space.
    - Right-justified post dates in the "Recent Posts" sidebar for a more structured, professional appearance.

### Changed
- **UI Cleanup**:
    - Removed redundant "Posts" feed from the bottom of the home page.
    - Removed the static "Main" category label from sidebar posts to reduce clutter.
    - Standardized date formatting across themes.

### Fixed
- **Critical Sidebar Date Bug**: Resolved a backend bug where `created_at` was excluded from sidebar database queries, causing posts to show "Jan 01" (the zero-value date).
- **Pixel Theme Timestamp Bug**: Fixed a literal "JAN" string in the Pixel Standard theme's date format that hardcoded all months to January.
- **Monaco Editor Stability**: Finalized the "nuclear" flexbox layout fixes to prevent the editor from collapsing to 1px height.

### Housekeeping
- **Repository Hygiene**: Updated `.gitignore` to exclude `*.db`, SQLite journal files, and binary build artifacts.
- **Remnant Cleanup**: Removed obsolete `migrate-categories` command and various development/test remnants (`test_template.go`, `editor_test.html`, `cookie.txt`, `public/js/app.js`, etc.).
- **Untracked Binaries**: Removed local build artifacts (the `res-cms` binary) and database files from the repository's tracked index.

## [2.1.0] - 2026-04-26 (Editor Stability & Monaco Fixes)

### Added
- **Unified Super Editor Stabilized**:
    - Resolved critical rendering issues where the Monaco editor would collapse to 1px height/width.
    - Standardized layout using hardcoded flex properties and `min-height: 0` to bypass framework conflicts.
    - Integrated Alpine.js reactive state for file saving and toggle status.
    - Implemented `Ctrl+S` keybind bridge between Monaco and Alpine.js via Custom Events.
    - Fixed theme file loading by resolving Go template scoping issues.
- **Improved Sidebar UX**:
    - Standardized active selection styles for consistent readability (blue background, white text).
    - Added file-type specific icons (JS, CSS, HTML) to the theme library browser.
    - Refactored toggle switches for better text visibility.

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