#!/bin/bash
# AVEX Backend — Start Everything (PostgreSQL + Redis + Server + Worker + Driver App)
# Usage: ./start-all.sh

echo "🚀 Starting AVEX Backend..."

# ===== Environment Variables =====
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="avex-secret-key-change-in-production-32chars"
export JWT_ISSUER="avex"
export MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"

# ===== 1. Start Docker (PostgreSQL + Redis) =====
echo "📦 Starting PostgreSQL + Redis..."
cd backend
docker compose up -d postgres redis 2>/dev/null || true

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL..."
for i in $(seq 1 15); do
  if docker exec avex-postgres pg_isready -U avex -d avex_dev > /dev/null 2>&1; then
    echo "✅ PostgreSQL ready"
    break
  fi
  sleep 1
done

# ===== 2. Kill any old processes =====
echo "🧹 Cleaning up old processes..."
kill $(lsof -t -i:8080) 2>/dev/null || true
kill $(lsof -t -i:5173 -i:5174) 2>/dev/null || true

# ===== 3. Start Backend Server =====
echo "🔧 Starting API Server..."
nohup go run ./cmd/server > server.log 2>&1 &
SERVER_PID=$!

# Wait for server to be ready (up to 30 seconds)
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ API Server ready (PID: $SERVER_PID)"
    break
  fi
  # Check if process died
  if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ API Server process died!"
    echo "=== Server Log (last 30 lines) ==="
    tail -30 server.log
    exit 1
  fi
  sleep 1
done

# Final check
if ! curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
  echo "❌ API Server failed to start after 30 seconds!"
  echo "=== Server Log (last 30 lines) ==="
  tail -30 server.log
  exit 1
fi

# ===== 4. Start Worker =====
echo "🔧 Starting Outbox Worker..."
nohup go run ./cmd/worker > worker.log 2>&1 &
WORKER_PID=$!
echo "✅ Worker started (PID: $WORKER_PID)"

# ===== 5. Start Driver App =====
echo "📱 Starting Driver App..."
cd ../artifacts/driver
nohup pnpm dev > driver.log 2>&1 &
DRIVER_PID=$!

# Wait for driver app
for i in $(seq 1 10); do
  sleep 1
done

echo ""
echo "============================================"
echo "  🎉 AVEX is running!"
echo "============================================"
echo "  API Server:  http://localhost:8080"
echo "  Health:      http://localhost:8080/health"
echo "  Driver App:  check driver.log for URL"
echo "============================================"
echo ""
echo "Logs:"
echo "  Server:  tail -f backend/server.log"
echo "  Worker:  tail -f backend/worker.log"
echo "  Driver:  tail -f artifacts/driver/driver.log"
echo ""
echo "Stop all:  kill $SERVER_PID $WORKER_PID $DRIVER_PID"
