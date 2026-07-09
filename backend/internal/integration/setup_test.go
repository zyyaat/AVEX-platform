//go:build integration

// Package integration_test contains cross-module integration tests that verify
// the full lifecycle flows of the AVEX delivery platform.
//
// These tests require a running PostgreSQL database. Set DATABASE_URL to
// enable them:
//
//      DATABASE_URL=postgres://user:pass@localhost:5432/avex_test?sslmode=disable
//      go test -tags=integration ./internal/integration/...
//
// If DATABASE_URL is not set, all tests are skipped.
package integration_test

import (
        "context"
        "embed"
        "fmt"
        "log/slog"
        "os"
        "strings"
        "testing"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/catalog"
        "avex-backend/internal/modules/financial"
        "avex-backend/internal/modules/identity"
        idp "avex-backend/internal/modules/identity/port"
        "avex-backend/internal/modules/orders"
        "avex-backend/internal/modules/permissions"
        "avex-backend/internal/modules/settings"
        "avex-backend/internal/modules/support"
        "avex-backend/internal/platform/config"
        "avex-backend/internal/platform/database"
        migrations "avex-backend/migrations"
)

// ===== Test Infrastructure =====

var (
        testPool       *pgxpool.Pool
        identityMod    *identity.Module
        ordersMod      *orders.Module
        catalogMod     *catalog.Module
        financialMod   *financial.Module
        permissionsMod *permissions.Module
        settingsMod    *settings.Module
        supportMod     *support.Module
        mockJWT        idp.JWTIssuer
)

func TestMain(m *testing.M) {
        dsn := os.Getenv("DATABASE_URL")
        if dsn == "" || !strings.HasPrefix(dsn, "postgres") {
                fmt.Fprintln(os.Stderr, "DATABASE_URL not set or not PostgreSQL — skipping integration tests")
                os.Exit(0)
        }
        ctx := context.Background()

        // Run all migrations in dependency order
        migrationSteps := []struct {
                fs   embed.FS
                name string
        }{
                {migrations.IdentityMigrations, "identity"},
                {migrations.OrdersMigrations, "orders"},
                {migrations.CatalogMigrations, "catalog"},
                {migrations.FinancialMigrations, "financial"},
                {migrations.PermissionsMigrations, "permissions"},
                {migrations.SettingsMigrations, "settings"},
                {migrations.SupportMigrations, "support"},
        }
        for _, step := range migrationSteps {
                if err := database.RunUp(ctx, dsn, step.fs, step.name, step.name); err != nil {
                        fmt.Fprintf(os.Stderr, "migration %s failed: %v\n", step.name, err)
                        os.Exit(1)
                }
        }

        // Create pool
        cfg, _ := pgxpool.ParseConfig(dsn)
        cfg.MaxConns = 10
        pool, err := pgxpool.NewWithConfig(ctx, cfg)
        if err != nil {
                fmt.Fprintf(os.Stderr, "pool: %v\n", err)
                os.Exit(1)
        }
        testPool = pool

        // Create app config
        appCfg := &config.Config{
                App:    config.AppConfig{Env: config.EnvDevelopment, Name: "avex-test", Port: "8080"},
                JWT:    config.JWTConfig{Secret: "test-secret-at-least-32-characters-long!!", Issuer: "avex-test", AccessTTL: 24 * time.Hour},
                Bcrypt: config.BcryptConfig{Cost: 4},
        }
        mockJWT = &mockJWTIssuer{}
        log := slog.Default()

        // Wire all modules
        identityMod = identity.New(appCfg, pool, log)
        ordersMod = orders.New(appCfg, pool, log, identityMod.JWTIssuer())
        catalogMod = catalog.New(appCfg, pool, log)
        financialMod = financial.New(appCfg, pool, log)
        permissionsMod = permissions.New(appCfg, pool, log)
        settingsMod = settings.New(appCfg, pool, log)
        supportMod = support.New(appCfg, pool, log)

        code := m.Run()
        pool.Close()
        os.Exit(code)
}

// ===== Mock JWT Issuer =====

type mockJWTIssuer struct{}

func (m *mockJWTIssuer) Issue(ctx context.Context, params idp.IssueJWTParams) (string, error) {
        return "mock-token-" + params.Subject, nil
}

func (m *mockJWTIssuer) Verify(ctx context.Context, token string) (*idp.JWTClaims, error) {
        return &idp.JWTClaims{
                Subject:   "test-user",
                Role:      "user",
                SessionID: "test-session",
                ExpiresAt: time.Now().Add(24 * time.Hour),
        }, nil
}

// ===== Cleanup Helper =====

func cleanupAll(t *testing.T) {
        t.Helper()
        ctx := context.Background()
        tables := []string{
                "support.outbox", "support.inbox", "support.ticket_attachments",
                "support.ticket_messages", "support.tickets",
                "settings.outbox", "settings.inbox", "settings.feature_flags",
                "settings.setting_revisions", "settings.settings",
                "permissions.outbox", "permissions.inbox", "permissions.user_roles",
                "permissions.role_permissions", "permissions.permissions",
                // Don't delete permissions.roles (seeded system roles)
                "financial.outbox", "financial.inbox", "financial.promotion_redemptions",
                "financial.promotions", "financial.surge_zones", "financial.pricing_rules",
                "financial.taxes", "financial.transactions",
                // Don't delete financial.wallets (needed for wallet tests)
                "catalog.outbox", "catalog.inbox", "catalog.store_hours",
                "catalog.menu_items", "catalog.categories", "catalog.restaurants",
                "orders.outbox", "orders.inbox", "orders.order_assignments",
                "orders.order_status_history", "orders.order_items", "orders.orders",
                "identity.outbox", "identity.inbox", "identity.sessions",
                "identity.drivers", "identity.merchants", "identity.users",
        }
        for _, table := range tables {
                _, _ = testPool.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", table))
        }
}

// suppress unused import warnings
var _ = catalogMod
var _ = financialMod
var _ = permissionsMod
var _ = settingsMod
var _ = supportMod
