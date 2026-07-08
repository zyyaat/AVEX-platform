#!/bin/bash
# AVEX Backend — Start API Server
# Usage: ./backend/start.sh

export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="avex-secret-key-change-in-production-32chars"
export JWT_ISSUER="avex"
export MAPBOX_ACCESS_TOKEN="MAPBOX_PUBLIC_TOKEN_PLACEHOLDER"

# Kill any existing server on port 8080
kill $(lsof -t -i:8080) 2>/dev/null

cd "$(dirname "$0")"
nohup go run ./cmd/server > server.log 2>&1 &
echo "✅ Server started on port 8080 (PID: $!)"
sleep 3
curl -s http://localhost:8080/health/live && echo "" || echo "❌ Server failed to start — check server.log"
