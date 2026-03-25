# backend/Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker/main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /api /api
COPY --from=builder /worker /worker

EXPOSE 8080
CMD ["/api"]
