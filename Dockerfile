# ═══════════════════════════════════════════════════════════════════════════
#  Loyalty Nexus — Production Multi-Stage Dockerfile
#  Build context: repo root
#
#  Stages
#  ──────
#  builder   Compiles all Go binaries: api, worker, migrate
#  api       Alpine runtime — serves HTTP on :10000 (Render's required port)
#  worker    Alpine runtime — background job processor
#
#  Migration strategy
#  ──────────────────
#  The /migrate binary is embedded in the api image.
#  Render's preDeployCommand runs "/migrate up" before each new version
#  goes live, using MIGRATE_DATABASE_URL (external hostname + SSL).
#  golang-migrate tracks applied versions in schema_migrations — fully
#  idempotent, safe to re-run on every deploy.
#  If any migration fails, the container exits non-zero and Render keeps
#  the previous healthy version live (zero-downtime guarantee).
# ═══════════════════════════════════════════════════════════════════════════

# ─── Stage 1: Builder ───────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy module manifests first — Docker layer cache only invalidated when
# go.mod / go.sum change, not on every source file change.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ .

ARG VERSION=dev

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s -X main.version=${VERSION}" \
      -o /bin/api \
      ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s" \
      -o /bin/worker \
      ./cmd/worker

# Build Migrate binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s" \
      -o /bin/migrate \
      ./cmd/migrate

# ─── Stage 2: API Runtime ───────────────────────────────────────────────────
FROM alpine:3.19 AS api

RUN apk add --no-cache ca-certificates tzdata

# API binary — binds to PORT env var (Render injects PORT=10000)
COPY --from=builder /bin/api     /api

# Migrate binary — called by Render's preDeployCommand before each deploy
COPY --from=builder /bin/migrate /migrate

# SQL migration files — needed by both entrypoint and preDeployCommand
COPY database/migrations/        /app/migrations/

# Lightweight entrypoint: just starts /api immediately.
# Migrations are handled separately by preDeployCommand — never block startup.
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV MIGRATIONS_DIR=/app/migrations

# Render requires port 10000 for web services
EXPOSE 10000

ENTRYPOINT ["/entrypoint.sh"]

# ─── Stage 3: Worker Runtime ────────────────────────────────────────────────
FROM alpine:3.19 AS worker

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/worker /worker

ENTRYPOINT ["/worker"]
