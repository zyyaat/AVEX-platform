# Contributing to AVEX Platform

Thank you for your interest in contributing to AVEX! This document outlines the workflow we follow to keep the codebase healthy, secure, and shippable.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Repository Layout](#repository-layout)
- [Development Environment](#development-environment)
- [Git Workflow](#git-workflow)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Continuous Integration](#continuous-integration)
- [Releasing](#releasing)
- [Issue Reporting](#issue-reporting)
- [Pull Request Checklist](#pull-request-checklist)

---

## Code of Conduct

By participating in this project you agree to abide by our [Code of Conduct](./CODE_OF_CONDUCT.md). Please be respectful, professional, and inclusive in all interactions.

---

## Repository Layout

```
.
├── backend/                  # Go modular monolith (12 modules)
│   ├── cmd/
│   │   ├── server/           # HTTP API server entrypoint
│   │   ├── worker/           # Background job worker entrypoint
│   │   └── migrate/          # Migration runner
│   ├── internal/
│   │   ├── modules/          # Domain modules (identity, orders, ...)
│   │   ├── platform/         # Cross-cutting: config, db, logger, bus
│   │   └── integration/      # Cross-module integration tests
│   └── migrations/           # Embedded SQL goose migrations
├── artifacts/                # React frontend apps (Vite + TS)
│   ├── driver/               # Driver mobile-first app
│   ├── admin/                # Admin dashboard
│   ├── customer/             # Customer ordering app
│   ├── merchant/             # Merchant catalog & orders
│   └── support/              # Support ticket system
├── scripts/                  # Workspace scripts (seed, ops)
├── .gitlab-ci.yml            # CI pipeline
├── pnpm-workspace.yaml       # pnpm workspace config
└── tsconfig.base.json        # Shared TS compiler options
```

---

## Development Environment

### Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | Required for `backend/` |
| Node.js | 22 LTS | Required for `artifacts/` |
| pnpm | 11+ | `npm i -g pnpm` |
| PostgreSQL | 16+ | Local dev DB |
| Redis | 7+ | Realtime + outbox |

### Setup

```bash
# Backend
cd backend
cp .env.example .env   # then edit DB / Redis / JWT secrets
go mod download
go run ./cmd/server

# Frontend (from repo root)
pnpm install
pnpm --filter driver dev      # or admin / customer / merchant / support
```

---

## Git Workflow

We use a **trunk-based development** model with short-lived feature branches.

1. **Branch naming:** `<type>/<scope>-<short-desc>`
   - Examples: `feat/orders-batch-cancel`, `fix/driver-white-screen`, `ci/add-gitlab-ci`, `docs/update-readme`
   - Types: `feat`, `fix`, `chore`, `ci`, `docs`, `test`, `refactor`, `perf`, `security`

2. **Commit messages** follow [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat(orders): add batch cancel endpoint

   - Adds POST /api/v1/orders/batch/cancel
   - Accepts up to 100 order IDs per call
   - Emits OrderCancelled event per cancelled order

   Closes #42
   ```

3. **Pull/Merge Requests:**
   - Target `main` directly for small/medium changes.
   - Keep MRs under ~500 LOC when possible. Split large work into stacked MRs.
   - Require at least one approval for non-trivial changes.

4. **Rebasing:** Prefer `git rebase main` over merge commits when syncing long-lived branches. Squash commits before merging if they're noisy.

---

## Coding Standards

### Go (backend)

- Run `gofmt -w .` before every commit. CI will fail on unformatted files.
- Run `go vet ./...` — fix all warnings before pushing.
- **Architecture:** Each module follows `domain → port → service → repository → transport`. Don't import another module's internals; go through its `port` package.
- **Errors:** Wrap with `fmt.Errorf("operation: %w", err)` at boundaries. Use typed errors in `domain/errors.go` for sentinel cases.
- **Logging:** Use `slog` (structured). Never `fmt.Println` in production code.
- **Context:** Pass `context.Context` as the first argument to every function that does I/O.
- **Migrations:** Every schema change gets a new goose migration file. Never edit existing migrations.

### TypeScript / React (frontend)

- Strict mode is on. Don't suppress with `any` unless absolutely necessary — prefer `unknown` + type narrowing.
- **API client:** All HTTP calls go through `src/lib/api.ts`. Don't use raw `fetch` in components.
- **State:** Use React Query for server state. Use Zustand for client-only state.
- **Forms:** `react-hook-form` + `zod` for validation.
- **Styling:** Tailwind CSS only. No CSS modules, no styled-components.
- **Path aliases:** Use `@/` (configured in `tsconfig.json` and `vite.config.ts`).

---

## Testing

### Backend

```bash
# Unit tests (no DB required)
cd backend && go test -race -count=1 ./...

# Integration tests (require Postgres + Redis)
DATABASE_URL=postgres://avex:avex@localhost:5432/avex_test?sslmode=disable \
REDIS_URL=redis://localhost:6379/0 \
JWT_SECRET=test-secret-at-least-32-characters-long \
go test -tags=integration -count=1 -v ./internal/integration/...
```

- **Race detector** is mandatory in CI (`-race` flag).
- Aim for ≥80% coverage on `domain/` and `service/` layers.
- Integration tests live in `internal/integration/` and are gated by the `integration` build tag.

### Frontend

```bash
pnpm -r --if-present test     # runs vitest across all apps
pnpm --filter driver test     # single app
```

- Vitest + happy-dom for unit/component tests.
- API client tests live in `src/lib/api.test.ts` and cover: token storage, `{data}` unwrapping, 401 redirect, error extraction, WS URL building.
- E2E (Playwright) is planned but not yet set up.

---

## Continuous Integration

CI is defined in [`.gitlab-ci.yml`](./.gitlab-ci.yml) and runs on every MR and `main` push.

**Pipeline stages:**

1. `lint` — `go vet` + `gofmt` check (backend only)
2. `test` — backend unit tests + frontend Vitest tests (parallel)
3. `build` — `go build` server + worker binaries
4. `integration` — backend integration tests with Postgres + Redis services
5. `release` — production binary build (main only), attached as artifact

**Trigger rules:** Jobs only run when files in their scope change (e.g., frontend tests don't run on backend-only changes).

---

## Releasing

We don't yet have automated semver tagging. For now:

1. Bump version in `backend/internal/platform/config/version.go` (planned).
2. Update `CHANGELOG.md` (planned).
3. Tag with `git tag vYYYY.MM.DD-<short-sha>`.
4. The `release:binaries` CI job will produce a `avex-vYYYY.MM.DD-<sha>.tar.gz` artifact.

---

## Issue Reporting

- **Bugs:** Use the Bug Report template. Include: Go version, browser, reproduction steps, expected vs actual behavior, logs.
- **Features:** Use the Feature Request template. Describe the user story and acceptance criteria.
- **Security:** Do NOT open a public issue. Email `security@zyyat-group.example` (replace with real address) with details.

---

## Pull Request Checklist

Before requesting review, ensure:

- [ ] Branch name follows `<type>/<scope>-<desc>` convention
- [ ] Commit messages follow Conventional Commits
- [ ] `gofmt -w .` and `go vet ./...` are clean (backend)
- [ ] `pnpm typecheck` passes in affected app(s) (frontend)
- [ ] New code has tests; existing tests still pass
- [ ] No new `console.log` / `fmt.Println` in production code
- [ ] No secrets, tokens, or credentials in the diff
- [ ] `CHANGELOG.md` updated if user-facing (planned)
- [ ] Documentation updated if API or behavior changed
- [ ] MR description explains the "why", not just the "what"

---

Thank you for helping make AVEX better! 🚀
