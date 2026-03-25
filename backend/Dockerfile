# Loyalty Nexus Backend — Multi-stage Dockerfile
# Stage: builder → api → worker
#
# RENDER DEPLOYMENT NOTE:
#   Build context is the `backend/` directory (rootDir in render.yaml).
#   database/migrations/ lives at the repo root and is NOT in this build context.
#   Run migrations separately via Render Shell or a pre-deploy job (see DEPLOYMENT.md §2).

# ─── Stage 1: Builder ───────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
# NOTE: `git describe` is removed — it requires .git which is not in the build context
# on Render (rootDir: backend). Pass VERSION as a build-arg or embed a static value.
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=${VERSION}" \
    -o /bin/api ./cmd/api

# Build worker binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" \
    -o /bin/worker ./cmd/worker

# ─── Stage 2: API Runtime (distroless) ──────────────────────────
FROM gcr.io/distroless/static-debian12 AS api

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/api /api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

# ─── Stage 3: Worker Runtime (distroless) ───────────────────────
FROM gcr.io/distroless/static-debian12 AS worker

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/worker /worker

USER nonroot:nonroot
ENTRYPOINT ["/worker"]
