# ResCMS Go Roadmap

## Phase 1: Core Porting (Completed)
- [x] Initial Go project structure with GORM (SQLite)
- [x] API-first backend for Posts, Categories, Tags, and Comments
- [x] Unified Admin Dashboard with Alpine.js
- [x] Basic Auth and Session Management
- [x] Tailwind CSS & Pines UI integration

## Phase 2: Refactoring & Theme Engine (Completed)
- [x] **Modular Theme Engine**: Dynamic loading of public skins
- [x] **Standard Theme Template (STT)**: Formalized directory structure and manifest
- [x] **Pixel Art Theme**: High-fidelity 8-bit aesthetic theme
- [x] **Classic Theme Migration**: Ported legacy modern design into the new engine
- [x] **Clean Room Architecture**: Relocated Admin UI to `internal/ui` and removed root `templates/` folder
- [x] **Theme Portability**: Zip upload and export functionality

## Phase 3: Modernization & Testing (In Progress)
- [x] **Unit Testing**: Core logic and Theme Engine tests
- [ ] **Integration Testing**: End-to-end API and UI tests
- [ ] **Advanced Asset Management**: Pixel-perfect gallery and media optimization
- [ ] **Hardened Security**: CSRF protection and rate limiting
- [ ] **Containerization**: Docker production builds

## Phase 4: Launch
- [ ] Production Deployment Guide
- [ ] Performance Benchmarking
- [ ] CI/CD Pipelines (GitHub Actions)