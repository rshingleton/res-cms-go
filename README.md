# ResCMS Go

ResCMS Go is a powerful, lightweight, and highly configurable Content Management System built with Go. It features a modern, reactive architecture, a robust dynamic theme engine, and a specialized administrative suite for full-site customization.

## 🚀 Features

- **High-Performance Go Backend**: Leveraging the speed and efficiency of Go for rapid content delivery.
- **Go Server-Side Rendering (SSR)**: Core themes are rendered on the server for maximum SEO, performance, and reliability.
- **Dynamic Theme Engine**: Swap entire site aesthetics in real-time. Supports ZIP theme uploads, extraction, and exporting.
- **Super Editor**: A unified administrative hub to edit theme files, manage global site customizations, and inject custom CSS/JS/HTML directly from the browser.
- **Standardized Editorial Experience**: Unified nomenclature and workflow for **Posts** and **Pages** using a customized Quill editor.
    - **Visual Image Resizing**: Professional resizing handles and display indicators.
    - **Optimized Media**: Images are uploaded to the server as files, avoiding database bloat.
- **Reactive Admin UI**: A fully-featured administrative dashboard built with Alpine.js and Axentix for a seamless management experience.
- **Axentix Integration**: The entire application (Admin & Public Themes) is built on the Axentix CSS framework, providing a component-based, stable layout engine.
- **Theme Hot Reloading**: Instant feedback for theme developers—see changes to templates and partials without restarting the server.
- **API-First Architecture**: Built-in JSON APIs for posts, pages, tags, and comments.
- **Multiple Built-in Themes**:
    - **Pixel Standard**: A premium 8-bit aesthetic for gaming, creative, or retro-styled blogs.
    - **Classic**: A clean, typography-focused editorial layout for journals and modern blogs.

## 🛠 Tech Stack

- **Backend**: Go (Golang) 1.22+
- **Database**: SQLite (via GORM)
- **Frontend Logic**: Alpine.js
- **CSS Frameworks**: Axentix (Unified for Admin and Themes)
- **Editors**: Monaco (for code), Quill (for content).

## 📦 Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/rshingleton/res-cms-go.git
   cd res-cms-go
   ```

2. Configuration (optional):
   Edit `rescms.yml` to change the listen port, database path, or production status.

3. Run the application:
   ```bash
   go run cmd/res-cms/main.go
   ```

4. Access the site:
   - **Public Site**: [http://localhost:3000](http://localhost:3000)
   - **Admin Dashboard**: [http://localhost:3000/manage](http://localhost:3000/manage)
   - **Default Credentials**: admin / admin

## 🎨 Theme Development

Themes are located in the `/themes` directory. Each theme requires:
- `theme.json`: Manifest file defining the theme name, version, and author.
- `layouts/index.html`: The home page / post feed template.
- `layouts/post.html`: The individual post view template.
- `partials/`: Reusable components like headers and footers.

Detailed specifications can be found in `docs/THEME_SPEC.md`.

## 📄 License

This project is licensed under the MIT License.
