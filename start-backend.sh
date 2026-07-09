#!/bin/bash
# AVEX Backend — Start server in background (survives terminal close)
# Usage: ./start-backend.sh

# Kill old processes
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
sleep 1

# ===== 1. Start Docker (PostgreSQL + Redis) =====
echo "📦 Starting PostgreSQL + Redis..."
cd backend
docker compose up -d postgres redis 2>/dev/null || true

# Wait for PostgreSQL to be ready (up to 30 seconds)
echo "⏳ Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if docker exec avex-postgres pg_isready -U avex -d avex_dev > /dev/null 2>&1; then
    echo "✅ PostgreSQL ready"
    break
  fi
  if [ $i -eq 30 ]; then
    echo "❌ PostgreSQL not ready after 30s"
    docker compose logs postgres | tail -10
    exit 1
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

# ===== 2. Start server in background =====
echo "🔧 Starting API Server..."
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="avex-secret-key-change-in-production-32chars"
export JWT_ISSUER="avex"
export MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"
cd backend
exec go run ./cmd/server
' > server.log 2>&1 &
disown

# Start worker in background
echo "🔧 Starting Worker..."
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
cd backend
exec go run ./cmd/worker
' > worker.log 2>&1 &
disown

# Wait for server to be ready (up to 30 seconds)
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo ""
    echo "✅ Backend ready — http://localhost:8080"
    echo "📄 Server log: tail -f backend/server.log"
    echo "📄 Worker log: tail -f backend/worker.log"
    echo "🛑 Stop: pkill -f cmd/server; pkill -f cmd/worker"
    exit 0
  fi
  sleep 1
done

echo "❌ Backend failed — check backend/server.log"
tail -20 backend/server.log
