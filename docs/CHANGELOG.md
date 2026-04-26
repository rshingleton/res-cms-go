# Changelog

All notable changes to this project will be documented in this file.

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
- **Repository Hygiene**: Updated `.gitignore` to exclude `*.db`, SQLite journal files, and binary build artifacts (including the `res-cms` executable).

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