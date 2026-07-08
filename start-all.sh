#!/bin/bash
# AVEX Backend — Start Everything (Server + Worker + Driver App)
# Runs in background with nohup — survives terminal close.
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
cd /home/runner/workspace/backend 2>/dev/null || cd backend
docker compose up -d postgres redis 2>/dev/null || true

# Wait for PostgreSQL
for i in $(seq 1 15); do
  if docker exec avex-postgres pg_isready -U avex -d avex_dev > /dev/null 2>&1; then
    echo "✅ PostgreSQL ready"
    break
  fi
  sleep 1
done

# ===== 2. Kill old processes =====
echo "🧹 Cleaning up..."
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
sleep 1

# ===== 3. Start Server (truly background) =====
echo "🔧 Starting API Server..."
nohup bash -c 'source /dev/null; export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable" REDIS_URL="redis://localhost:6379/0" JWT_SECRET="avex-secret-key-change-in-production-32chars" JWT_ISSUER="avex" MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"; cd /home/runner/workspace/backend 2>/dev/null || cd backend; exec go run ./cmd/server' > server.log 2>&1 &
disown

# Wait for server
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ API Server ready"
    break
  fi
  sleep 1
done

if ! curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
  echo "❌ Server failed — check server.log"
  tail -20 server.log
  exit 1
fi

# ===== 4. Start Worker (truly background) =====
echo "🔧 Starting Worker..."
nohup bash -c 'export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable" REDIS_URL="redis://localhost:6379/0"; cd /home/runner/workspace/backend 2>/dev/null || cd backend; exec go run ./cmd/worker' > worker.log 2>&1 &
disown
echo "✅ Worker started"

# ===== 5. Start Driver App (truly background) =====
echo "📱 Starting Driver App..."
nohup bash -c 'cd /home/runner/workspace/artifacts/driver 2>/dev/null || cd artifacts/driver; exec pnpm dev' > driver.log 2>&1 &
disown

sleep 3

echo ""
echo "============================================"
echo "  🎉 AVEX is running!"
echo "============================================"
echo "  API:     http://localhost:8080"
echo "  Health:  http://localhost:8080/health"
echo "============================================"
echo ""
echo "Logs:"
echo "  tail -f backend/server.log"
echo "  tail -f backend/worker.log"  
echo "  tail -f artifacts/driver/driver.log"
echo ""
echo "Stop:  pkill -f 'cmd/server'; pkill -f 'cmd/worker'; pkill -f vite"
