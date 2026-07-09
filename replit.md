# AVEX

A multi-app delivery platform (like a regional Uber Eats): a Go backend with 12 clean-architecture modules serves five separate web apps — customer, driver, merchant, admin, support.

## Run & Operate

- Backend API: `pnpm --filter @workspace/api-server run dev` (shells out to `cd backend && go run ./cmd/server`) — served through the `/api` artifact proxy prefix
- Backend outbox worker: workflow "Backend Worker" — `cd backend && go run ./cmd/worker`
- Redis: workflow "Redis" — `redis-server --port 6379` (self-hosted via Nix package, no managed connector exists for Redis)
- Frontends: each is its own artifact/workflow — `pnpm --filter @workspace/{admin,customer,driver,merchant,support} run dev`
- Required env (set as shared env vars / secrets): `DATABASE_URL` (Replit-managed Postgres, auto-injected), `REDIS_URL`, `JWT_SECRET`, `JWT_ISSUER`, `MAPBOX_ACCESS_TOKEN`
- Go toolchain: requires Go >= 1.25 (module `go-1.25`); the old `go-1.21` module was removed to avoid PATH shadowing

## Stack

- Backend: Go 1.25, Clean Architecture (Domain→Port→Service→Repository→Transport), 12 modules (identity, orders, catalog, financial, dispatch, realtime, notifications, support, permissions, settings, audit, system/localization)
- DB: PostgreSQL (Replit-managed), per-module Goose migrations, Outbox event pattern published over Redis pub/sub
- Frontends: 5 separate Vite + React apps (admin/customer/driver/merchant/support), each its own artifact with its own `/api/v1/...` calls to the shared Go backend through the `api-server` artifact's `/api` proxy prefix

## Architecture decisions

- `api-server` artifact's dev script shells out to the real Go binary (`go run ./cmd/server`) rather than being a Node service — do not "fix" this by rewriting it as Node.
- Go reads its port from `APP_PORT`, but the artifact system injects `PORT`; `config.go` was patched so `APP_PORT` falls back to `PORT`.
- The Go server's real health checks live at root (`/health`, `/healthz`), but the artifact proxy only forwards paths under `/api`. Added `/api/health` and `/api/healthz` alias routes in `cmd/server/main.go` so the artifact's health check can reach them.
- Phone numbers must be stored in normalized local Egyptian format (`01XXXXXXXXX`, 11 digits) in the DB — `identity.domain.NewPhone` normalizes `+20.../20...` input before lookup, so any seeded/inserted phone not already in that format will silently fail login with "invalid credentials".
- Redis is a hard startup dependency for both the API server and the worker (event bus + health check) — there is no Redis connector/integration on Replit, so it runs self-hosted via the Nix `redis` package.

## Known gaps (not yet fixed)

- The merchant and support frontends call backend routes under `/api/merchant/*` and `/api/support/*` (auth, menu, tickets dashboard, etc.) that do not exist anywhere in the Go backend — these apps' login/dashboards will not work until that API surface is built server-side (or the frontends are repointed to the real routes, e.g. dispatch/orders/financial modules use different paths than the merchant app expects). This is a larger follow-up, not a one-line fix.
- No seed script exists in the repo; test accounts for each role were inserted directly into the DB (see below) to match the credential hints already hardcoded in each app's login screen.

## Product

Five apps against one Go backend:
- **Customer**: browse restaurants, order food, track delivery
- **Driver**: accept dispatch offers, navigate, mark pickup/delivery, view earnings
- **Merchant**: manage a restaurant's menu, hours, and incoming orders (currently broken — see Known gaps)
- **Admin**: platform-wide oversight (drivers, restaurants, pricing, promotions)
- **Support**: customer support ticket queue (currently broken — see Known gaps)

## User preferences

- User wants the agent to own JWT secret generation/integration end-to-end rather than being asked to supply one.
- User wants seeded test accounts across all five apps for real usability testing (not just documentation).

## Gotchas

- Always restart the relevant workflow after backend Go code changes — `go run` workflows don't hot-reload.
- After installing/uninstalling a Go language module, PATH ordering in `.replit`'s `modules` list determines which `go` binary wins; keep only one Go version installed to avoid version-mismatch confusion.
- Test credentials seeded in the DB (bcrypt cost 12), matching the hints shown on each app's login screen:
  - Customer: `01000000001` / `Passw0rd!`
  - Admin: `01000000000` / `admin123`
  - Driver: `01000000003` / `Passw0rd!`
  - Merchant: `01200000001` or `01200000005` / `123456` (login itself won't work — see Known gaps)
  - Support agent: `01500000001` or `01500000002` / `123456` (login itself won't work — see Known gaps)

## Pointers

- See the `pnpm-workspace` skill for workspace structure, TypeScript setup, and package details
- `backend/README.md` and `backend/.env.example` are the source of truth for the Go backend's env vars and module layout
