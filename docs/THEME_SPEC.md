# ResCMS Theme Specification (Standard Theme Template - STT)

## 1. Overview
The ResCMS Theme Engine allows for modular, swappable public-facing "skins." Themes are stored in the `/themes` directory and are managed via the Admin Dashboard.

## 2. Directory Structure
A valid theme must follow this structure:
```text
/themes/[theme-name]/
├── theme.json           # Required: Metadata and configuration
├── layouts/             # Required: Page templates
│   ├── main.html        # Optional: Master layout wrapper
│   ├── index.html       # Required: Homepage / Post grid
│   ├── post.html        # Required: Single post view
│   └── 404.html         # Optional: Error page
├── partials/            # Required: Reusable UI fragments
│   ├── header.html
│   ├── footer.html
│   └── sidebar.html
└── assets/              # Optional: Static files
    ├── css/
    ├── js/
    └── img/
```

## 3. Theme Manifest (`theme.json`)
The manifest defines the theme's identity and visual tokens.
```json
{
  "name": "Theme Name",
  "version": "1.0.0",
  "author": "Author Name",
  "description": "Short description",
  "config": {
    "colors": {
      "primary": "#hex",
      "secondary": "#hex"
    },
    "typography": {
      "font_family": "..."
    }
  }
}
```

## 4. UI Standards (Pixel Art Engine)
For Pixel Art themes, the following standards apply:
- **Scaling**: All images and UI elements must use `image-rendering: pixelated;`.
- **Borders**: "Chunky" borders of 2px or 4px are preferred.
- **Typography**: Aliases for 8-bit fonts (e.g., "Press Start 2P") must be provided.
- **Grid**: Use CSS Grid for the layout to maintain "fixed-grid" alignment while being responsive.

## 5. Master Layout Logic
Themes can define a `layouts/main.html` which acts as the wrapper. Pages like `index.html` should define a `content` block:
```html
{{define "content"}}
  <!-- Page Content -->
{{end}}
```
The master layout should then include:
```html
{{block "content" .}}{{end}}
```
