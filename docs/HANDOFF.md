# Handoff Document

## Project Overview
ResCMS Go is a lightweight, page-based CMS built for speed and flexibility. It replaces the legacy Perl implementation with a modern Go backend and a reactive admin interface.

## Current Technical Stack
- **Backend**: Go 1.22+
- **Database**: SQLite (`data/rescms.db`)
- **Admin UI**: Alpine.js, Axentix, SortableJS, CKEditor 5 (Posts), Quill (Pages)
- **Public UI**: Axentix CSS framework

## Key Operations
- **Run**: `go run cmd/res-cms/main.go`
- **Default Port**: `:3009`
- **Admin URL**: `/manage`

## Critical Path for Next Dev
1. **Quill Retrieval**: We are currently experiencing issues where saved content sometimes fails to load into the Quill editor on the Pages edit screen. We've moved to a textarea-based transfer method, but this needs verification.
2. **Image Resizing**: The image resize module for Quill is integrated via CDN but has activation issues. Ensure `window.Quill` is defined before module load.
3. **Theme Editor**: The next major feature is an in-browser theme editor for JS and SCSS files.

## Database Schema
The system uses a unified `pages` table for both standalone pages and category-like taxonomies. 
`SortOrder` in the `pages` table determines the site-wide navigation order.