#!/bin/bash
# AVEX — Start Everything (Backend + Worker + Driver App)
# Usage: ./run.sh
# This script starts everything in the background and keeps it running.

echo "🚀 Starting AVEX..."

# ===== 1. Kill everything first =====
echo "🧹 Killing old processes..."
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
pkill -f "node.*driver" 2>/dev/null || true
sleep 2

# ===== 2. Start Docker =====
echo "📦 Starting PostgreSQL + Redis..."
cd /home/runner/workspace/backend 2>/dev/null || cd backend
docker compose up -d postgres redis 2>/dev/null || true

# Wait for PostgreSQL
echo "⏳ Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if docker exec avex-postgres psql -U avex -d avex_dev -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ PostgreSQL ready"
    break
  fi
  sleep 1
done

# Wait for Redis
for i in $(seq 1 10); do
  if docker exec avex-redis redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis ready"
    break
  fi
  sleep 1
done

# ===== 3. Start Backend Server =====
echo "🔧 Starting API Server..."
WORKSPACE=$(cd /home/runner/workspace 2>/dev/null && pwd || cd .. && pwd)
nohup bash -c "
export DATABASE_URL='postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
export JWT_SECRET='avex-secret-key-change-in-production-32chars'
export JWT_ISSUER='avex'
export MAPBOX_ACCESS_TOKEN='MAPBOX_PUBLIC_TOKEN_PLACEHOLDER'
cd '$WORKSPACE/backend'
exec go run ./cmd/server
" > "$WORKSPACE/backend/server.log" 2>&1 &
disown
SERVER_PID=$!

# Wait for server
for i in $(seq 1 40); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ API Server ready (PID: $SERVER_PID)"
    break
  fi
  if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server process died!"
    tail -20 "$WORKSPACE/backend/server.log"
    exit 1
  fi
  sleep 1
done

if ! curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
  echo "❌ Server failed to start!"
  tail -20 "$WORKSPACE/backend/server.log"
  exit 1
fi

# ===== 4. Start Worker =====
echo "🔧 Starting Worker..."
nohup bash -c "
export DATABASE_URL='postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
cd '$WORKSPACE/backend'
exec go run ./cmd/worker
" > "$WORKSPACE/backend/worker.log" 2>&1 &
disown
WORKER_PID=$!
echo "✅ Worker started (PID: $WORKER_PID)"

# ===== 5. Start Driver App =====
echo "📱 Starting Driver App..."
cd "$WORKSPACE/artifacts/driver"
export PORT=5174
export BASE_PATH=/
nohup bash -c "
export PORT=5174
export BASE_PATH=/
cd '$WORKSPACE/artifacts/driver'
exec pnpm dev
" > "$WORKSPACE/artifacts/driver/driver.log" 2>&1 &
disown
DRIVER_PID=$!

# Wait for driver app
for i in $(seq 1 15); do
  sleep 1
done

echo ""
echo "============================================"
echo "  🎉 AVEX is running!"
echo "============================================"
echo "  API Server:  http://localhost:8080"
echo "  Health:      http://localhost:8080/health"
echo "  Driver App:  http://localhost:5174"
echo "============================================"
echo ""
echo "Logs:"
echo "  Server:  tail -f $WORKSPACE/backend/server.log"
echo "  Worker:  tail -f $WORKSPACE/backend/worker.log"
echo "  Driver:  tail -f $WORKSPACE/artifacts/driver/driver.log"
echo ""
echo "Stop:  pkill -f 'cmd/server'; pkill -f 'cmd/worker'; pkill -f vite"
echo ""
echo "Test login:"
echo "  curl -s -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{\"phone\":\"01012345678\",\"password\":\"12345678\"}'"
