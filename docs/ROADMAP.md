# Project Roadmap

## Phase 1: Stability & Foundation (Completed)
- [x] Migrate from Perl to Go Backend
- [x] Rename "Taxonomy" to "Pages"
- [x] Rename "Appearance" to "Theme"
- [x] Stabilize Quill editor content retrieval
- [x] Implement robust Image Resizing for Quill
- [x] Build custom server-side Image Upload adapter
- [x] Standardize editorial styling across all themes (Classic & Pixel)

## Phase 2: Theme Management (In Progress)
- [x] **Theme Operations**:
    - [x] **Copy Theme**: Duplicate any installed theme from the UI.
    - [ ] **Export Theme**: Package a theme as a ZIP for download.
    - [ ] **Upload Theme**: Drag-and-drop ZIP upload.
- [ ] **Integrated Theme Editor**:
    - Full-screen code editor (Monaco-based).
    - Support for HTML, CSS, JS, and JSON.
    - Recursive file explorer for the `themes/` directory.

## Phase 3: Media & Assets
- [ ] Dedicated Media Library in the admin panel.
- [ ] Automatic image optimization on upload (WebP conversion).
- [ ] Drag-and-drop image placement from library to editor.

## Phase 4: Core Extensions
- [ ] Plugin architecture for third-party widgets and hooks.
- [ ] Multi-user permissions and role-based access control.
- [ ] Revision history and content versioning.