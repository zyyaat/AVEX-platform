#!/usr/bin/env bash
# =============================================================================
# AVEX Platform — Unified startup script for Replit
# =============================================================================
# Starts the FULL stack natively on Replit (NO Docker):
#   1. PostgreSQL (via Replit postgresql-16 module)
#   2. Redis (via Nix package, started as daemon)
#   3. Backend API server (Go) — runs in FOREGROUND to keep workflow alive
#   4. Backend worker (Go) — runs in background
#
# IMPORTANT for Replit: The server MUST run in the foreground (not background).
# If this script exits, Replit kills ALL child processes, including the server.
# The script stays alive by blocking on the server process (`wait $SERVER_PID`).
#
# Frontend apps (driver, customer, admin, merchant, support) are auto-started
# by Replit from their artifact.toml files — they appear as separate tabs in
# the bottom panel of the Replit workspace. Do NOT start them here.
#
# Usage:
#   ./run.sh              Start everything (server in foreground)
#   ./run.sh --background Start everything in background (logs to files)
#   ./run.sh --stop       Stop all AVEX processes
#   ./run.sh --status     Show status of all components
#   ./run.sh --debug      Start with verbose debug output
#
# Required Replit Secrets (set via the lock icon in Replit sidebar):
#   - JWT_SECRET          — random string ≥32 chars
#   - MAPBOX_ACCESS_TOKEN — public token from https://account.mapbox.com/
# =============================================================================

# Don't use `set -e` during startup — we want to handle errors gracefully.
# We'll explicitly check critical commands instead.
set -uo pipefail

# -----------------------------------------------------------------------------
# Colors for pretty output
# -----------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log()  { echo -e "${GREEN}[$(date +%H:%M:%S)]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[$(date +%H:%M:%S)] ⚠️  ${NC} $*" >&2; }
err()  { echo -e "${RED}[$(date +%H:%M:%S)] ❌ ${NC} $*" >&2; }
info() { echo -e "${BLUE}[$(date +%H:%M:%S)] ℹ️  ${NC} $*" >&2; }
debug() { [ "${DEBUG:-0}" = "1" ] && echo -e "${BLUE}[$(date +%H:%M:%S)] 🔍 ${NC} $*" >&2 || true; }

# -----------------------------------------------------------------------------
# Resolve workspace directory
# -----------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# -----------------------------------------------------------------------------
# Parse arguments
# -----------------------------------------------------------------------------
BACKGROUND=false
DEBUG=0
ACTION="start"
case "${1:-}" in
  --background) BACKGROUND=true ;;
  --debug)      DEBUG=1; BACKGROUND=false ;;
  --stop)       ACTION="stop" ;;
  --status)     ACTION="status" ;;
  --help|-h)
    head -30 "$0"
    exit 0
    ;;
esac
export DEBUG

# -----------------------------------------------------------------------------
# Cleanup function — called on exit
# -----------------------------------------------------------------------------
cleanup() {
  local exit_code=$?
  if [ "$ACTION" = "start" ] && [ "$BACKGROUND" = "false" ]; then
    log "Shutting down AVEX backend..."
    if [ -n "${SERVER_PID:-}" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
      kill "$SERVER_PID" 2>/dev/null
      wait "$SERVER_PID" 2>/dev/null
      log "Backend server stopped"
    fi
    if [ -n "${WORKER_PID:-}" ] && kill -0 "$WORKER_PID" 2>/dev/null; then
      kill "$WORKER_PID" 2>/dev/null
      wait "$WORKER_PID" 2>/dev/null
      log "Backend worker stopped"
    fi
    # Don't kill Redis — it may be used by other things, and it's lightweight
    # Don't kill PostgreSQL — managed by Replit module
  fi
  exit $exit_code
}
trap cleanup EXIT INT TERM

# =============================================================================
# STOP mode
# =============================================================================
if [ "$ACTION" = "stop" ]; then
  log "Stopping all AVEX processes..."
  pkill -f "avex-server"   2>/dev/null && log "Stopped backend server"   || info "Backend server not running"
  pkill -f "avex-worker"   2>/dev/null && log "Stopped backend worker"   || info "Backend worker not running"
  pkill -f "redis-server"  2>/dev/null && log "Stopped Redis"           || info "Redis not running"
  # Don't kill PostgreSQL — managed by Replit module
  log "Done. (PostgreSQL is managed by Replit module — not stopped)"
  exit 0
fi

# =============================================================================
# STATUS mode
# =============================================================================
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
  if pgrep -f "avex-worker" >/dev/null 2>&1; then
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
debug "SCRIPT_DIR=$SCRIPT_DIR"
debug "BACKGROUND=$BACKGROUND"
debug "DEBUG=$DEBUG"

# -----------------------------------------------------------------------------
# 1. Validate required environment variables (secrets)
# -----------------------------------------------------------------------------
log "🔑 Checking required secrets..."
missing=()
if [ -z "${JWT_SECRET:-}" ]; then
  missing+=("JWT_SECRET")
fi
if [ -z "${MAPBOX_ACCESS_TOKEN:-}" ]; then
  missing+=("MAPBOX_ACCESS_TOKEN")
fi
if [ ${#missing[@]} -gt 0 ]; then
  err "Missing required secrets: ${missing[*]}"
  err ""
  err "On Replit: click the 🔒 lock icon in the left sidebar and add:"
  err "  JWT_SECRET          → run 'openssl rand -hex 32' in the shell to generate"
  err "  MAPBOX_ACCESS_TOKEN → get from https://account.mapbox.com/access-tokens/"
  err ""
  err "After adding secrets, click Run again."
  exit 1
fi
log "✅ All required secrets are set"
debug "JWT_SECRET length: ${#JWT_SECRET}"
debug "MAPBOX_ACCESS_TOKEN starts with: ${MAPBOX_ACCESS_TOKEN:0:3}..."

# -----------------------------------------------------------------------------
# 2. Ensure PostgreSQL is running (start it manually if needed)
# -----------------------------------------------------------------------------
log "📦 Checking PostgreSQL..."

# Data directory for our manually-init'd PostgreSQL cluster.
# On Replit, the postgresql-16 module doesn't always auto-start, so we
# manage our own cluster in $HOME/.local/share/postgresql-16/data.
PG_DATA_DIR="${PG_DATA_DIR:-$HOME/.local/share/postgresql-16/data}"
PG_LOG_FILE="${PG_LOG_FILE:-/tmp/postgres.log}"
PG_RUN_DIR="/run/postgresql"

# Helper: check if PostgreSQL is responding
pg_is_running() {
  pg_isready -h 127.0.0.1 -p 5432 >/dev/null 2>&1
}

# Helper: start PostgreSQL (init the cluster first if needed)
start_postgres() {
  # Ensure /run/postgresql exists (PostgreSQL needs it for the socket lock file)
  if [ ! -d "$PG_RUN_DIR" ]; then
    debug "Creating $PG_RUN_DIR for PostgreSQL socket"
    mkdir -p "$PG_RUN_DIR"
    chmod 775 "$PG_RUN_DIR"
  fi

  # Init the cluster if the data directory doesn't exist yet
  if [ ! -f "$PG_DATA_DIR/PG_VERSION" ]; then
    log "📦 Initializing PostgreSQL data directory ($PG_DATA_DIR)..."
    mkdir -p "$PG_DATA_DIR"
    if ! initdb -D "$PG_DATA_DIR" --auth=trust >/tmp/initdb.log 2>&1; then
      err "initdb failed. Log:"
      tail -20 /tmp/initdb.log >&2
      exit 1
    fi
    log "✅ PostgreSQL cluster initialized"
  fi

  # Start the server (if not already running)
  log "🔧 Starting PostgreSQL server..."
  if ! pg_ctl -D "$PG_DATA_DIR" -l "$PG_LOG_FILE" start >/dev/null 2>&1; then
    # Maybe it's already running — check
    if pg_is_running; then
      log "✅ PostgreSQL was already running"
      return 0
    fi
    err "Failed to start PostgreSQL. Log:"
    tail -20 "$PG_LOG_FILE" >&2
    exit 1
  fi
}

# Try to detect if PostgreSQL is already running (from a previous run.sh)
if pg_is_running; then
  log "✅ PostgreSQL is already running"
else
  start_postgres
  # Wait for it to be ready (up to 30 seconds)
  for i in $(seq 1 30); do
    if pg_is_running; then
      log "✅ PostgreSQL is ready"
      break
    fi
    if [ $i -eq 30 ]; then
      err "PostgreSQL did not become ready within 30 seconds."
      err "Check $PG_LOG_FILE for details:"
      tail -20 "$PG_LOG_FILE" >&2
      exit 1
    fi
    sleep 1
  done
fi

# -----------------------------------------------------------------------------
# 3. Ensure the avex_dev database + avex user exist
# -----------------------------------------------------------------------------
log "📦 Ensuring 'avex_dev' database exists..."

# Find a working PostgreSQL connection
PSQL_BASE=""
for conn_user in postgres "$(whoami)" avex; do
  debug "Trying PostgreSQL as user: $conn_user"
  if psql -h 127.0.0.1 -p 5432 -U "$conn_user" -d postgres -tAc "SELECT 1" >/dev/null 2>&1; then
    PSQL_BASE="-h 127.0.0.1 -p 5432 -U $conn_user -d postgres"
    debug "Connected as: $conn_user"
    break
  fi
done

if [ -z "$PSQL_BASE" ]; then
  warn "Cannot connect to PostgreSQL to create database."
  warn "The backend may fail if DATABASE_URL points to a non-existent database."
  warn "Continuing anyway..."
else
  # Create avex role (ignore error if exists)
  psql $PSQL_BASE -tAc "CREATE ROLE avex WITH LOGIN PASSWORD 'avex' CREATEDB SUPERUSER;" 2>/dev/null || true
  # Create avex_dev database (ignore error if exists)
  psql $PSQL_BASE -tAc "CREATE DATABASE avex_dev OWNER avex;" 2>/dev/null || true
  log "✅ Database 'avex_dev' and role 'avex' are ready"
fi

export DATABASE_URL="${DATABASE_URL:-postgres://avex:avex@127.0.0.1:5432/avex_dev?sslmode=disable}"
debug "DATABASE_URL=$DATABASE_URL"

# -----------------------------------------------------------------------------
# 4. Ensure Redis is running
# -----------------------------------------------------------------------------
log "📦 Checking Redis..."
if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
  log "✅ Redis is already running"
else
  info "Starting Redis daemon..."
  mkdir -p /tmp/avex-redis
  redis-server \
    --daemonize yes \
    --port 6379 \
    --bind 127.0.0.1 \
    --dir /tmp/avex-redis \
    --logfile /tmp/avex-redis/redis.log \
    --save "" \
    --appendonly no 2>/dev/null

  for i in $(seq 1 10); do
    if redis-cli -h 127.0.0.1 -p 6379 ping >/dev/null 2>&1; then
      log "✅ Redis is ready"
      break
    fi
    if [ $i -eq 10 ]; then
      err "Redis did not start. Check /tmp/avex-redis/redis.log"
      cat /tmp/avex-redis/redis.log 2>/dev/null | tail -10
      exit 1
    fi
    sleep 0.5
  done
fi
export REDIS_URL="${REDIS_URL:-redis://127.0.0.1:6379/0}"

# -----------------------------------------------------------------------------
# 5. Kill any stale processes (backend + frontend Vite dev servers)
# -----------------------------------------------------------------------------
log "🧹 Cleaning up stale processes..."
pkill -f "avex-server" 2>/dev/null || true
pkill -f "avex-worker" 2>/dev/null || true
# Kill stale Vite dev servers that may be holding artifact ports
pkill -f "vite.*--config" 2>/dev/null || true
sleep 1

# -----------------------------------------------------------------------------
# 6. Build the backend
# -----------------------------------------------------------------------------
log "🔧 Compiling backend..."
cd "$SCRIPT_DIR/backend"

debug "Running: go build -o /tmp/avex-server ./cmd/server"
if ! go build -o /tmp/avex-server ./cmd/server 2>&1; then
  err "Failed to build backend server"
  err "Build errors above. Check your Go code."
  exit 1
fi

debug "Running: go build -o /tmp/avex-worker ./cmd/worker"
if ! go build -o /tmp/avex-worker ./cmd/worker 2>&1; then
  err "Failed to build backend worker"
  err "Build errors above. Check your Go code."
  exit 1
fi
log "✅ Backend compiled"

# -----------------------------------------------------------------------------
# 7. Start the backend worker (BACKGROUND — it's secondary)
# -----------------------------------------------------------------------------
log "⚙️  Starting backend worker..."
/tmp/avex-worker > "$SCRIPT_DIR/backend/worker.log" 2>&1 &
WORKER_PID=$!
echo "$WORKER_PID" > "$SCRIPT_DIR/backend/worker.pid"
debug "Worker PID: $WORKER_PID"

# Give worker a moment to start
sleep 1
if ! kill -0 "$WORKER_PID" 2>/dev/null; then
  warn "Worker exited immediately. Check $SCRIPT_DIR/backend/worker.log"
  tail -10 "$SCRIPT_DIR/backend/worker.log" 2>/dev/null
  # Continue anyway — the server can work without the worker (just no background jobs)
fi

# -----------------------------------------------------------------------------
# 8. Start the backend API server (BACKGROUND initially, for health check)
# -----------------------------------------------------------------------------
log "🚀 Starting backend API server on port ${APP_PORT:-8080}..."
/tmp/avex-server > "$SCRIPT_DIR/backend/server.log" 2>&1 &
SERVER_PID=$!
echo "$SERVER_PID" > "$SCRIPT_DIR/backend/server.pid"
debug "Server PID: $SERVER_PID"

cd "$SCRIPT_DIR"

# -----------------------------------------------------------------------------
# 9. Wait for the API server to become healthy
# -----------------------------------------------------------------------------
log "⏳ Waiting for API server to be healthy..."
for i in $(seq 1 30); do
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    err "Backend server process died during startup!"
    err "=== Server log (last 30 lines) ==="
    tail -30 "$SCRIPT_DIR/backend/server.log" 2>/dev/null
    exit 1
  fi
  if curl -sf http://127.0.0.1:8080/api/healthz >/dev/null 2>&1; then
    log "✅ Backend API is healthy"
    break
  fi
  if [ $i -eq 30 ]; then
    err "Backend API did not become healthy within 30 seconds."
    err "=== Server log (last 30 lines) ==="
    tail -30 "$SCRIPT_DIR/backend/server.log" 2>/dev/null
    err ""
    err "Common causes:"
    err "  1. Database migration failed — check the log above"
    err "  2. Port 8080 already in use — run: ./run.sh --stop && ./run.sh"
    err "  3. Missing environment variable — check the log for 'config:' errors"
    exit 1
  fi
  sleep 1
done

# -----------------------------------------------------------------------------
# 10. Print summary
# -----------------------------------------------------------------------------
echo ""
echo "============================================================" >&2
echo -e "${GREEN}  AVEX Platform is up and running${NC}" >&2
echo "============================================================" >&2
echo "" >&2
echo "  Backend API:    http://127.0.0.1:8080" >&2
echo "  Health check:   http://127.0.0.1:8080/api/healthz" >&2
echo "  API base:       http://127.0.0.1:8080/api/v1" >&2
echo "" >&2
echo "  Frontend apps (auto-started by Replit from artifact.toml):" >&2
echo "    Driver:   /driver/     (tab at bottom of Replit)" >&2
echo "    Customer: /            (tab at bottom of Replit)" >&2
echo "    Admin:    /admin/      (tab at bottom of Replit)" >&2
echo "    Merchant: /merchant/   (tab at bottom of Replit)" >&2
echo "    Support:  /support/    (tab at bottom of Replit)" >&2
echo "" >&2
echo "  Logs:" >&2
echo "    Backend server: $SCRIPT_DIR/backend/server.log" >&2
echo "    Backend worker: $SCRIPT_DIR/backend/worker.log" >&2
echo "    Redis:          /tmp/avex-redis/redis.log" >&2
echo "" >&2
echo "  Commands:" >&2
echo "    ./run.sh --status   Check status" >&2
echo "    ./run.sh --stop     Stop everything" >&2
echo "    ./seed-driver.sh    Create a test driver account" >&2
echo "============================================================" >&2
echo "" >&2

# =============================================================================
# 11. CRITICAL: Keep the script alive so Replit doesn't kill everything
# =============================================================================
# Replit's workflow monitor watches this script's process. If the script exits,
# Replit kills the entire process group — including our backend server.
# We must NOT exit. We block here, streaming the server log to stdout, until
# the server process dies or we receive SIGTERM/SIGINT.

if [ "$BACKGROUND" = "true" ]; then
  # Background mode: disown the processes and exit
  info "Backend running in background. Logs in backend/server.log"
  disown "$SERVER_PID" 2>/dev/null || true
  disown "$WORKER_PID" 2>/dev/null || true
  exit 0
fi

# Foreground mode (DEFAULT for Replit):
# Stream the server log to stdout AND block on the server process.
# This keeps the Replit workflow alive and shows real-time logs in the console.
info "Backend is running in foreground. Press Ctrl+C or Stop button to shut down."
info "Streaming server logs (live)..."

# Tail the log in background, capture its PID
tail -f "$SCRIPT_DIR/backend/server.log" 2>/dev/null &
TAIL_PID=$!

# Block until the server process exits
# When the server dies (or is killed), we'll exit and cleanup() will fire.
wait "$SERVER_PID" 2>/dev/null
SERVER_EXIT_CODE=$?

# Clean up the tail process
kill "$TAIL_PID" 2>/dev/null || true
wait "$TAIL_PID" 2>/dev/null || true

if [ $SERVER_EXIT_CODE -ne 0 ]; then
  err "Backend server exited with code $SERVER_EXIT_CODE"
  err "=== Server log (last 50 lines) ==="
  tail -50 "$SCRIPT_DIR/backend/server.log" 2>/dev/null
fi

exit $SERVER_EXIT_CODE
