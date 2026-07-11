# Changelog

All notable changes to the AVEX Platform are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Docker setup:** Multi-stage `Dockerfile` for backend (server + worker in one image, role selected via `APP_ROLE` env var).
- **Docker setup:** Multi-stage `Dockerfile` + `nginx.conf` for each frontend app (driver, customer, admin, merchant, support). nginx serves the SPA, proxies `/api/*` to the backend (with WebSocket support for `/api/v1/ws`).
- **Docker setup:** `docker-compose.yml` for local development — brings up PostgreSQL, Redis, backend (server + worker), and all 5 frontend apps. Endpoints:
  - PostgreSQL: `localhost:5432`
  - Redis: `localhost:6379`
  - Backend API: `http://localhost:8080`
  - Driver: `http://localhost:3001` · Customer: `:3002` · Admin: `:3003` · Merchant: `:3004` · Support: `:3005`
- **CI/CD:** `docker` stage in `.gitlab-ci.yml` — builds and pushes images to GitLab Container Registry on main pushes. Tags: `:latest` + `:<short-sha>`.
- **CI/CD:** `docker-build` matrix job in GitHub Actions (builds all 6 images, no push).
- **CI/CD:** `security:backend` job — `gosec` scan on Go code (high severity, medium confidence).
- **CI/CD:** `security:frontend` job — `pnpm audit` on workspace (high+ severity).
- **CI/CD:** `e2e:driver` job — Playwright E2E tests for the driver app with mocked API.
- **E2E tests:** Playwright setup for driver app (`playwright.config.ts` + `e2e/smoke.spec.ts`). 7 tests covering:
  - App boots (no white screen)
  - Login page renders phone + password fields
  - Login form validation (rejects empty input)
  - Successful login navigates to home
  - Home page renders map container
  - Login error shown on invalid credentials
  - App handles API failure gracefully

### Changed
- **Backend Dockerfile:** Fixed `HEALTHCHECK` to use `/api/healthz` (the actual endpoint) instead of non-existent `/health/live`.

### Security
- **CI:** All Docker images now run as non-root user (`avex`) in the runtime stage.
- **CI:** Security scanning fails the pipeline on HIGH or CRITICAL findings (frontend audit is `allow_failure: true` for now — many transitive deps).

## [0.2.0] — 2026-07-11

### Added
- **CI/CD:** GitLab CI pipeline (`.gitlab-ci.yml`) with 5 stages: `lint → test → build → integration → release`. 7 jobs total.
- **CI/CD:** GitHub Actions mirror (`.github/workflows/ci.yml`) with 5 jobs: `backend-lint`, `backend-test`, `backend-build`, `backend-integration`, `frontend-test`.
- **CI/CD:** `release:binaries` job — builds production linux/amd64 binaries on main pushes, attaches as artifacts (30-day retention).
- **Tests:** Backend integration tests — `http_smoke_test.go` with 3 tests covering `/api/healthz`, `/api/health`, and 404 handling. Tag-gated (`//go:build integration`).
- **Tests:** Vitest infrastructure for all 5 frontend apps. 37 tests covering API client behavior (token storage, `{data}` unwrapping, 401 redirect, error extraction, WebSocket URL building).
- **Governance:** `LICENSE` (MIT), `CONTRIBUTING.md` (repo layout, dev setup, git workflow, coding standards, testing, CI, releasing, PR checklist), `CODE_OF_CONDUCT.md` (Contributor Covenant 2.0).
- **Docs:** Comprehensive `README.md` with architecture overview, quick start, and env var reference.

### Changed
- **CI/CD:** Switched `frontend-test` to `--frozen-lockfile` (was `--no-frozen-lockfile`).
- **Lockfile:** Regenerated `pnpm-lock.yaml` to include `vitest` + `happy-dom` across all 5 apps.
- **Workspace:** `pnpm-workspace.yaml` now allows esbuild build (was blocking `pnpm install`).

### Removed
- **Dead code:** `lib/api-spec/` (OpenAPI spec + orval config, unused).
- **Dead code:** `lib/api-client-react/` (generated Orval client, unused).
- **Dead code:** `lib/api-zod/` (generated Zod schemas, unused).
- **Dead code:** `@workspace/api-client-react` dependency from all 5 apps' `package.json`.
- **Dead code:** `lib/*` path references from root + app `tsconfig.json` files.
- **Dead code:** `lib/*` and `lib/integrations/*` entries from `pnpm-workspace.yaml`.

### Fixed
- **TypeScript:** Exported `ButtonProps` type from `button.tsx` in admin/merchant/support (was causing `TS2305` in `pagination.tsx`).
- **TypeScript:** Fixed `size` prop duplication in `pagination.tsx` (TS2783) — now uses `size ?? 'default'`.
- **TypeScript:** Added `merchant` field to merchant login response type.
- **TypeScript:** Added `agent` field to support login response type.
- **TypeScript:** Added `agentId` field to support `getTickets` response type.
- **TypeScript:** Made `agentId` parameter in `assignTicket` optional (default `''`).
- **TypeScript:** Removed redundant `?? '0.0'` fallback on `toFixed()` (TS2881).

## [0.1.0] — 2026-07-10

### Added
- **Backend (Go):** Complete modular monolith with 12 modules (Identity, Orders, Catalog, Financial, Dispatch, Realtime, Notifications, Support, Permissions, Settings, Audit, System, Localization).
- **Backend (Go):** 14 implementation phases completed, 707 tests passing.
- **Backend (Go):** PostgreSQL + Redis + WebSocket + Outbox pattern + OpenTelemetry.
- **Frontend (React):** 5 Vite + TypeScript + Tailwind + shadcn/ui apps (driver, admin, customer, merchant, support).
- **Frontend (React):** Mapbox GL JS loaded from CDN (avoids Vite worker bundling issues).
- **Frontend (React):** JWT auth with Bearer tokens, 401 redirects.
- **Frontend (React):** WebSocket via `/api/v1/ws?token=...`.
- **Frontend (React):** Vite dev server proxy for `/api/*` → `http://localhost:8080`.
- **Backend (Go):** Driver registration endpoint (`POST /api/v1/auth/driver/register`).
- **Backend (Go):** Seed script for test driver (`seed-driver.sh`).

### Known Issues
- Driver app white screen on Replit — needs investigation of artifact proxy vs Vite proxy interaction.
- Only driver app has Vite proxy config — other apps may need it too.

---

## Versioning Policy

We follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **MAJOR** (X.0.0): Breaking API changes, schema migrations that aren't backward-compatible.
- **MINOR** (0.X.0): New features, new endpoints, backward-compatible schema changes.
- **PATCH** (0.0.X): Bug fixes, performance improvements, no new features.

## Release Process

1. Update this `CHANGELOG.md` under `[Unreleased]` → move to a new `[X.Y.Z] — YYYY-MM-DD` section.
2. Tag with `git tag vX.Y.Z && git push --tags`.
3. The CI `release:binaries` job will produce downloadable artifacts.
4. (Planned) Create a GitHub Release attaching the binaries + changelog entry.
