#!/bin/bash
# Stop all running processes on Replit and start only what we need
echo "🧹 Stopping everything..."
pkill -f "vite" 2>/dev/null || true
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
pkill -f "node" 2>/dev/null || true
sleep 2

echo "📦 Starting PostgreSQL + Redis..."
cd backend
docker compose up -d postgres redis 2>/dev/null || true
sleep 5

echo "🔧 Starting Backend..."
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="avex-secret-key-change-in-production-32chars"
export JWT_ISSUER="avex"
export MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"
cd /home/runner/workspace/backend
exec go run ./cmd/server
' > /home/runner/workspace/backend/server.log 2>&1 &
disown

echo "🔧 Starting Worker..."
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
cd /home/runner/workspace/backend
exec go run ./cmd/worker
' > /home/runner/workspace/backend/worker.log 2>&1 &
disown

# Wait for backend
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ Backend ready"
    break
  fi
  sleep 1
done

echo ""
echo "✅ Done! Backend on :8080"
echo "Now click Run to start the driver app"
