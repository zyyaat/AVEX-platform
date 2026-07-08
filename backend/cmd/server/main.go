// Package main is the HTTP API server entry point.
package main

import (
        "context"
        "errors"
        "fmt"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "avex-backend/internal/modules/catalog"
        "avex-backend/internal/modules/dispatch"
        "avex-backend/internal/modules/financial"
        "avex-backend/internal/modules/identity"
        httptransport "avex-backend/internal/modules/identity/transport/http"
        "avex-backend/internal/modules/orders"
        "avex-backend/internal/platform/config"
        "avex-backend/internal/platform/database"
        "avex-backend/internal/platform/logger"
        migrations "avex-backend/migrations"
)

func main() {
        ctx := context.Background()

        // 1. Load config.
        cfg, err := config.Load()
        if err != nil {
                fmt.Fprintf(os.Stderr, "❌ config: %v\n", err)
                os.Exit(1)
        }

        // 2. Init logger.
        log := logger.New(cfg)
        log.Info("starting server", "app", cfg.App.Name, "env", cfg.App.Env, "port", cfg.App.Port)

        // 3. Connect to database.
        dbPool, err := database.Connect(ctx, cfg.Database)
        if err != nil {
                log.Error("database connect failed", "error", err)
                os.Exit(1)
        }
        defer dbPool.Close()
        log.Info("database connected")

        // 4. Run migrations — each module has its own goose version table.
        if err := database.RunUp(ctx, cfg.Database.URL, migrations.IdentityMigrations, "identity", "identity"); err != nil {
                log.Error("identity migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("identity migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.OrdersMigrations, "orders", "orders"); err != nil {
                log.Error("orders migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("orders migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.CatalogMigrations, "catalog", "catalog"); err != nil {
                log.Error("catalog migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("catalog migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.FinancialMigrations, "financial", "financial"); err != nil {
                log.Error("financial migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("financial migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.DispatchMigrations, "dispatch", "dispatch"); err != nil {
                log.Error("dispatch migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("dispatch migrations complete")

        // 5. Wire modules.
        identityMod := identity.New(cfg, dbPool.Pool(), log)
        defer identityMod.Close()
        log.Info("identity module wired")

        // Orders module needs identity's JWT issuer for auth middleware.
        ordersMod := orders.New(cfg, dbPool.Pool(), log, identityMod.JWTIssuer())
        defer ordersMod.Close()
        log.Info("orders module wired")

        catalogMod := catalog.New(cfg, dbPool.Pool(), log)
        defer catalogMod.Close()
        log.Info("catalog module wired")

        financialMod := financial.New(cfg, dbPool.Pool(), log)
        defer financialMod.Close()
        log.Info("financial module wired")

        // Dispatch module needs the Mapbox token from env.
        mapboxToken := os.Getenv("MAPBOX_ACCESS_TOKEN")
        dispatchMod := dispatch.New(cfg, dbPool.Pool(), log, mapboxToken)
        defer dispatchMod.Close()
        log.Info("dispatch module wired")

        // 6. Setup HTTP server.
        mux := http.NewServeMux()
        identityMod.RegisterRoutes(mux, cfg)
        ordersMod.RegisterRoutes(mux)
        catalogMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        financialMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        dispatchMod.RegisterRoutes(mux, identityMod.JWTIssuer())

        handler := httptransport.RequestID(mux)
        handler = httptransport.Logging(log)(handler)
        handler = httptransport.Recovery(log)(handler)
        handler = httptransport.CORS(cfg.CORS.AllowedOrigins)(handler)

        server := &http.Server{
                Addr:         ":" + cfg.App.Port,
                Handler:      handler,
                ReadTimeout:  15 * time.Second,
                WriteTimeout: 15 * time.Second,
                IdleTimeout:  60 * time.Second,
        }

        // 7. Start server.
        go func() {
                log.Info("http server listening", "addr", server.Addr)
                if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
                        log.Error("server error", "error", err)
                        os.Exit(1)
                }
        }()

        // 8. Wait for shutdown signal.
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        sig := <-quit
        log.Info("shutdown signal received", "signal", sig)

        // 9. Graceful shutdown.
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := server.Shutdown(shutdownCtx); err != nil {
                log.Error("server shutdown error", "error", err)
        }
        log.Info("server stopped gracefully")
}
