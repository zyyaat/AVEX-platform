#!/bin/bash
# AVEX Platform — Post-merge script
# Runs after every git pull/merge to ensure dependencies are up to date.
#
# IMPORTANT: This script must NOT fail. If it fails, Replit will not start
# the frontend artifacts, and the user will see HTTP 502 errors.
# We use '|| true' to ensure the script always exits 0.

# Install Node dependencies (frozen lockfile for reproducible builds)
# Don't use --frozen-lockfile here because it can fail if the lockfile
# is slightly out of sync. We regenerate if needed.
pnpm install 2>&1 || pnpm install --no-frozen-lockfile 2>&1 || true

# Note: The old 'pnpm --filter db push' command was removed because the
# lib/db package was deleted during the P0 cleanup. The Go backend now
# manages its own database migrations via goose (run automatically by
# the backend server on startup — see backend/cmd/server/main.go).

exit 0
