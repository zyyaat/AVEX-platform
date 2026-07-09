#!/bin/bash
# AVEX — Start Everything (Backend + Worker + Driver App)
# Usage: ./run.sh

echo "🚀 Starting AVEX..."

# Kill everything
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
sleep 2

# Determine workspace path
WORKSPACE="/home/runner/workspace"
if [ ! -d "$WORKSPACE" ]; then
  WORKSPACE="$(cd "$(dirname "$0")" && pwd)"
fi

# Start Docker
echo "📦 Starting PostgreSQL + Redis..."
cd "$WORKSPACE/backend"
docker compose up -d postgres redis 2>/dev/null || true

echo "⏳ Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if docker exec avex-postgres psql -U avex -d avex_dev -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ PostgreSQL ready"
    break
  fi
  sleep 1
done

# Start Backend
echo "🔧 Starting API Server..."
nohup bash -c "
export DATABASE_URL='postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
export JWT_SECRET='avex-secret-key-change-in-production-32chars'
export JWT_ISSUER='avex'
export MAPBOX_ACCESS_TOKEN='MAPBOX_PUBLIC_TOKEN_PLACEHOLDER'
cd $WORKSPACE/backend
exec go run ./cmd/server
" > $WORKSPACE/backend/server.log 2>&1 &
disown

# Start Worker
echo "🔧 Starting Worker..."
nohup bash -c "
export DATABASE_URL='postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
cd $WORKSPACE/backend
exec go run ./cmd/worker
" > $WORKSPACE/backend/worker.log 2>&1 &
disown

# Wait for backend
for i in $(seq 1 40); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ Backend ready"
    break
  fi
  sleep 1
done

# Start Driver App
echo "📱 Starting Driver App..."
nohup bash -c "
export PORT=5174
export BASE_PATH=/
cd $WORKSPACE/artifacts/driver
exec pnpm dev
" > $WORKSPACE/artifacts/driver/driver.log 2>&1 &
disown

sleep 5

echo ""
echo "============================================"
echo "  🎉 AVEX is running!"
echo "============================================"
echo "  Backend:  http://localhost:8080"
echo "  Driver:   http://localhost:5174"
echo "============================================"
echo ""
echo "Logs:"
echo "  tail -f $WORKSPACE/backend/server.log"
echo "  tail -f $WORKSPACE/artifacts/driver/driver.log"
echo ""
echo "Stop: pkill -f 'cmd/server'; pkill -f 'cmd/worker'; pkill -f vite"
