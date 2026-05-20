package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectFramework scans dir for well-known framework marker files and returns
// a short framework identifier, or "" if nothing is recognised.
func DetectFramework(dir string) string {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	switch {
	case has("package.json"):
		return "nodejs"
	case has("requirements.txt") || has("pyproject.toml"):
		return "python"
	case has("go.mod"):
		return "golang"
	case has("Gemfile"):
		return "ruby"
	case has("composer.json"):
		return "php"
	case has("pom.xml"):
		return "java-maven"
	case has("build.gradle") || has("build.gradle.kts"):
		return "java-gradle"
	default:
		return ""
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func fileAt(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func readStr(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return string(b)
}

// toCMD converts a shell command to a Dockerfile CMD directive.
// Uses exec form for simple commands, shell form when env vars / pipes present.
func toCMD(cmd string) string {
	if strings.ContainsAny(cmd, "$|&;") {
		return `["sh", "-c", "` + cmd + `"]`
	}
	parts := strings.Fields(cmd)
	q := make([]string, len(parts))
	for i, p := range parts {
		q[i] = `"` + p + `"`
	}
	return "[" + strings.Join(q, ", ") + "]"
}

// ── Node.js deep detection ────────────────────────────────────────────────────

type pkgJSON struct {
	Main            string            `json:"main"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Prisma          struct {
		Schema string `json:"schema"`
	} `json:"prisma"`
}

type nodeInfo struct {
	base       string // Docker base image
	install    string // install command
	prismaGen  string // prisma generate command (empty if not used)
	buildCmd   string // build command (empty if no build script)
	pruneCmd   string // post-build prune (empty if not applicable)
	startCmd   string // container start command
}

func detectNode(dir string) nodeInfo {
	n := nodeInfo{}

	// Package manager from lockfile
	switch {
	case fileAt(dir, "bun.lockb"):
		n.base = "oven/bun:1-alpine"
		n.install = "bun install --frozen-lockfile"
		n.startCmd = "bun run start"
	case fileAt(dir, "pnpm-lock.yaml"):
		n.base = "node:20-alpine"
		n.install = "corepack enable pnpm && pnpm install --frozen-lockfile"
		n.pruneCmd = "pnpm prune --prod 2>/dev/null || true"
		n.startCmd = "pnpm start"
	case fileAt(dir, "yarn.lock"):
		n.base = "node:20-alpine"
		n.install = "yarn install --frozen-lockfile"
		n.pruneCmd = "yarn install --production --ignore-scripts 2>/dev/null || true"
		n.startCmd = "yarn start"
	case fileAt(dir, "package-lock.json"):
		n.base = "node:20-alpine"
		n.install = "npm ci"
		n.pruneCmd = "npm prune --omit=dev 2>/dev/null || true"
		n.startCmd = "npm start"
	default:
		n.base = "node:20-alpine"
		n.install = "npm install"
		n.pruneCmd = "npm prune --omit=dev 2>/dev/null || true"
		n.startCmd = "npm start"
	}

	// Parse package.json
	var pkg pkgJSON
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		_ = json.Unmarshal(data, &pkg)
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// Prisma: resolve schema path from multiple sources
	if _, hasPrisma := allDeps["prisma"]; hasPrisma {
		schema := pkg.Prisma.Schema // custom path in package.json { prisma: { schema: "..." } }
		if schema == "" {
			for _, cand := range []string{
				"prisma/schema.prisma",
				"schema.prisma",
				"db/schema.prisma",
				"database/schema.prisma",
				"src/prisma/schema.prisma",
			} {
				if fileAt(dir, cand) {
					schema = cand
					break
				}
			}
		}
		if schema != "" {
			n.prismaGen = fmt.Sprintf("npx prisma generate --schema=%s", schema)
		} else {
			n.prismaGen = "npx prisma generate"
		}
	}

	// Build script
	pkgMgr := "npm run"
	switch {
	case strings.Contains(n.install, "bun"):
		pkgMgr = "bun run"
	case strings.Contains(n.install, "pnpm"):
		pkgMgr = "pnpm run"
	case strings.Contains(n.install, "yarn"):
		pkgMgr = "yarn"
	}
	if _, ok := pkg.Scripts["build"]; ok {
		n.buildCmd = pkgMgr + " build"
	}

	// Start command: specific framework > scripts.start > main > entry file scan
	_, hasNext := allDeps["next"]
	_, hasCRA := allDeps["react-scripts"]
	switch {
	case hasNext:
		if _, ok := pkg.Scripts["start"]; ok {
			n.startCmd = pkgMgr + " start"
		} else {
			n.startCmd = "npx next start -p $PORT"
		}
	case hasCRA:
		n.startCmd = "npx serve -s build -l $PORT"
	case pkg.Scripts["start"] != "":
		n.startCmd = pkgMgr + " start"
	case pkg.Main != "":
		n.startCmd = "node " + pkg.Main
	default:
		for _, cand := range []string{
			"index.js", "server.js", "app.js", "main.js",
			"src/index.js", "src/server.js", "src/app.js", "src/main.js",
			"dist/index.js", "dist/server.js",
		} {
			if fileAt(dir, cand) {
				n.startCmd = "node " + cand
				break
			}
		}
	}

	return n
}

// ── Python deep detection ─────────────────────────────────────────────────────

type pythonInfo struct {
	install  string
	startCmd string
}

func detectPython(dir string) pythonInfo {
	py := pythonInfo{}

	if fileAt(dir, "requirements.txt") {
		py.install = "pip install --no-cache-dir -r requirements.txt"
	} else {
		py.install = "pip install --no-cache-dir ."
	}

	combined := strings.ToLower(
		readStr(dir, "requirements.txt") +
			readStr(dir, "pyproject.toml") +
			readStr(dir, "setup.py") +
			readStr(dir, "setup.cfg"),
	)

	switch {
	case strings.Contains(combined, "django"):
		if fileAt(dir, "manage.py") {
			py.startCmd = "python manage.py runserver 0.0.0.0:$PORT"
		} else {
			py.startCmd = "python -m django runserver 0.0.0.0:$PORT"
		}
	case strings.Contains(combined, "fastapi") || strings.Contains(combined, "uvicorn"):
		for _, cand := range []string{"main.py", "app.py", "src/main.py", "src/app.py", "api/main.py", "api/app.py"} {
			if fileAt(dir, cand) {
				mod := strings.TrimSuffix(strings.ReplaceAll(cand, "/", "."), ".py")
				py.startCmd = fmt.Sprintf("uvicorn %s:app --host 0.0.0.0 --port $PORT", mod)
				break
			}
		}
		if py.startCmd == "" {
			py.startCmd = "uvicorn main:app --host 0.0.0.0 --port $PORT"
		}
	case strings.Contains(combined, "flask") || strings.Contains(combined, "gunicorn"):
		for _, cand := range []string{"app.py", "main.py", "run.py", "wsgi.py", "application.py"} {
			if fileAt(dir, cand) {
				mod := strings.TrimSuffix(cand, ".py")
				py.startCmd = fmt.Sprintf("gunicorn %s:app --bind 0.0.0.0:$PORT", mod)
				break
			}
		}
		if py.startCmd == "" {
			py.startCmd = "gunicorn app:app --bind 0.0.0.0:$PORT"
		}
	default:
		for _, cand := range []string{"main.py", "app.py", "run.py", "server.py", "manage.py", "wsgi.py"} {
			if fileAt(dir, cand) {
				py.startCmd = "python " + cand
				break
			}
		}
		if py.startCmd == "" {
			py.startCmd = "python main.py"
		}
	}

	return py
}

// ── Go deep detection ─────────────────────────────────────────────────────────

// detectGoBuildTarget returns the best go build target path.
func detectGoBuildTarget(dir string) string {
	cmdDir := filepath.Join(dir, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				return "./cmd/" + e.Name()
			}
		}
	}
	return "./..."
}

// ── Ruby deep detection ───────────────────────────────────────────────────────

func detectRubyStart(dir string) string {
	if fileAt(dir, "config/routes.rb") || fileAt(dir, "config/application.rb") {
		return "bundle exec rails server -b 0.0.0.0 -p $PORT"
	}
	for _, cand := range []string{"config.ru", "app.rb", "server.rb", "main.rb"} {
		if fileAt(dir, cand) {
			if cand == "config.ru" {
				return "bundle exec rackup --host 0.0.0.0 --port $PORT"
			}
			return "bundle exec ruby " + cand
		}
	}
	return "bundle exec ruby app.rb"
}

// ── DockerfileTemplate ────────────────────────────────────────────────────────

// DockerfileTemplate generates a production Dockerfile by deep-inspecting dir.
func DockerfileTemplate(framework string, dir string, port int) string {
	p := fmt.Sprintf("%d", port)
	switch framework {
	case "nodejs":
		n := detectNode(dir)
		var sb strings.Builder
		sb.WriteString("FROM " + n.base + "\n")
		sb.WriteString("WORKDIR /app\n")
		// Copy ALL source files first so any postinstall script (e.g. "prisma generate")
		// can find files like prisma/schema.prisma, configs, etc.
		sb.WriteString("COPY . .\n")
		if n.prismaGen != "" {
			// Prisma 6+ loads prisma.config.ts at generate time and requires DATABASE_URL
			// to be present even though no real DB connection is made during generate.
			// ARG is build-time only — it does NOT persist in the final image and will NOT
			// override the real DATABASE_URL provided at container runtime via env vars.
			sb.WriteString("ARG DATABASE_URL=postgresql://placeholder:placeholder@localhost:5432/placeholder\n")
		}
		sb.WriteString("RUN " + n.install + "\n")
		if n.prismaGen != "" {
			// Explicit generate after install (idempotent; handles cases where postinstall
			// doesn't call it or uses a non-standard schema path)
			sb.WriteString("RUN " + n.prismaGen + "\n")
		}
		if n.buildCmd != "" {
			sb.WriteString("RUN " + n.buildCmd + "\n")
		}
		if n.pruneCmd != "" {
			sb.WriteString("RUN " + n.pruneCmd + "\n")
		}
		sb.WriteString("ENV NODE_ENV=production\n")
		sb.WriteString("ENV PORT=" + p + "\n")
		sb.WriteString("EXPOSE " + p + "\n")
		sb.WriteString("CMD " + toCMD(n.startCmd) + "\n")
		return sb.String()

	case "python":
		py := detectPython(dir)
		return "FROM python:3.12-slim\n" +
			"WORKDIR /app\n" +
			"COPY requirements.txt* pyproject.toml* setup.py* setup.cfg* ./\n" +
			"RUN " + py.install + "\n" +
			"COPY . .\n" +
			"ENV PORT=" + p + "\n" +
			"EXPOSE " + p + "\n" +
			"CMD " + toCMD(py.startCmd) + "\n"

	case "golang":
		target := detectGoBuildTarget(dir)
		return "FROM golang:1.22-alpine AS builder\n" +
			"WORKDIR /app\n" +
			"COPY go.mod go.sum* ./\n" +
			"RUN go mod download\n" +
			"COPY . .\n" +
			"RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server " + target + "\n\n" +
			"FROM alpine:latest\n" +
			"RUN apk --no-cache add ca-certificates tzdata\n" +
			"WORKDIR /app\n" +
			"COPY --from=builder /app/server .\n" +
			"ENV PORT=" + p + "\n" +
			"EXPOSE " + p + "\n" +
			"CMD [\"./server\"]\n"

	case "ruby":
		startCmd := detectRubyStart(dir)
		return "FROM ruby:3.3-alpine\n" +
			"WORKDIR /app\n" +
			"RUN apk --no-cache add build-base nodejs yarn tzdata postgresql-dev\n" +
			"COPY Gemfile Gemfile.lock* ./\n" +
			"RUN bundle install --jobs 4 --retry 3 --without development test\n" +
			"COPY . .\n" +
			"ENV PORT=" + p + "\n" +
			"ENV RACK_ENV=production\n" +
			"ENV RAILS_ENV=production\n" +
			"EXPOSE " + p + "\n" +
			"CMD " + toCMD(startCmd) + "\n"

	case "php":
		return "FROM php:8.3-apache\n" +
			"WORKDIR /var/www/html\n" +
			"RUN apt-get update && apt-get install -y unzip curl libzip-dev " +
			"&& docker-php-ext-install zip pdo pdo_mysql && rm -rf /var/lib/apt/lists/*\n" +
			"COPY composer.json composer.lock* ./\n" +
			"RUN curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer " +
			"&& composer install --no-dev --optimize-autoloader --no-interaction\n" +
			"COPY . .\n" +
			"RUN chown -R www-data:www-data /var/www/html\n" +
			"EXPOSE 80\n"

	case "java-maven":
		return "FROM maven:3.9-eclipse-temurin-21 AS builder\n" +
			"WORKDIR /app\n" +
			"COPY pom.xml .\n" +
			"RUN mvn dependency:go-offline -q\n" +
			"COPY src ./src\n" +
			"RUN mvn package -DskipTests -q\n\n" +
			"FROM eclipse-temurin:21-jre-alpine\n" +
			"WORKDIR /app\n" +
			"COPY --from=builder /app/target/*.jar app.jar\n" +
			"ENV PORT=" + p + "\n" +
			"EXPOSE " + p + "\n" +
			"CMD [\"java\", \"-jar\", \"app.jar\"]\n"

	case "java-gradle":
		return "FROM gradle:8-jdk21-alpine AS builder\n" +
			"WORKDIR /app\n" +
			"COPY build.gradle* settings.gradle* gradle* ./\n" +
			"COPY src ./src\n" +
			"RUN gradle build --no-daemon -q\n\n" +
			"FROM eclipse-temurin:21-jre-alpine\n" +
			"WORKDIR /app\n" +
			"COPY --from=builder /app/build/libs/*.jar app.jar\n" +
			"ENV PORT=" + p + "\n" +
			"EXPOSE " + p + "\n" +
			"CMD [\"java\", \"-jar\", \"app.jar\"]\n"

	default:
		return ""
	}
}
