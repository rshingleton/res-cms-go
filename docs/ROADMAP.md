# Project Roadmap

## Phase 1: Stability & Foundation (Completed)
- [x] Migrate from Perl to Go Backend
- [x] Rename "Taxonomy" to "Pages"
- [x] Rename "Appearance" to "Theme"
- [x] Stabilize Quill editor content retrieval
- [x] Implement robust Image Resizing for Quill
- [x] Build custom server-side Image Upload adapter
- [x] Standardize editorial styling across all themes (Classic & Pixel)

## Phase 2: Theme Management & Customization (Completed)
- [x] **Theme Operations**:
    - [x] **Duplicate Theme**: Rename and copy any installed theme via UI modal.
    - [x] **Export Theme**: Package a theme as a ZIP for download.
    - [x] **Upload Theme**: Drag-and-drop ZIP upload.
- [x] **Unified Super Editor**:
    - [x] **Cross-Theme Selection**: Browse and edit files from any theme in one view.
    - [x] **Global Injections**: Manage custom CSS, JS, and HTML with auto-wrapping.
    - [x] **Enable/Disable Toggles**: Instantly control injection visibility via database flags.
    - [x] **Monaco-based**: Full-screen code editor with persistent toolbar (v2.2.0 Stabilized).

## Phase 3: Media & Assets (In Progress)
- [ ] Dedicated Media Library in the admin panel.
- [ ] Automatic image optimization on upload (WebP conversion).
- [ ] Drag-and-drop image placement from library to editor.

## Phase 4: Core Extensions
- [ ] Plugin architecture for third-party widgets and hooks.
- [ ] Multi-user permissions and role-based access control.
- [ ] Revision history and content versioning.
