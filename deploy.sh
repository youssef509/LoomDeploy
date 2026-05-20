#!/bin/bash
# LoomDeploy — One-command installer
# Usage: curl -fsSL https://raw.githubusercontent.com/youssef509/loomdeploy/main/deploy.sh | sudo bash
set -e

REPO="https://github.com/youssef509/loomdeploy.git"
INSTALL_DIR="/opt/loomdeploy"
BOLD="\033[1m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
RESET="\033[0m"

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║          LoomDeploy Installer            ║${RESET}"
echo -e "${BOLD}║   Self-hosted deployment platform        ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════╝${RESET}"
echo ""

# ── Must run as root ─────────────────────────────────────
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo bash deploy.sh"
  exit 1
fi

# ── 1. Install Docker if missing ─────────────────────────
echo -e "${CYAN}[1/6]${RESET} Checking Docker..."
if ! command -v docker &>/dev/null; then
  echo "      Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker --now
else
  echo "      Docker $(docker --version | cut -d' ' -f3 | tr -d ',') — already installed"
fi

# ── 2. Install git if missing ────────────────────────────
echo -e "${CYAN}[2/6]${RESET} Checking git..."
if ! command -v git &>/dev/null; then
  echo "      Installing git..."
  apt-get update -qq && apt-get install -y -qq git
else
  echo "      git $(git --version | cut -d' ' -f3) — already installed"
fi

# ── 3. Clone or update repository ───────────────────────
echo -e "${CYAN}[3/6]${RESET} Fetching LoomDeploy..."
if [ -d "$INSTALL_DIR/.git" ]; then
  git -C "$INSTALL_DIR" pull --ff-only
  echo "      Updated to latest version"
else
  git clone "$REPO" "$INSTALL_DIR"
  echo "      Cloned to $INSTALL_DIR"
fi
cd "$INSTALL_DIR"

# ── 4. Configure environment ─────────────────────────────
echo -e "${CYAN}[4/6]${RESET} Configuring environment..."

if [ ! -f "$INSTALL_DIR/.env" ]; then
  echo ""
  echo -e "${BOLD}  Let's configure your LoomDeploy instance:${RESET}"
  echo ""

  # Dashboard domain
  read -rp "  → Dashboard domain (e.g. deploy.example.com): " APP_DOMAIN
  while [ -z "$APP_DOMAIN" ]; do
    read -rp "  → Dashboard domain cannot be empty: " APP_DOMAIN
  done

  # Email for Let's Encrypt
  read -rp "  → Email for SSL certificates (Let's Encrypt): " ACME_EMAIL
  while [ -z "$ACME_EMAIL" ]; do
    read -rp "  → Email cannot be empty: " ACME_EMAIL
  done

  # Base domain for project auto-domains
  read -rp "  → Base domain for projects (e.g. apps.example.com) [optional, press Enter to skip]: " BASE_DOMAIN_INPUT

  # Auto-generate a secure JWT secret
  JWT_SECRET=$(openssl rand -hex 32)

  # Write .env
  cat > "$INSTALL_DIR/.env" <<EOF
# ── LoomDeploy Configuration ──────────────────────────────
APP_DOMAIN=${APP_DOMAIN}
ACME_EMAIL=${ACME_EMAIL}
BASE_DOMAIN=${BASE_DOMAIN_INPUT}
JWT_SECRET=${JWT_SECRET}
DB_PATH=/var/lib/loomdeploy/data.db
EOF

  echo ""
  echo -e "      ${GREEN}✓${RESET} .env created with auto-generated JWT secret"
else
  # Load existing values for display
  APP_DOMAIN=$(grep '^APP_DOMAIN' .env | cut -d= -f2)
  echo "      Using existing .env (APP_DOMAIN=${APP_DOMAIN})"
fi

# ── 5. Create Docker network ─────────────────────────────
echo -e "${CYAN}[5/6]${RESET} Ensuring Docker network exists..."
docker network inspect paas_network &>/dev/null || docker network create paas_network

# ── 6. Start Traefik + LoomDeploy ────────────────────────
echo -e "${CYAN}[6/6]${RESET} Starting LoomDeploy (first run may take a few minutes)..."
echo ""

docker compose -f docker-compose.traefik.yml up -d
docker compose up -d --build

APP_DOMAIN=$(grep '^APP_DOMAIN' .env | cut -d= -f2)
echo ""
echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${GREEN}║       LoomDeploy is ready! 🚀            ║${RESET}"
echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════╝${RESET}"
echo ""
echo -e "  ${BOLD}Dashboard:${RESET}  https://${APP_DOMAIN}"
echo ""
echo -e "  ${YELLOW}First time?${RESET} Visit the dashboard to create your admin account."
echo -e "  ${YELLOW}Update:${RESET}     Run this script again to pull the latest version."
echo ""
