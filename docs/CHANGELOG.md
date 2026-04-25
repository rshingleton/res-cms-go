# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased] - 2026-04-19

### Added
- Implemented `asset_version` Mojolicious helper for intelligent browser caching of CSS/JS assets
- Added automated Tailwind CSS rebuilds when `.ep` templates are modified in development mode
- Installed Playwright and added automated browser layout verification tests
- Implemented separate PID files for development (`rescms-development.pid`) and production (`rescms-production.pid`)
- Added support for `HYPNOTOAD_LISTEN` environment variable in `rcms` script for flexible port configuration
- Added multiple public templates (Default, Compact, Masonry) - selectable via Admin Settings

### Changed
- Refactored frontend layout to a wide 4-column grid (75% content, 25% sidebar) using `max-w-screen-2xl`
- Updated Typography (`prose`) classes to `max-w-none` to allow better use of horizontal space
- Consolidated dark mode logic using Alpine.js reactive bindings (`:class="{ 'dark': isDark }"`)
- Improved dark mode consistency by applying `bg-gray-900` directly to the `body` element
- Enhanced `rcms` CLI script to properly handle `--dev` flag and `hypnotoad` production execution
- Updated `ResCMS::Plugin::Tailwind` to monitor template changes for CSS rebuilds

### Fixed
- **Resolved Sidebar Flash/Visibility**: Fixed admin sidebar visibility and toggle functionality
- **Fixed Theme Toggle**: Dark mode now correctly persists and applies to the entire document
- **SQL Ambiguity**: Fixed "ambiguous column" errors in admin queries by prefixing `status` columns
- **Admin Template Errors**: Fixed missing variables in Posts, Comments, Categories, and Users list templates
- **Post Status Bug**: Fixed issue where new posts were hardcoded to 'draft' status
- **Category Processing**: Corrected handling of array references for `every_param` calls in categories

### Removed
- Removed legacy dark mode CSS overrides in favor of native Tailwind dark mode support


## [1.0.0] - 2026-04-18

### Added
- Initial Mojolicious application structure for ResCMS
- Core controllers: `Root`, `Auth`, `Admin`, and `Admin::Post`
- Integration with Mojo::SQLite for database
- Base layouts and theme partials in `.ep` format
- `rescms.yml` configuration file
- `script/rescms` execution entry point
- Ported static assets from legacy PearlBee
- Initial Roadmap and Changelog in `docs/`

### Changed
- Migrated web backend from Dancer2 to Mojolicious
- Transitioned templates from Template Toolkit (`.tt`) to Embedded Perl (`.ep`)