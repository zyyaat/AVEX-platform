// Package system is the composition root for the system module.
//
// The system module is different from other modules:
//   - No database schema (no migration needed — it reads from other modules)
//   - No outbox/inbox (it doesn't publish events)
//   - No domain tests (the types are simple value objects)
//   - Health checkers are adapters that wrap pgxpool + redis client
package system

import (
        "context"
        "net/http"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"
        "github.com/redis/go-redis/v9"

        "avex-backend/internal/modules/system/domain"
        "avex-backend/internal/modules/system/port"
        "avex-backend/internal/modules/system/service"
        httptransport "avex-backend/internal/modules/system/transport/http"
        "avex-backend/internal/platform/config"
)

type Module struct {
        svc port.ServicePort
}

func New(cfg *config.Config, pool *pgxpool.Pool, redisClient *redis.Client, maintenance port.MaintenanceChecker) *Module {
        info := domain.NewSystemInfo(
                cfg.App.Name,
                "1.0.0",              // version
                "",                   // build date (set at build time via -ldflags)
                "",                   // git commit (set at build time)
                "",                   // go version (filled at runtime)
                string(cfg.App.Env),  // environment
                time.Now().UTC(),     // started at
        )

        // Build health checkers
        checkers := []port.HealthChecker{
                &dbHealthChecker{pool: pool},
                &redisHealthChecker{client: redisClient},
        }

        // Build module list
        modules := []domain.ModuleInfo{
                domain.NewModuleInfo("identity", "User/driver/merchant authentication", 1, true),
                domain.NewModuleInfo("orders", "Order lifecycle management", 1, true),
                domain.NewModuleInfo("catalog", "Restaurants + menu items", 1, true),
                domain.NewModuleInfo("financial", "Wallets + pricing + promotions", 1, true),
                domain.NewModuleInfo("dispatch", "Driver matching + offers", 1, true),
                domain.NewModuleInfo("realtime", "WebSocket hub", 1, true),
                domain.NewModuleInfo("notifications", "Push + SMS + email", 1, true),
                domain.NewModuleInfo("support", "Tickets + messages", 1, true),
                domain.NewModuleInfo("permissions", "RBAC", 1, true),
                domain.NewModuleInfo("settings", "Versioned config + feature flags", 1, true),
                domain.NewModuleInfo("audit", "Immutable audit log", 1, true),
                domain.NewModuleInfo("system", "Health checks + system info", 1, true),
        }

        svc := service.New(info, checkers, modules, maintenance)
        return &Module{svc: svc}
}

func (m *Module) Service() port.ServicePort { return m.svc }

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
        httptransport.RegisterRoutes(mux, m.svc)
}

func (m *Module) Close() {}

// ===== DB Health Checker =====

type dbHealthChecker struct {
        pool *pgxpool.Pool
}

func (c *dbHealthChecker) Name() string { return "database" }

func (c *dbHealthChecker) CheckHealth(ctx context.Context) domain.ComponentHealth {
        start := time.Now()
        if c.pool == nil {
                return domain.NewComponentHealth("database", domain.HealthStatusUnhealthy, "pool is nil", 0, nil)
        }
        // Ping with a 3-second timeout
        pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
        defer cancel()
        var result int
        err := c.pool.QueryRow(pingCtx, "SELECT 1").Scan(&result)
        latency := time.Since(start)
        if err != nil {
                return domain.NewComponentHealth("database", domain.HealthStatusUnhealthy, err.Error(), latency, nil)
        }
        return domain.NewComponentHealth("database", domain.HealthStatusHealthy, "connected", latency, map[string]any{
                "result": result,
        })
}

// ===== Redis Health Checker =====

type redisHealthChecker struct {
        client *redis.Client
}

func (c *redisHealthChecker) Name() string { return "redis" }

func (c *redisHealthChecker) CheckHealth(ctx context.Context) domain.ComponentHealth {
        start := time.Now()
        if c.client == nil {
                return domain.NewComponentHealth("redis", domain.HealthStatusUnhealthy, "client is nil", 0, nil)
        }
        pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
        defer cancel()
        pong, err := c.client.Ping(pingCtx).Result()
        latency := time.Since(start)
        if err != nil {
                return domain.NewComponentHealth("redis", domain.HealthStatusUnhealthy, err.Error(), latency, nil)
        }
        return domain.NewComponentHealth("redis", domain.HealthStatusHealthy, "connected", latency, map[string]any{
                "response": pong,
        })
}
