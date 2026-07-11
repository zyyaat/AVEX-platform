# AVEX Platform — Replit Setup Guide

This guide walks you through running the entire AVEX platform (backend + 5 frontend apps) on Replit.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Initial Setup](#2-initial-setup)
3. [Configure Secrets](#3-configure-secrets)
4. [Run the Platform](#4-run-the-platform)
5. [Access the Apps](#5-access-the-apps)
6. [Seed Test Data](#6-seed-test-data)
7. [Daily Operations](#7-daily-operations)
8. [Troubleshooting](#8-troubleshooting)

---

## 1. Prerequisites

- A [Replit](https://replit.com) account (free tier works)
- A [Mapbox](https://account.mapbox.com/) account (free tier: 50k loads/month)

The Replit repl auto-installs:
- Go 1.25
- Node.js 20
- PostgreSQL 16 (as a Replit module)
- Redis (via Nix)
- pnpm (via corepack)

---

## 2. Initial Setup

### Option A: Import from GitLab (recommended)

1. Go to https://replit.com/~ and click **"Create Repl"**
2. Select **"Import from GitHub/GitLab"**
3. Paste the GitLab URL: `https://gitlab.com/zyyat-group/Zyyat-project.git`
4. Select the `main` branch
5. Click **"Import"**

Replit will:
- Clone the repo
- Detect the `pnpm` workspace from `.replit`
- Install all Node dependencies via `scripts/post-merge.sh`
- Start PostgreSQL (module auto-starts)

### Option B: Use an existing Replit

If you already have a Repl with the code:

```bash
git pull origin main
pnpm install
```

---

## 3. Configure Secrets

**Before running anything**, you must set two secrets.

See [`REPLIT_SECRETS.md`](./REPLIT_SECRETS.md) for detailed instructions.

**Quick version:**

1. Click the **lock icon** 🔒 in the Replit sidebar
2. Add secret `JWT_SECRET`:
   - Generate a value by running `openssl rand -hex 32` in the shell
3. Add secret `MAPBOX_ACCESS_TOKEN`:
   - Get a public token from https://account.mapbox.com/access-tokens/

Without these secrets, `run.sh` will refuse to start with a clear error message.

---

## 4. Run the Platform

### Start everything

Click the green **"Run"** button at the top of Replit.

This executes the `Backend` workflow (defined in `.replit`), which runs `bash run.sh`.

`run.sh` does the following automatically:

1. ✅ Validates that `JWT_SECRET` and `MAPBOX_ACCESS_TOKEN` are set
2. ✅ Waits for PostgreSQL (Replit module) to be ready
3. ✅ Creates the `avex_dev` database + `avex` role if they don't exist
4. ✅ Starts Redis as a background daemon (via Nix)
5. ✅ Compiles the Go backend (`cmd/server` + `cmd/worker`)
6. ✅ Starts the API server on port 8080
7. ✅ Starts the worker process
8. ✅ Waits for the API to become healthy
9. ✅ Prints a summary with all URLs

### Frontend apps

The 5 frontend apps are **NOT** started by `run.sh`. Replit auto-discovers them from their `artifact.toml` files and starts each one as a separate **artifact** (visible as tabs in the bottom panel of the Replit workspace).

You'll see these tabs appear:
- **Driver App** (port 19574, path `/driver/`)
- **Customer App** (port 24173, path `/`)
- **Admin App** (port 23744, path `/admin/`)
- **Merchant App** (port 23719, path `/merchant/`)
- **Support App** (port 22770, path `/support/`)

---

## 5. Access the Apps

Once everything is running, click any of the bottom tabs in Replit to open that app in the webview. Or, use the URLs:

| App | URL (Replit webview) | Purpose |
|-----|---------------------|---------|
| Driver | `/driver/` | Driver mobile-first app |
| Customer | `/` | Customer ordering app |
| Admin | `/admin/` | Admin dashboard |
| Merchant | `/merchant/` | Merchant catalog & orders |
| Support | `/support/` | Support ticket system |

**Backend API** (not user-facing, but useful for debugging):
- Health check: `https://<your-repl>.replit.app/api/healthz`
- API base: `https://<your-repl>.replit.app/api/v1`

All frontend apps proxy `/api/*` requests to the backend via the Vite dev server proxy (configured in each app's `vite.config.ts`).

---

## 6. Seed Test Data

To create a test driver account (so you can log in to the Driver app):

```bash
./seed-driver.sh
```

This creates:
- Phone: `01012345678`
- Password: `12345678`
- Name: `Ahmed`

You can override these:

```bash
./seed-driver.sh 01099999999 mypassword Mohamed
```

The script will:
1. Register the driver via the API
2. Auto-verify them
3. Create a dispatch record
4. Print the driver ID + auth token

---

## 7. Daily Operations

### Check status

```bash
./run.sh --status
```

Output:
```
AVEX Platform Status
====================
  PostgreSQL:  ✅ running
  Redis:       ✅ running
  Backend API: ✅ running
  Worker:      ✅ running
```

### Stop everything

```bash
./run.sh --stop
```

This stops the backend server, worker, and Redis. PostgreSQL is left running (it's managed by Replit).

### View logs

```bash
# Backend server
tail -f backend/server.log

# Backend worker
tail -f backend/worker.log

# Redis
tail -f /tmp/avex-redis/redis.log
```

### Start in background (no console output)

```bash
./run.sh --background
```

Useful when you want to run other commands in the shell while the backend runs.

---

## 8. Troubleshooting

### "Missing required environment variables: JWT_SECRET MAPBOX_ACCESS_TOKEN"

→ See [Step 3: Configure Secrets](#3-configure-secrets).

### "PostgreSQL did not become ready within 30 seconds"

Replit's PostgreSQL module auto-starts on first use. Try:

1. Open the **Database** tab in the Replit sidebar
2. Click on PostgreSQL
3. Wait for it to show "Running"
4. Re-run `./run.sh`

If that doesn't work, manually trigger it:

```bash
psql -h 127.0.0.1 -U postgres -c "SELECT 1"
```

### "Redis did not start"

Check the Redis log:

```bash
cat /tmp/avex-redis/redis.log
```

Common causes:
- Port 6379 already in use: `pkill redis-server && ./run.sh`
- Permission error: `rm -rf /tmp/avex-redis && ./run.sh`

### Frontend app shows white screen

1. Check the **Console** tab in the Replit webview for errors
2. Make sure the backend is running: `./run.sh --status`
3. Make sure the Vite proxy is working: open `<app-url>/api/healthz` in the browser — should return `{"status":"ok"}`
4. Check the **Server** logs panel for the specific app

### Login fails with "Invalid credentials"

For driver login:
- Phone must be exactly 11 digits (Egyptian format: `01XXXXXXXXX`)
- Password is set during registration
- Use `./seed-driver.sh` to create a known test account

### Map doesn't load in Driver app

1. Verify `MAPBOX_ACCESS_TOKEN` is set (Replit Secrets)
2. Open browser dev tools → Network tab → look for requests to `api.mapbox.com`
3. If requests fail with 401, the token is invalid or expired

### Port conflicts

If you see "address already in use":

```bash
./run.sh --stop
sleep 2
./run.sh
```

### Build errors after pulling new code

```bash
# Clean Go build cache
cd backend && go clean -cache && cd ..

# Reinstall Node deps
pnpm install

# Restart
./run.sh
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Replit Workspace                                            │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  run.sh (started by "Run" button)                   │    │
│  │  ├── PostgreSQL (module, port 5432)                 │    │
│  │  ├── Redis (Nix daemon, port 6379)                  │    │
│  │  ├── avex-server (Go binary, port 8080)             │    │
│  │  └── avex-worker (Go binary, outbox processor)      │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Artifact tabs (auto-started by Replit)             │    │
│  │  ├── Driver   → Vite dev (port 19574, /driver/)     │    │
│  │  ├── Customer → Vite dev (port 24173, /)            │    │
│  │  ├── Admin    → Vite dev (port 23744, /admin/)      │    │
│  │  ├── Merchant → Vite dev (port 23719, /merchant/)   │    │
│  │  └── Support  → Vite dev (port 22770, /support/)    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  Each Vite app proxies /api/* → http://127.0.0.1:8080       │
└─────────────────────────────────────────────────────────────┘
```

---

## File Reference

| File | Purpose |
|------|---------|
| `.replit` | Replit project config: modules, workflow, Nix packages, env vars |
| `replit.nix` | Additional Nix packages (redis, curl, jq, gettext) |
| `run.sh` | Unified start/stop/status script |
| `seed-driver.sh` | Creates a test driver account |
| `.env.example` | Template for local development (not used on Replit) |
| `REPLIT_SECRETS.md` | How to set JWT_SECRET + MAPBOX_ACCESS_TOKEN |
| `artifacts/<app>/.replit-artifact/artifact.toml` | Per-app Replit artifact config |
| `artifacts/<app>/vite.config.ts` | Vite dev server config (includes API proxy) |
