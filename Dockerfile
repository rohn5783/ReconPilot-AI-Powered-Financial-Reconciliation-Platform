# ==============================================================================
# ReconPilot Multi-Stage Production Container Architecture
# ==============================================================================

# Stage 1: Build Golang Backend & CLI Engine
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app/backend

RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/reconpilot-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/reconpilot-cli ./cmd/cli

# Stage 2: Build Industrial React 19 Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package.json ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# Stage 3: Hardened Backend Production Runner
FROM alpine:3.20 AS backend
RUN apk add --no-cache \
    ca-certificates \
    nmap \
    bind-tools \
    curl \
    tzdata

# Security: Dedicated non-root user with minimal privilege
RUN addgroup -g 10001 -S reconpilot && \
    adduser -u 10001 -S reconpilot -G reconpilot -s /sbin/nologin

WORKDIR /app

COPY --from=backend-builder /bin/reconpilot-api /usr/local/bin/reconpilot-api
COPY --from=backend-builder /bin/reconpilot-cli /usr/local/bin/reconpilot-cli
COPY backend/migrations /app/migrations

RUN chown -R reconpilot:reconpilot /app

USER reconpilot:reconpilot

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/reconpilot-api"]

# Stage 4: High-Performance Frontend Nginx Reverse Proxy
FROM nginx:alpine AS frontend
COPY docker/nginx.conf /etc/nginx/nginx.conf
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

