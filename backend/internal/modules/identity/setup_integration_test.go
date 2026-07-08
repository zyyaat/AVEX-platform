// Package identity integration tests: shared setup for all integration tests.
//
// This file contains the single TestMain for the identity_test package.
// It sets up PostgreSQL, Redis, the identity module, and the E2E HTTP server.
// Both outbox and E2E tests share this infrastructure.
//
//go:build integration

package identity_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/identity"
	httptransport "avex-backend/internal/modules/identity/transport/http"
	"avex-backend/internal/platform/bus"
	"avex-backend/internal/platform/config"
	"avex-backend/internal/platform/database"
	migrations "avex-backend/migrations"
)

var (
	integDBPool *pgxpool.Pool
	integRedis  *bus.RedisBus
	integCfg    *config.Config
	e2eServer   *httptest.Server
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL not set — skipping integration tests")
		os.Exit(0)
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	ctx := context.Background()

	// Run migrations.
	if err := database.RunUp(ctx, dsn, migrations.IdentityMigrations, "identity"); err != nil {
		fmt.Fprintf(os.Stderr, "migrations: %v\n", err)
		os.Exit(1)
	}

	// Create pool.
	poolCfg, _ := pgxpool.ParseConfig(dsn)
	poolCfg.MaxConns = 5
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	integDBPool = pool

	// Create Redis bus.
	redisCfg := config.RedisConfig{URL: redisURL, PoolSize: 5}
	rb, err := bus.NewRedisBus(ctx, redisCfg, slog.Default())
	if err != nil {
		fmt.Fprintf(os.Stderr, "redis: %v\n", err)
		os.Exit(1)
	}
	integRedis = rb

	// Build config.
	integCfg = &config.Config{
		App:      config.AppConfig{Env: config.EnvDevelopment, Name: "avex-test"},
		Database: config.DatabaseConfig{URL: dsn},
		Redis:    redisCfg,
		JWT:      config.JWTConfig{Secret: "test-secret-at-least-32-characters-long!!", Issuer: "avex-test", AccessTTL: 24 * time.Hour},
		Bcrypt:   config.BcryptConfig{Cost: 4},
		Outbox:   config.OutboxConfig{PollInterval: 100 * time.Millisecond, BatchSize: 10, MaxRetries: 3, RetryBaseDelay: 1 * time.Second},
		CORS:     config.CORSConfig{AllowedOrigins: []string{"*"}},
	}

	// Set up E2E HTTP server.
	mod := identity.New(integCfg, integDBPool, slog.Default())
	mux := http.NewServeMux()
	mod.RegisterRoutes(mux, integCfg)
	handler := httptransport.RequestID(mux)
	handler = httptransport.Logging(slog.Default())(handler)
	handler = httptransport.Recovery(slog.Default())(handler)
	handler = httptransport.CORS(integCfg.CORS.AllowedOrigins)(handler)
	e2eServer = httptest.NewServer(handler)

	code := m.Run()

	e2eServer.Close()
	pool.Close()
	rb.Close()
	os.Exit(code)
}

func cleanupIntegTables(t *testing.T) {
	t.Helper()
	_, err := integDBPool.Exec(context.Background(), `
		TRUNCATE identity.users, identity.drivers, identity.merchants,
		         identity.support_agents, identity.sessions,
		         identity.password_resets, identity.outbox, identity.inbox
		CASCADE
	`)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}
