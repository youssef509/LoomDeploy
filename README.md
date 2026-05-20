# LoomDeploy

**Self-hosted deployment platform for your VPS** — deploy any app from a Git repo in one click, with automatic SSL, custom domains, live logs, and health monitoring.

> Think Vercel or Coolify, but fully open source and running entirely on your own server.

---

## ✨ Features

- 🚀 **One-click deploys** from any Git repository (GitHub, GitLab, Bitbucket)
- 🔄 **Auto-deploy on push** via webhooks — push to a branch, deploy automatically
- 🐙 **GitHub App integration** — connect with one click, no PAT needed
- 🌐 **Custom domains** + auto-generated subdomains with **Let's Encrypt SSL** (via Traefik)
- 📋 **Live build logs** — streamed in real time while your app builds
- 🔁 **One-click rollback** — redeploy any previous commit instantly
- 🔐 **Environment variables** — per-project, stored securely
- 📊 **Resource limits** — CPU and memory limits per container
- ❤️ **Health checks** — auto-restart unhealthy containers
- 👥 **Multi-user** — admin / developer / viewer roles
- 📈 **Server stats** — CPU, RAM, disk usage at a glance
- 🐳 **Framework auto-detection** — Node.js, Python, Go, Ruby, PHP, Java — no Dockerfile needed

---

## 🚀 Quick Install

Run this on your VPS (Ubuntu 20.04+ recommended):

```bash
curl -fsSL https://raw.githubusercontent.com/youssef509/loomdeploy/main/deploy.sh | sudo bash
```

The script will:
1. Install Docker if not present
2. Ask for your domain and email (for SSL)
3. Auto-generate a secure JWT secret
4. Start Traefik (reverse proxy + SSL) and LoomDeploy

**After install:** visit your dashboard URL — you'll be guided through creating your admin account on the first visit.

### Requirements
- VPS with **Ubuntu 20.04+** (or any modern Debian-based Linux)
- A **domain name** pointed at your server's IP (`A` record)
- Ports **80** and **443** open

---

## 🔧 Configuration

All configuration lives in `/opt/loomdeploy/.env`:

| Variable | Description | Example |
|---|---|---|
| `APP_DOMAIN` | Domain for your LoomDeploy dashboard | `deploy.example.com` |
| `ACME_EMAIL` | Email for Let's Encrypt SSL certs | `admin@example.com` |
| `BASE_DOMAIN` | Base domain for auto-generated project URLs | `apps.example.com` |
| `JWT_SECRET` | Auto-generated — change if needed | *(auto-generated)* |
| `DB_PATH` | SQLite database path | `/var/lib/loomdeploy/data.db` |

After editing `.env`, restart with:
```bash
cd /opt/loomdeploy && docker compose up -d
```

---

## 🏗️ Architecture

```
Your VPS
├── Traefik (reverse proxy + SSL)
├── LoomDeploy Backend (Go + Gin + SQLite)
├── LoomDeploy Frontend (Nuxt 4)
└── Your deployed apps (Docker containers)
     └── Traefik routes traffic automatically
```

- **Backend:** Go + Gin + GORM (SQLite)
- **Frontend:** Nuxt 4 + Nuxt UI + Pinia
- **Proxy:** Traefik v3 (auto SSL, routing, Docker labels)
- **Database:** SQLite (single file, zero config)

---

## 📦 Self-Hosting / Development

```bash
git clone https://github.com/youssef509/loomdeploy.git
cd loomdeploy

# Copy and fill in your config
cp .env.example .env

# Start everything
docker compose -f docker-compose.traefik.yml up -d
docker compose up -d --build
```

---

## 🔒 Security

- Registration is **open only for the first user** (your admin account). After that, new users must be invited by an admin.
- All API routes (except login and setup) require a valid JWT token.
- Webhook endpoints use HMAC-SHA256 signature verification.
- Docker socket is proxied through nginx for isolation.

---

## 🗺️ Roadmap

See [ROADMAP.md](./_dev/ROADMAP.md) for planned features including:
- Uptime tracking & resource graphs
- Template marketplace (Postgres, Redis, MySQL one-click)
- Multi-server support
- CLI tool (`npx loomdeploy deploy`)
- Email invite system

---

## 🤝 Contributing

Contributions are welcome! Please open an issue first for major changes.

1. Fork the repo
2. Create your branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push and open a Pull Request

---

## 📄 License

[MIT](./LICENSE) — free to use, modify, and distribute.

---

<p align="center">Built with ❤️ — <a href="https://loomdeploy.com">loomdeploy.com</a></p>
