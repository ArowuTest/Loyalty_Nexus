# Loyalty Nexus Backend — Multi-stage Dockerfile
# Stage: builder → api → worker

# ─── Stage 1: Builder ───────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
    -o /bin/api ./cmd/api

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
