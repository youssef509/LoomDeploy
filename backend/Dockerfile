# ── Build stage ──────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: glebarez/sqlite is pure Go — no CGO needed
# Building on linux/amd64 so docker_linux.go + system_linux.go are included automatically
# -mod=mod: auto-reconciles go.sum without running go mod tidy (which breaks Docker SDK errors.Is)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -ldflags="-s -w" -o loomdeploy-backend .

# ── Runtime stage ─────────────────────────────────────────
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata git

WORKDIR /app

COPY --from=builder /app/loomdeploy-backend .

# Data directory for SQLite DB
RUN mkdir -p /var/lib/loomdeploy
VOLUME ["/var/lib/loomdeploy"]

EXPOSE 8080

CMD ["./loomdeploy-backend"]
