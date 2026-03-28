# ═══════════════════════════════════════════════════════════════════════════
#  Loyalty Nexus — Production Multi-Stage Dockerfile
#  Build context: repo root (Render default, no rootDir set)
#
#  Stages
#  ──────
#  builder   Compiles all Go binaries: api, worker, migrate
#  api       Distroless runtime — serves HTTP on :8080
#             Contains /migrate binary + /app/migrations/ so Render's
#             preDeployCommand can run `/migrate up` before the API starts.
#  worker    Distroless runtime — background job processor
#
#  Migration strategy
#  ──────────────────
#  golang-migrate/v4 tracks applied versions in schema_migrations.
#  Running `/migrate up` on every deploy is fully idempotent — it is a
#  no-op when the schema is already current. If any migration fails the
#  preDeployCommand exits non-zero and Render aborts the deploy, keeping
#  the previous healthy version live (zero-downtime guarantee).
# ═══════════════════════════════════════════════════════════════════════════

# ─── Stage 1: Builder ───────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

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
# Uses distroless/static — no shell, no package manager, minimal attack surface.
# The /migrate binary and /app/migrations/ are included so Render's
# preDeployCommand ("/migrate up") can run inside this same image.
FROM gcr.io/distroless/static-debian12 AS api

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo

# API server binary
COPY --from=builder /bin/api     /api

# Migration runner binary + SQL files (embedded for preDeployCommand)
COPY --from=builder /bin/migrate /migrate
COPY database/migrations/        /app/migrations/

ENV MIGRATIONS_DIR=/app/migrations

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

# ─── Stage 3: Worker Runtime ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12 AS worker

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /bin/worker                         /worker

USER nonroot:nonroot
ENTRYPOINT ["/worker"]
