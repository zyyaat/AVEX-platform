// Package main is the outbox publisher worker entry point.
//
// The worker processes outbox tables from ALL modules:
//   - identity.outbox
//   - orders.outbox
//
// Each module has its own outbox table. The worker runs a publisher for
// each module concurrently, polling the outbox and publishing events to Redis.
package main

import (
        "context"
        "fmt"
        "os"
        "os/signal"
        "sync"
        "syscall"
        "time"

        "avex-backend/internal/platform/bus"
        "avex-backend/internal/platform/config"
        "avex-backend/internal/platform/database"
        "avex-backend/internal/platform/logger"
        "avex-backend/internal/platform/outbox"
)

func main() {
        ctx := context.Background()

        // 1. Load config.
        cfg, err := config.Load()
        if err != nil {
                fmt.Fprintf(os.Stderr, "config: %v\n", err)
                os.Exit(1)
        }

        // 2. Init logger.
        log := logger.New(cfg)
        log.Info("starting outbox worker", "app", cfg.App.Name, "env", cfg.App.Env)

        // 3. Connect to database.
        dbPool, err := database.Connect(ctx, cfg.Database)
        if err != nil {
                log.Error("database connect failed", "error", err)
                os.Exit(1)
        }
        defer dbPool.Close()
        log.Info("database connected")

        // 4. Connect to Redis (event bus).
        redisBus, err := bus.NewRedisBus(ctx, cfg.Redis, log)
        if err != nil {
                log.Error("redis connect failed", "error", err)
                os.Exit(1)
        }
        defer redisBus.Close()
        log.Info("redis bus connected")

        // 5. Define outboxes for each module.
        outboxes := []struct {
                name  string
                table string
        }{
                {"identity", "identity.outbox"},
                {"orders", "orders.outbox"},
                {"financial", "financial.outbox"},
                {"dispatch", "dispatch.outbox"},
                {"notifications", "notifications.outbox"},
                {"support", "support.outbox"},
                {"permissions", "permissions.outbox"},
                {"settings", "settings.outbox"},
                {"audit", "audit.outbox"},
        }

        // 6. Start a publisher worker for each module.
        workerCtx, workerCancel := context.WithCancel(ctx)
        defer workerCancel()

        var wg sync.WaitGroup
        for _, ob := range outboxes {
                obInstance := outbox.NewPostgresOutbox(dbPool.Pool(), outbox.Config{
                        Table:          ob.table,
                        MaxRetries:     cfg.Outbox.MaxRetries,
                        RetryBaseDelay: cfg.Outbox.RetryBaseDelay,
                })

                worker := outbox.NewPublisherWorker(
                        obInstance,
                        redisBus,
                        cfg.Outbox.PollInterval,
                        cfg.Outbox.BatchSize,
                        log.With("module", ob.name),
                )

                wg.Add(1)
                go func(name string, w *outbox.PublisherWorker) {
                        defer wg.Done()
                        log.Info("starting outbox publisher", "module", name)
                        if err := w.Run(workerCtx); err != nil {
                                log.Error("outbox publisher exited with error", "module", name, "error", err)
                        }
                        log.Info("outbox publisher stopped", "module", name)
                }(ob.name, worker)
        }

        log.Info("all outbox workers running",
                "modules", len(outboxes),
                "poll_interval", cfg.Outbox.PollInterval,
                "batch_size", cfg.Outbox.BatchSize,
        )

        // 7. Wait for shutdown signal.
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        sig := <-quit
        log.Info("shutdown signal received", "signal", sig)

        // 8. Graceful shutdown.
        workerCancel()

        // Wait for all workers to finish (with timeout).
        done := make(chan struct{})
        go func() {
                wg.Wait()
                close(done)
        }()

        select {
        case <-done:
                log.Info("all workers stopped gracefully")
        case <-time.After(10 * time.Second):
                log.Warn("shutdown timeout — some workers may not have finished cleanly")
        }
}
