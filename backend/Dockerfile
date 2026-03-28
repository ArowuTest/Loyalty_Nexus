# ═══════════════════════════════════════════════════════════════════════════
#  Loyalty Nexus Backend — Multi-Stage Dockerfile
#  Build context: backend/ directory (for local development)
#
#  NOTE: For Render deployments use the repo-root Dockerfile which has
#  access to database/migrations/. This file is for local dev and CI only.
# ═══════════════════════════════════════════════════════════════════════════

# ─── Stage 1: Builder ───────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

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
FROM gcr.io/distroless/static-debian12 AS api

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /bin/api                            /api
COPY --from=builder /bin/migrate                        /migrate

# NOTE: When building from backend/ context, migrations must be mounted
# at /app/migrations at runtime or MIGRATIONS_DIR must point to them.
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
