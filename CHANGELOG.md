# Changelog

All notable changes to LoomDeploy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

### Added
- Initial public alpha release

---

## [0.1.0] — 2025

### Added
- One-command VPS installer (`deploy.sh`) with Docker, Traefik, and Let's Encrypt
- Go + Gin backend with SQLite database (GORM)
- Nuxt 4 + Nuxt UI dashboard frontend
- JWT authentication with role-based access (Admin / Developer / Viewer)
- Project management — create, edit, delete projects from Git repositories
- One-click deploy from any Git repo with framework auto-detection (Node.js, Python, Go, Ruby, PHP, Java)
- GitHub App integration — connect with one click, browse repos
- Personal Access Token support for private repos
- Live build log streaming via Server-Sent Events (SSE)
- Container runtime log streaming via SSE
- One-click rollback to any previous deployment
- Per-project environment variables
- Webhook auto-deploy on Git push
- Container actions (start, stop, restart)
- Server stats dashboard (CPU, memory, disk, uptime)
- Multi-user support with invite system
- Health check loop with auto-restart for unhealthy containers
- Auto-generated subdomains with Traefik routing
- Automatic HTTPS with Let's Encrypt (ACME HTTP-01)
