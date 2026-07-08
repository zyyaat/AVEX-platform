#!/bin/bash
# AVEX Driver App — Start Dev Server
# Usage: ./artifacts/driver/start.sh

cd "$(dirname "$0")"
nohup pnpm dev > driver.log 2>&1 &
echo "✅ Driver app started (PID: $!)"
