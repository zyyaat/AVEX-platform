#!/usr/bin/env bash
# =============================================================================
# AVEX Platform — Unified startup script for Replit
# =============================================================================
# Starts the FULL stack natively on Replit (NO Docker):
#   1. PostgreSQL (via Replit postgresql-16 module)
#   2. Redis (via Nix package, started as daemon)
#   3. Backend API server (Go)
#   4. Backend worker (Go) — for outbox + notifications jobs
#
# Frontend apps (driver, customer, admin, merchant, support) are auto-started
# by Replit from their artifact.toml files — they appear as separate tabs in
# the bottom panel of the Replit workspace. Do NOT start them here.
#
# Usage:
#   ./run.sh              Start everything (interactive, logs to console)
#   ./run.sh --background Start everything in background (logs to files)
#   ./run.sh --stop       Stop all AVEX processes
#   ./run.sh --status     Show status of all components
#
# Required Replit Secrets (set via the lock icon in Replit sidebar):
#   - JWT_SECRET          — random string ≥32 chars
#   - MAPBOX_ACCESS_TOKEN — public token from https://account.mapbox.com/
#
# If running locally (not Replit), set these in your shell or .env file.
# =============================================================================

set -euo pipefail

# -----------------------------------------------------------------------------
# Colors for pretty output
# -----------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log()  { echo -e "${GREEN}[$(date +%H:%M:%S)]${NC} $*"; }
warn() { echo -e "${YELLOW}[$(date +%H:%M:%S)] ⚠️  ${NC} $*"; }
err()  { echo -e "${RED}[$(date +%H:%M:%S)] ❌ ${NC} $*" >&2; }
info() { echo -e "${BLUE}[$(date +%H:%M:%S)] ℹ️  ${NC} $*"; }

# -----------------------------------------------------------------------------
# Resolve workspace directory (Replit: /home/runner/<repl-name>, local: script dir)
# -----------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BACKGROUND=false
ACTION="start"
case "${1:-}" in
  --background) BACKGROUND=true ;;
  --stop)       ACTION="stop" ;;
  --status)     ACTION="status" ;;
  --help|-h)
    head -30 "$0"
    exit 0
    ;;
esac

# -----------------------------------------------------------------------------
# STOP mode
# -----------------------------------------------------------------------------
if [ "$ACTION" = "stop" ]; then
  log "Stopping all AVEX processes..."
  pkill -f "cmd/server"  2>/dev/null && log "Stopped backend server"   || info "Backend server not running"
  pkill -f "cmd/worker"  2>/dev/null && log "Stopped backend worker"   || info "Backend worker not running"
  pkill -f "redis-server" 2>/dev/null && log "Stopped Redis"           || info "Redis not running"
  # Don't kill PostgreSQL on Replit — it's managed by the module
  log "Done. (PostgreSQL is managed by Replit module — not stopped)"
  exit 0
fi

# -----------------------------------------------------------------------------
# STATUS mode
# -----------------------------------------------------------------------------
if [ "$ACTION" = "status" ]; then
  echo ""
  echo "AVEX Platform Status"
  echo "===================="
  if pg_isready -h 127.0.0.1 -p 5432 >/dev/null 2>&1; then
    echo -e "  PostgreSQL:  ${GREEN}✅ running${NC}"
  else
    echo -e "  PostgreSQL:  ${RED}❌ not running${NC}"
  fi
  if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
    echo -e "  Redis:       ${GREEN}✅ running${NC}"
  else
    echo -e "  Redis:       ${RED}❌ not running${NC}"
  fi
  if curl -sf http://127.0.0.1:8080/api/healthz >/dev/null 2>&1; then
    echo -e "  Backend API: ${GREEN}✅ running${NC}"
  else
    echo -e "  Backend API: ${RED}❌ not running${NC}"
  fi
  if pgrep -f "cmd/worker" >/dev/null 2>&1; then
    echo -e "  Worker:      ${GREEN}✅ running${NC}"
  else
    echo -e "  Worker:      ${RED}❌ not running${NC}"
  fi
  echo ""
  exit 0
fi

# =============================================================================
# START mode
# =============================================================================

log "🚀 Starting AVEX Platform on Replit..."

# -----------------------------------------------------------------------------
# 1. Validate required environment variables (secrets)
# -----------------------------------------------------------------------------
validate_env() {
  local missing=()
  if [ -z "${JWT_SECRET:-}" ]; then
    missing+=("JWT_SECRET")
  fi
  if [ -z "${MAPBOX_ACCESS_TOKEN:-}" ]; then
    missing+=("MAPBOX_ACCESS_TOKEN")
  fi
  if [ ${#missing[@]} -gt 0 ]; then
    err "Missing required environment variables: ${missing[*]}"
    err ""
    err "On Replit: set them as Secrets (lock icon in the left sidebar)."
    err "Locally:   export JWT_SECRET='...' && export MAPBOX_ACCESS_TOKEN='...'"
    err ""
    err "To generate a JWT_SECRET:  openssl rand -hex 32"
    err "To get a MAPBOX_ACCESS_TOKEN:  https://account.mapbox.com/access-tokens/"
    exit 1
  fi
}

validate_env

# -----------------------------------------------------------------------------
# 2. Ensure PostgreSQL is running
# -----------------------------------------------------------------------------
ensure_postgres() {
  log "📦 Checking PostgreSQL..."
  # Replit's postgresql-16 module auto-starts on first query.
  # We just need to wait for it.
  for i in $(seq 1 30); do
    if pg_isready -h 127.0.0.1 -p 5432 >/dev/null 2>&1; then
      log "✅ PostgreSQL is ready"
      return 0
    fi
    # Try to wake it up with a connection attempt
    if [ $i -eq 1 ]; then
      info "PostgreSQL not yet responding, waiting for Replit module to start it..."
    fi
    sleep 1
  done
  err "PostgreSQL did not become ready within 30 seconds."
  err "On Replit, the postgresql-16 module should start automatically."
  err "Try opening a database shell in the Replit UI to trigger it, then re-run this script."
  exit 1
}

ensure_postgres

# -----------------------------------------------------------------------------
# 3. Ensure the avex_dev database + avex user exist
# -----------------------------------------------------------------------------
ensure_database() {
  log "📦 Ensuring 'avex_dev' database exists..."
  # On Replit, the default user is the system user; postgres is often trusted.
  # Try common connection strings.
  local psql_opts="-h 127.0.0.1 -p 5432 -U postgres -d postgres -tAc"

  # Try with no password first (Replit local trust auth)
  if psql $psql_opts "SELECT 1" >/dev/null 2>&1; then
    :
  # Try as current user
  elif psql -h 127.0.0.1 -p 5432 -U "$(whoami)" -d postgres -tAc "SELECT 1" >/dev/null 2>&1; then
    psql_opts="-h 127.0.0.1 -p 5432 -U $(whoami) -d postgres -tAc"
  else
    warn "Cannot connect to PostgreSQL as postgres or $(whoami)."
    warn "Will try to proceed — backend may fail if DATABASE_URL is wrong."
    return 0
  fi

  # Create avex role (ignore error if exists)
  psql $psql_opts "CREATE ROLE avex WITH LOGIN PASSWORD 'avex' CREATEDB SUPERUSER;" 2>/dev/null || true
  # Create avex_dev database (ignore error if exists)
  psql $psql_opts "CREATE DATABASE avex_dev OWNER avex;" 2>/dev/null || true
  log "✅ Database 'avex_dev' and role 'avex' are ready"
}

ensure_database

# Export DATABASE_URL if not already set (Replit module sets it automatically
# on Replit, but for local dev we set it here).
export DATABASE_URL="${DATABASE_URL:-postgres://avex:avex@127.0.0.1:5432/avex_dev?sslmode=disable}"

# -----------------------------------------------------------------------------
# 4. Ensure Redis is running (started as a daemon, not via Docker)
# -----------------------------------------------------------------------------
ensure_redis() {
  log "📦 Checking Redis..."
  if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
    log "✅ Redis is already running"
    return 0
  fi

  info "Starting Redis daemon..."
  # Start Redis as a background daemon, logs to file
  mkdir -p /tmp/avex-redis
  redis-server \
    --daemonize yes \
    --port 6379 \
    --bind 127.0.0.1 \
    --dir /tmp/avex-redis \
    --logfile /tmp/avex-redis/redis.log \
    --save "" \
    --appendonly no

  # Wait for it
  for i in $(seq 1 10); do
    if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
      log "✅ Redis is ready"
      return 0
    fi
    sleep 0.5
  done
  err "Redis did not start. Check /tmp/avex-redis/redis.log"
  exit 1
}

ensure_redis
export REDIS_URL="${REDIS_URL:-redis://127.0.0.1:6379/0}"

# -----------------------------------------------------------------------------
# 5. Kill any stale backend processes
# -----------------------------------------------------------------------------
log "🧹 Cleaning up stale processes..."
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
sleep 1

# -----------------------------------------------------------------------------
# 6. Build the backend (fast — Go caches compiled deps)
# -----------------------------------------------------------------------------
log "🔧 Compiling backend..."
cd "$SCRIPT_DIR/backend"
if ! go build -o /tmp/avex-server ./cmd/server; then
  err "Failed to build backend server"
  exit 1
fi
if ! go build -o /tmp/avex-worker ./cmd/worker; then
  err "Failed to build backend worker"
  exit 1
fi
log "✅ Backend compiled"

# -----------------------------------------------------------------------------
# 7. Start the backend API server
# -----------------------------------------------------------------------------
log "🚀 Starting backend API server on port ${APP_PORT:-8080}..."
if [ "$BACKGROUND" = "true" ]; then
  nohup /tmp/avex-server > "$SCRIPT_DIR/backend/server.log" 2>&1 &
  echo $! > "$SCRIPT_DIR/backend/server.pid"
  disown
else
  /tmp/avex-server > "$SCRIPT_DIR/backend/server.log" 2>&1 &
  echo $! > "$SCRIPT_DIR/backend/server.pid"
fi

# -----------------------------------------------------------------------------
# 8. Start the backend worker (outbox publisher + job processor)
# -----------------------------------------------------------------------------
log "⚙️  Starting backend worker..."
if [ "$BACKGROUND" = "true" ]; then
  nohup /tmp/avex-worker > "$SCRIPT_DIR/backend/worker.log" 2>&1 &
  echo $! > "$SCRIPT_DIR/backend/worker.pid"
  disown
else
  /tmp/avex-worker > "$SCRIPT_DIR/backend/worker.log" 2>&1 &
  echo $! > "$SCRIPT_DIR/backend/worker.pid"
fi

cd "$SCRIPT_DIR"

# -----------------------------------------------------------------------------
# 9. Wait for the API server to become healthy
# -----------------------------------------------------------------------------
log "⏳ Waiting for API server to be healthy..."
for i in $(seq 1 30); do
  if curl -sf http://127.0.0.1:8080/api/healthz >/dev/null 2>&1; then
    log "✅ Backend API is healthy"
    break
  fi
  if [ $i -eq 30 ]; then
    err "Backend API did not become healthy within 30 seconds."
    err "Check $SCRIPT_DIR/backend/server.log for details."
    tail -20 "$SCRIPT_DIR/backend/server.log" 2>/dev/null || true
    exit 1
  fi
  sleep 1
done

# -----------------------------------------------------------------------------
# 10. Print summary
# -----------------------------------------------------------------------------
echo ""
echo "============================================================"
echo -e "${GREEN}  AVEX Platform is up and running${NC}"
echo "============================================================"
echo ""
echo "  Backend API:    http://127.0.0.1:8080"
echo "  Health check:   http://127.0.0.1:8080/api/healthz"
echo "  API base:       http://127.0.0.1:8080/api/v1"
echo ""
echo "  Frontend apps (auto-started by Replit from artifact.toml):"
echo "    Driver:   /driver/"
echo "    Customer: /"
echo "    Admin:    /admin/"
echo "    Merchant: /merchant/"
echo "    Support:  /support/"
echo ""
echo "  Logs:"
echo "    Backend server: $SCRIPT_DIR/backend/server.log"
echo "    Backend worker: $SCRIPT_DIR/backend/worker.log"
echo "    Redis:          /tmp/avex-redis/redis.log"
echo ""
echo "  Commands:"
echo "    ./run.sh --status   Check status of all components"
echo "    ./run.sh --stop     Stop all AVEX processes"
echo "    ./seed-driver.sh    Create a test driver account"
echo "============================================================"

# In foreground mode, keep the script alive so the Replit workflow doesn't end
if [ "$BACKGROUND" = "false" ]; then
  info "Press Ctrl+C to stop. Backend is running in background."
  # Tail the server log so the workflow shows output
  tail -f "$SCRIPT_DIR/backend/server.log" 2>/dev/null || wait
fi
