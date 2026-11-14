# ==============================================================================
# ReconPilot: Production Golang 1.25 Container (Root Dockerfile)
# Optimized for Cloud Deployments (Render, Railway, Fly.io, Docker Hub)
# ==============================================================================

# Stage 1: Build binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git ca-certificates tzdata

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/reconpilot-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/reconpilot ./cmd/cli

# Stage 2: Minimal runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/reconpilot-api /usr/local/bin/reconpilot-api
COPY --from=builder /app/bin/reconpilot /usr/local/bin/reconpilot
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["reconpilot-api"]
