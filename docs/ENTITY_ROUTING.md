# ResCMS Entity Routing Documentation

## Core Handlers

| Entity | Package | Source File |
|--------|---------|-------------|
| Root | `handlers` | `internal/handlers/root.go` |
| Auth | `handlers` | `internal/handlers/auth.go` |
| Admin | `handlers` | `internal/handlers/admin.go` |
| API | `handlers` | `internal/handlers/api.go` |

---

## Routes by Entity

### Public Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | `/` | `IndexHandler` |
| GET | `/page/{page}` | `IndexHandler` |
| GET | `/post/{slug}` | `PostHandler` |
| POST | `/comment/add` | `AddCommentHandler` |
| GET | `/posts/page/{page}` | `PostsByPageHandler` |
| GET | `/posts/tag/{tag}` | `PostsByTagHandler` |
| GET | `/access/login` | `LoginFormHandler` |
| POST | `/access/login` | `LoginHandler` |
| GET | `/access/logout` | `LogoutHandler` |

### Management Routes (`/manage` prefix)

| Method | Path | Handler |
|--------|------|---------|
| GET | `/manage` | `AdminIndexHandler` |
| GET | `/manage/posts` | `AdminListPostsHandler` |
| GET | `/manage/themes` | `AdminListThemesHandler` |
| POST | `/manage/themes/upload` | `AdminUploadThemeHandler` |
| GET | `/manage/themes/activate/{name}` | `AdminActivateThemeHandler` |
| GET | `/manage/themes/export/{name}` | `AdminExportThemeHandler` |

### API Routes (`/api/v1/` prefix)

| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/posts` | `GetPostsAPI` |
| POST | `/api/v1/contact` | `ContactAPI` |

---

## Database Schema (GORM)

| Table | Model | Description |
|-------|-------|-------------|
| `users` | `User` | Authentication and profile data |
| `posts` | `Post` | Blog posts and static pages |
| `pages` | `Page` | Hierarchical content grouping |
| `tags` | `Tag` | Flat content tagging |
| `comments` | `Comment` | User-submitted feedback |
| `site_settings` | `SiteSetting` | Application configuration |

---

## Theme Standard
For detailed routing within themes, see [THEME_SPEC.md](./THEME_SPEC.md).
