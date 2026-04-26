# ResCMS Theme Specification (Standard Theme Template - STT)

## 1. Overview
The ResCMS Theme Engine allows for modular, swappable public-facing "skins." Themes are stored in the `/themes` directory and are managed via the Admin Dashboard's Super Editor.

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

## 3. Global Injections
The CMS automatically provides several variables for global customization. Themes must include these variables in their master layout (usually `layouts/main.html`) to support user customizations.

- **`{{.res_custom_css_style}}`**: Injected into the `<head>`. Automatically wrapped in `<style>` tags.
- **`{{.res_custom_js_script}}`**: Injected into the `<head>` (or before `</body>`). Automatically wrapped in `<script>` tags.
- **`{{.res_custom_header_html_parsed}}`**: Raw HTML injected at the end of the `<head>`.
- **`{{.res_custom_footer_html_parsed}}`**: Raw HTML injected before the closing `</body>` tag.

## 4. Manifest (`theme.json`)
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

## 5. Master Layout Logic
Themes should define a master layout wrapper. Individual pages define a `content` block which is then rendered by the master layout.

Example `index.html`:
```html
{{define "content"}}
  <div class="posts">...</div>
{{end}}
```

Example `layouts/main.html`:
```html
<!DOCTYPE html>
<html>
<head>
  <title>{{.res_blog_name}}</title>
  {{.res_custom_css_style}}
  {{.res_custom_header_html_parsed}}
</head>
<body>
  {{template "partials/header.html" .}}
  {{block "content" .}}{{end}}
  {{template "partials/footer.html" .}}
  {{.res_custom_js_script}}
  {{.res_custom_footer_html_parsed}}
</body>
</html>
```
