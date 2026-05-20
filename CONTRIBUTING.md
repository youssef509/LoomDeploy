# Contributing to LoomDeploy

Thanks for your interest in contributing! Below are everything you need to get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Commit Convention](#commit-convention)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)

---

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](./CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

## Getting Started

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/YOUR_USERNAME/loomdeploy.git`
3. Create a **new branch**: `git checkout -b feat/your-feature-name`
4. Make your changes, commit, and push
5. Open a **Pull Request** against `main`

For **bug fixes**, branch from `main` and name it `fix/issue-description`.  
For **features**, branch from `main` and name it `feat/feature-name`.

---

## Development Setup

### Requirements

- **Go 1.22+** (for the backend)
- **Node.js 20+** (for the frontend)
- **Docker + Docker Compose** (for running locally)
- A Linux or WSL environment is recommended

### Backend (Go + Gin)

```bash
cd backend
go mod tidy
go run main.go
```

The API will start on `http://localhost:8080`.

### Frontend (Nuxt 4)

```bash
cd frontend
npm install
npm run dev
```

The dashboard will start on `http://localhost:3000`.  
API requests proxy automatically to `localhost:8080`.

### Full stack (Docker Compose)

```bash
cp .env.example .env
# Edit .env with your domain and settings
docker compose -f docker-compose.traefik.yml up -d
docker compose up -d --build
```

---

## Project Structure

```
loomdeploy/
├── backend/                  # Go backend (Gin, GORM, SQLite)
│   ├── internal/
│   │   ├── database/         # SQLite + GORM setup
│   │   ├── docker/           # Docker SDK + auto-detection
│   │   ├── handlers/         # HTTP route handlers
│   │   ├── healthcheck/      # Container health loop
│   │   ├── logbroker/        # SSE build log streaming
│   │   ├── middleware/       # JWT auth middleware
│   │   └── models/           # GORM models
│   └── main.go
├── frontend/                 # Nuxt 4 dashboard
│   └── app/
│       ├── components/
│       ├── composables/
│       ├── middleware/
│       ├── pages/
│       └── stores/
├── docker-compose.yml
├── docker-compose.traefik.yml
└── deploy.sh                 # One-command installer
```

---

## Making Changes

- **Keep changes focused** — one feature or fix per PR
- **Follow existing code style** — Go: `gofmt`, Frontend: ESLint
- **Test your changes** before opening a PR
- **Update documentation** if you're changing behavior

### Backend style

- Use `gofmt` to format Go code
- Follow the existing handler/middleware/model separation
- Return errors through Gin's `c.JSON(...)` pattern

### Frontend style

- Use Nuxt UI components where possible
- Keep composables for API calls (`useApi.ts`)
- Use Pinia stores for global state

---

## Commit Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add GitHub App webhook validation
fix: correct JWT expiry calculation
docs: update installation steps
chore: bump Go dependencies
refactor: extract deployment logic to service layer
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `style`

---

## Pull Request Process

1. Ensure your branch is up to date with `main`
2. Fill in the PR template completely
3. Link any related issues (`Closes #123`)
4. A maintainer will review within a few days
5. Address review comments and push updates to the same branch
6. Once approved, a maintainer will merge using **Squash and Merge**

---

## Reporting Bugs

Use the [Bug Report](./.github/ISSUE_TEMPLATE/bug_report.yml) issue template.  
Include reproduction steps, expected vs. actual behavior, and environment details.

---

## Requesting Features

Use the [Feature Request](./.github/ISSUE_TEMPLATE/feature_request.yml) issue template.  
Explain the use case and why it would benefit LoomDeploy users.

---

## Security Vulnerabilities

**Do not open a public issue for security vulnerabilities.**  
Please read [SECURITY.md](./SECURITY.md) and follow the responsible disclosure process.
