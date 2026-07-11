# AVEX Delivery Platform

Full-stack delivery platform with Go backend and 5 React frontend apps.

## Architecture

### Backend (Go)
- 12 modules: Identity, Orders, Catalog, Financial, Dispatch, Realtime, Notifications, Support, Permissions, Settings, Audit, System, Localization
- 14 implementation phases completed
- 707 tests passing
- PostgreSQL + Redis + WebSocket

### Frontend (5 React Apps)
| App | Path | Purpose |
|-----|------|---------|
| Driver | `artifacts/driver` | Driver mobile app — login, map, location tracking, order acceptance |
| Admin | `artifacts/admin` | Admin dashboard — users, orders, financials, settings |
| Customer | `artifacts/customer` | Customer ordering app |
| Merchant | `artifacts/merchant` | Merchant catalog & order management |
| Support | `artifacts/support` | Support ticket system |

All apps: Vite + TypeScript + Tailwind + shadcn/ui

## Quick Start

### Prerequisites
- Go 1.22+
- Node 20+
- PostgreSQL 16
- Redis 7
- pnpm

### Backend
```bash
cd backend
cp .env.example .env  # Edit with your credentials
go run ./cmd/server
```

### Frontend
```bash
pnpm install
pnpm --filter driver dev
pnpm --filter admin dev
pnpm --filter customer dev
pnpm --filter merchant dev
pnpm --filter support dev
```

## Environment Variables

Create `.env` files in each app directory:

### Backend (`backend/.env`)
```
DATABASE_URL=postgres://user:pass@localhost:5432/avex_dev?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=your-secret-key-at-least-32-chars
JWT_ISSUER=avex
MAPBOX_ACCESS_TOKEN=your-mapbox-public-token
```

### Frontend (`artifacts/driver/.env`)
```
VITE_MAPBOX_TOKEN=your-mapbox-public-token
VITE_API_BASE=/api/v1
```

## API

Base URL: `/api/v1`
Auth: Bearer JWT token

## License

Proprietary — All rights reserved.
