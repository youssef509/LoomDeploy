# ── Build stage ──────────────────────────────────────────
FROM node:20-alpine AS builder

# Enable corepack so pnpm@10 (declared in package.json) is used automatically
RUN corepack enable

WORKDIR /app

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --no-frozen-lockfile

COPY . .

# BACKEND_URL is baked in at build time so the Nitro proxy target is correct
ARG BACKEND_URL=http://backend:8080
ENV BACKEND_URL=$BACKEND_URL

RUN pnpm run build

# ── Runtime stage ─────────────────────────────────────────
FROM node:20-alpine

WORKDIR /app

COPY --from=builder /app/.output ./

ENV HOST=0.0.0.0
ENV PORT=3000
ENV NODE_ENV=production

EXPOSE 3000

CMD ["node", "server/index.mjs"]
