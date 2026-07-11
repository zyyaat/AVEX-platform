# =============================================================================
# AVEX Frontend App — Multi-stage Dockerfile
# =============================================================================
# Builds any of the 5 frontend apps (merchant, admin, customer, merchant, support)
# as a static SPA served by nginx.
#
# Build (from repo root):
#   docker build -f artifacts/merchant/Dockerfile -t avex-merchant:latest .
#   docker build -f artifacts/admin/Dockerfile -t avex-admin:latest .
#
# The build context MUST be the repo root because pnpm workspace requires the
# root pnpm-lock.yaml and pnpm-workspace.yaml.
#
# Runtime:
#   docker run -p 8080:80 -e VITE_API_BASE=/api/v1 avex-merchant:latest
# =============================================================================

# ===== Stage 1: Build the SPA =====
FROM node:22-alpine AS builder

RUN apk add --no-cache git
RUN corepack enable && corepack prepare pnpm@11.11.0 --activate

WORKDIR /build

# Copy workspace manifests first (better layer caching)
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json tsconfig.base.json ./
COPY artifacts/ ./artifacts/
COPY scripts/ ./scripts/

# Install deps with frozen lockfile
RUN pnpm install --frozen-lockfile

# Build the specific app (set via APP_NAME build arg)
ARG APP_NAME=merchant
WORKDIR /build/artifacts/${APP_NAME}

RUN pnpm build

# ===== Stage 2: Runtime (nginx) =====
FROM nginx:1.27-alpine

ARG APP_NAME=merchant
LABEL org.opencontainers.image.title="avex-${APP_NAME}" \
      org.opencontainers.image.description="AVEX ${APP_NAME} frontend app" \
      org.opencontainers.image.source="https://github.com/zyyaat/AVEX-platform"

# Replace default nginx config with SPA-aware one
COPY artifacts/${APP_NAME}/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built static assets from builder
COPY --from=builder /build/artifacts/${APP_NAME}/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost/ || exit 1
