# ResCMS Go

ResCMS Go is a powerful, lightweight, and highly configurable Content Management System built with Go. It features a modern, reactive architecture and a robust dynamic theme engine.

## 🚀 Features

- **High-Performance Go Backend**: Leveraging the speed and efficiency of Go for rapid content delivery.
- **Dynamic Theme Engine**: Swap entire site aesthetics in real-time. Supports ZIP theme uploads, extraction, and exporting.
- **Reactive Admin UI**: A fully-featured administrative dashboard built with Alpine.js and Tailwind CSS for a seamless management experience.
- **Axentix Integration**: Public themes are built on the Axentix CSS framework, providing a component-based, stable layout engine.
- **Theme Hot Reloading**: Instant feedback for theme developers—see changes to templates and partials without restarting the server.
- **API-First Architecture**: Built-in JSON APIs for posts, categories, tags, and comments, enabling decoupled frontend implementations.
- **Multiple Built-in Themes**:
    - **Pixel Standard**: A premium 8-bit aesthetic for gaming, creative, or retro-styled blogs.
    - **Classic**: A clean, typography-focused editorial layout for journals and modern blogs.

## 🛠 Tech Stack

- **Backend**: Go (Golang)
- **Database**: SQLite (via GORM)
- **Frontend Logic**: Alpine.js
- **CSS Frameworks**: Axentix (Public Themes), Tailwind CSS (Admin Dashboard)
- **Templating**: Go `html/template` with extended FuncMaps

## 📦 Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/rshingleton/res-cms-go.git
   cd res-cms-go
   ```

2. Run the application:
   ```bash
   go run cmd/res-cms/main.go
   ```

3. Access the site:
   - **Public Site**: [http://localhost:3000](http://localhost:3000)
   - **Admin Dashboard**: [http://localhost:3000/manage](http://localhost:3000/manage)

## 🎨 Theme Development

Themes are located in the `/themes` directory. Each theme requires:
- `theme.json`: Manifest file defining the theme name, version, and author.
- `layouts/index.html`: The home page template.
- `layouts/post.html`: The individual post view template.
- `partials/`: Reusable components like headers and footers.

During development, ResCMS automatically reloads theme templates on every request, providing a "Hot Reload" experience.

## 📄 License

This project is licensed under the MIT License.
