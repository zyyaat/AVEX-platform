#!/bin/bash
# AVEX Platform — Post-merge script
# Runs after every git pull/merge to ensure dependencies are up to date.
set -e

# Install Node dependencies (frozen lockfile for reproducible builds)
pnpm install --frozen-lockfile

# Note: The old 'pnpm --filter db push' command was removed because the
# lib/db package was deleted during the P0 cleanup. The Go backend now
# manages its own database migrations via goose (run automatically by
# the backend server on startup — see backend/cmd/server/main.go).
