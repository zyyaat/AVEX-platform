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

        "avex-backend/internal/modules/audit"
        auditjobs "avex-backend/internal/modules/audit/jobs"
        "avex-backend/internal/modules/catalog"
        "avex-backend/internal/modules/dispatch"
        "avex-backend/internal/modules/financial"
        "avex-backend/internal/modules/identity"
        httptransport "avex-backend/internal/modules/identity/transport/http"
        "avex-backend/internal/modules/notifications"
        notifjobs "avex-backend/internal/modules/notifications/jobs"
        "avex-backend/internal/modules/orders"
        "avex-backend/internal/modules/permissions"
        "avex-backend/internal/modules/realtime"
        realtimejobs "avex-backend/internal/modules/realtime/jobs"
        "avex-backend/internal/modules/settings"
        "avex-backend/internal/modules/support"
        "avex-backend/internal/platform/bus"
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

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.RealtimeMigrations, "realtime", "realtime"); err != nil {
                log.Error("realtime migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("realtime migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.NotificationsMigrations, "notifications", "notifications"); err != nil {
                log.Error("notifications migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("notifications migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.SupportMigrations, "support", "support"); err != nil {
                log.Error("support migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("support migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.PermissionsMigrations, "permissions", "permissions"); err != nil {
                log.Error("permissions migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("permissions migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.SettingsMigrations, "settings", "settings"); err != nil {
                log.Error("settings migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("settings migrations complete")

        if err := database.RunUp(ctx, cfg.Database.URL, migrations.AuditMigrations, "audit", "audit"); err != nil {
                log.Error("audit migrations failed", "error", err)
                os.Exit(1)
        }
        log.Info("audit migrations complete")

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

        // Realtime module (WebSocket hub).
        realtimeMod := realtime.New(cfg, dbPool.Pool(), log)
        defer realtimeMod.Close()
        log.Info("realtime module wired")

        // Notifications module (push/SMS/email).
        notificationsMod := notifications.New(cfg, dbPool.Pool(), log)
        defer notificationsMod.Close()
        log.Info("notifications module wired")

        // Support module (tickets + messages).
        supportMod := support.New(cfg, dbPool.Pool(), log)
        defer supportMod.Close()
        log.Info("support module wired")

        // Permissions module (RBAC).
        permissionsMod := permissions.New(cfg, dbPool.Pool(), log)
        defer permissionsMod.Close()
        log.Info("permissions module wired")

        // Settings module (versioned config + feature flags).
        settingsMod := settings.New(cfg, dbPool.Pool(), log)
        defer settingsMod.Close()
        log.Info("settings module wired")

        // Audit module (immutable audit log).
        auditMod := audit.New(cfg, dbPool.Pool(), log)
        defer auditMod.Close()
        log.Info("audit module wired")

        // 5b. Connect to Redis bus for the realtime subscriber (consumes events
        // from orders/dispatch/financial and broadcasts to WebSocket clients).
        redisBus, err := bus.NewRedisBus(ctx, cfg.Redis, log)
        if err != nil {
                log.Error("redis bus connect failed (realtime subscriber)", "error", err)
                os.Exit(1)
        }
        defer redisBus.Close()
        log.Info("redis bus connected (realtime + notifications subscriber)")

        // 5c. Start the realtime event subscriber.
        realtimeInbox := realtimeMod.NewInbox()
        realtimeSub := realtimejobs.NewSubscriber(realtimeMod.Service(), redisBus, realtimeInbox, log)
        if err := realtimeSub.Start(ctx); err != nil {
                log.Error("realtime subscriber start failed", "error", err)
                os.Exit(1)
        }
        log.Info("realtime subscriber started")

        // 5d. Start the notifications event subscriber.
        notifInbox := notificationsMod.NewInbox()
        notifSub := notifjobs.NewSubscriber(notificationsMod.Service(), redisBus, notifInbox, log)
        if err := notifSub.Start(ctx); err != nil {
                log.Error("notifications subscriber start failed", "error", err)
                os.Exit(1)
        }
        log.Info("notifications subscriber started")

        // 5e. Start the audit event subscriber (listens to ALL module events).
        auditInbox := auditMod.NewInbox()
        auditSub := auditjobs.NewSubscriber(auditMod.Service(), redisBus, auditInbox, log)
        if err := auditSub.Start(ctx); err != nil {
                log.Error("audit subscriber start failed", "error", err)
                os.Exit(1)
        }
        log.Info("audit subscriber started")

        // 6. Setup HTTP server.
        mux := http.NewServeMux()
        identityMod.RegisterRoutes(mux, cfg)
        ordersMod.RegisterRoutes(mux)
        catalogMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        financialMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        dispatchMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        realtimeMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        notificationsMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        supportMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        permissionsMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        settingsMod.RegisterRoutes(mux, identityMod.JWTIssuer())
        auditMod.RegisterRoutes(mux, identityMod.JWTIssuer())

        handler := httptransport.RequestID(mux)
        handler = httptransport.Logging(log)(handler)
        handler = httptransport.Recovery(log)(handler)
        handler = httptransport.CORS(cfg.CORS.AllowedOrigins)(handler)

        server := &http.Server{
                Addr:         ":" + cfg.App.Port,
                Handler:      handler,
                ReadTimeout:  15 * time.Second,
                WriteTimeout: 0, // 0 = no timeout (WebSocket connections are long-lived)
                IdleTimeout:  120 * time.Second,
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
