# ═══════════════════════════════════════════════════════════════════════════
#  Loyalty Nexus — Production Multi-Stage Dockerfile
#  Build context: repo root
#
#  Stages
#  ──────
#  builder   Compiles all Go binaries: api, worker, migrate
#  api       Alpine runtime — runs migrations then serves HTTP on :8080
#  worker    Alpine runtime — background job processor
#
#  Migration strategy
#  ──────────────────
#  golang-migrate/v4 tracks applied versions in schema_migrations.
#  entrypoint.sh runs /migrate up before starting /api on every deploy.
#  This is fully idempotent — already-applied migrations are skipped.
#  If any migration fails, the container exits non-zero and Render keeps
#  the previous healthy version live (zero-downtime guarantee).
# ═══════════════════════════════════════════════════════════════════════════

# ─── Stage 1: Builder ───────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy module manifests first — Docker layer cache is only invalidated
# when go.mod or go.sum change, not on every source file change.
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
# Uses alpine:3.19 so the entrypoint shell script can execute.
FROM alpine:3.19 AS api

RUN apk add --no-cache ca-certificates tzdata

# API and migration binaries
COPY --from=builder /bin/api     /api
COPY --from=builder /bin/migrate /migrate

# SQL migration files
COPY database/migrations/        /app/migrations/

# Entrypoint script: run migrations first, then start the API
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV MIGRATIONS_DIR=/app/migrations

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]

# ─── Stage 3: Worker Runtime ────────────────────────────────────────────────
FROM alpine:3.19 AS worker

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/worker /worker

ENTRYPOINT ["/worker"]
