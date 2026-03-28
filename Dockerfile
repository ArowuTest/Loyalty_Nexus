# Root-level Dockerfile for Render deployment
# Render expects ./Dockerfile at repo root; this builds the backend API
# with the full repo as build context so backend/ code is accessible.
#
# ─── Stage 1: Builder ───────────────────────────────────────────
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
# Copy only the backend module files
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=${VERSION}" \
    -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" \
    -o /bin/worker ./cmd/worker

# ─── Stage 2: API Runtime ───────────────────────────────────────
FROM gcr.io/distroless/static-debian12 AS api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

# ─── Stage 3: Worker Runtime ────────────────────────────────────
FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
