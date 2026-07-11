# =============================================================================
# AVEX Backend — Multi-stage Dockerfile
# =============================================================================
# Builds both the API server and the outbox worker as separate binaries,
# then packages them into a single lightweight image. The ENTRYPOINT
# selects which binary to run based on the APP_ROLE env var:
#   APP_ROLE=server  → run the HTTP API server
#   APP_ROLE=worker  → run the outbox publisher worker
#
# Build:
#   docker build -t avex-backend:latest .
#
# Run server:
#   docker run -e APP_ROLE=server -e DATABASE_URL=... -e REDIS_URL=... avex-backend:latest
#
# Run worker:
#   docker run -e APP_ROLE=worker -e DATABASE_URL=... -e REDIS_URL=... avex-backend:latest
# =============================================================================

# ===== Stage 1: Build =====
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build arguments for version info (injected by CI/CD)
ARG VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${GIT_COMMIT}" \
    -o /out/server \
    ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w" \
    -o /out/worker \
    ./cmd/worker

# ===== Stage 2: Runtime =====
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S avex && adduser -S avex -G avex

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /out/server /app/server
COPY --from=builder /out/worker /app/worker

# Copy .env.example for reference
COPY .env.example /app/.env.example

# Switch to non-root user
USER avex

EXPOSE 8080

# Health check (only meaningful for the server role)
# The server exposes /api/healthz and /api/health endpoints.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/healthz || exit 1

# The APP_ROLE env var determines which binary to run.
# Default to "server" if not set.
ENV APP_ROLE=server

# Use a shell entrypoint to select the binary based on APP_ROLE.
ENTRYPOINT ["sh", "-c", "if [ \"$APP_ROLE\" = \"worker\" ]; then exec /app/worker; else exec /app/server; fi"]
