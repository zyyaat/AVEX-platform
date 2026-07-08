#!/bin/bash
# AVEX Backend — Start Outbox Worker
# Usage: ./backend/start-worker.sh

export DATABASE_URL="postgres://avex:avex@localhost:5432/avex_dev?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"

cd "$(dirname "$0")"
nohup go run ./cmd/worker > worker.log 2>&1 &
echo "✅ Worker started (PID: $!)"
