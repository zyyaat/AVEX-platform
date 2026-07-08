#!/bin/bash
# AVEX Backend — Start server in background (survives terminal close)
# Usage: ./start-backend.sh

# Kill old server
pkill -f "cmd/server" 2>/dev/null || true
pkill -f "cmd/worker" 2>/dev/null || true
sleep 1

# Start server in background (truly detached)
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="avex-secret-key-change-in-production-32chars"
export JWT_ISSUER="avex"
export MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"
cd backend
exec go run ./cmd/server
' > backend/server.log 2>&1 &
disown

# Start worker in background
nohup bash -c '
export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
cd backend
exec go run ./cmd/worker
' > backend/worker.log 2>&1 &
disown

# Wait for server
for i in $(seq 1 30); do
  if curl -s http://localhost:8080/health/live > /dev/null 2>&1; then
    echo "✅ Backend ready — http://localhost:8080"
    echo "📄 Logs: tail -f backend/server.log"
    echo "🛑 Stop: pkill -f cmd/server; pkill -f cmd/worker"
    exit 0
  fi
  sleep 1
done

echo "❌ Backend failed — check backend/server.log"
tail -20 backend/server.log
