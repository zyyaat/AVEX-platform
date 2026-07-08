// Package identity benchmarks: performance benchmarks for hot service methods.
//
// These benchmarks run against real PostgreSQL + Redis to measure end-to-end
// performance including DB I/O, bcrypt hashing, JWT issuance, and event publishing.
//
// Run with:
//   DATABASE_URL=postgres://avex@localhost:5432/avex_test?sslmode=disable \
//   REDIS_URL=redis://localhost:6379/0 \
//   go test -tags=integration -bench=. -benchmem -benchtime=3s \
//   ./internal/modules/identity/...
//
//go:build integration

package identity_test

import (
        "context"
        "fmt"
        "log/slog"
        "testing"
        "time"

        "github.com/google/uuid"

        "avex-backend/internal/modules/identity"
        "avex-backend/internal/modules/identity/port"
)

// benchModule holds a pre-wired identity module for benchmarks.
// It is created once per benchmark run and reused across iterations.
var benchModule *identity.Module

// benchCounter ensures each benchmark iteration uses a unique phone number
// to avoid unique constraint violations.
var benchCounter int

// setupBench creates a fresh module + cleans tables before each benchmark.
func setupBench(b *testing.B) {
        b.Helper()
        ctx := context.Background()

        // Clean tables before benchmark.
        _, err := integDBPool.Exec(ctx, `
                TRUNCATE identity.users, identity.drivers, identity.merchants,
                         identity.support_agents, identity.sessions,
                         identity.password_resets, identity.outbox, identity.inbox
                CASCADE
        `)
        if err != nil {
                b.Fatalf("cleanup: %v", err)
        }

        // Create a fresh module (avoids state from previous benchmarks).
        if benchModule != nil {
                benchModule.Close()
        }
        benchModule = identity.New(integCfg, integDBPool, slog.Default())
        benchCounter = 0
}

// nextPhone returns a unique Egyptian phone number for each call.
// Uses the 011 prefix (valid Egyptian mobile) with a counter.
func nextPhone() string {
        benchCounter++
        return fmt.Sprintf("011%08d", benchCounter)
}

// ===== Benchmarks =====

// BenchmarkRegisterUser measures the full register flow:
// validate → hash password → create user → create session → issue JWT → publish event.
func BenchmarkRegisterUser(b *testing.B) {
        setupBench(b)
        ctx := context.Background()
        svc := benchModule.Service()

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
                _, err := svc.RegisterUser(ctx, port.RegisterUserInput{
                        Name:     "Bench User",
                        Phone:    nextPhone(),
                        Password: "password123",
                        Email:    "",
                })
                if err != nil {
                        b.Fatalf("RegisterUser: %v", err)
                }
        }
}

// BenchmarkLoginUser measures the login flow:
// fetch user by phone → compare password → create session → issue JWT → publish event.
func BenchmarkLoginUser(b *testing.B) {
        setupBench(b)
        ctx := context.Background()
        svc := benchModule.Service()

        // Pre-register a user to login with.
        phone := nextPhone()
        _, err := svc.RegisterUser(ctx, port.RegisterUserInput{
                Name: "Bench Login User", Phone: phone, Password: "password123",
        })
        if err != nil {
                b.Fatalf("pre-register: %v", err)
        }

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
                _, err := svc.LoginUser(ctx, port.LoginInput{
                        Phone:    phone,
                        Password: "password123",
                        IP:       "10.0.0.1",
                        Agent:    "bench",
                })
                if err != nil {
                        b.Fatalf("LoginUser: %v", err)
                }
        }
}

// BenchmarkGetUser measures a simple read: fetch user by ID.
// This is the hottest path (called on every authenticated request via /users/me).
func BenchmarkGetUser(b *testing.B) {
        setupBench(b)
        ctx := context.Background()
        svc := benchModule.Service()

        // Pre-register a user to fetch.
        result, err := svc.RegisterUser(ctx, port.RegisterUserInput{
                Name: "Bench Get User", Phone: nextPhone(), Password: "password123",
        })
        if err != nil {
                b.Fatalf("pre-register: %v", err)
        }
        userID := result.User.ID

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
                _, err := svc.GetUser(ctx, userID)
                if err != nil {
                        b.Fatalf("GetUser: %v", err)
                }
        }
}

// BenchmarkGetDriverProfile measures driver profile retrieval.
func BenchmarkGetDriverProfile(b *testing.B) {
        setupBench(b)
        ctx := context.Background()
        svc := benchModule.Service()

        // Create a driver via the service — need to go through domain directly
        // since RegisterDriver is not on the ServicePort yet.
        // Use the module's internal service via a hash.
        // For benchmarking, we seed a driver directly via domain + repo.
        // However, we don't have direct repo access from this package.
        // Instead, we use a workaround: register via the testutil mock approach
        // is not applicable here (we need real DB).
        //
        // Alternative: use the service's HashPassword + direct DB insert.
        // But the simplest approach: skip if we can't seed a driver.
        // For now, we'll use the LoginDriver flow which requires a pre-existing driver.
        // Since we can't create a driver via ServicePort, we'll insert one via SQL.

        hash, err := svc.HashPassword(ctx, "password123")
        if err != nil {
                b.Fatalf("HashPassword: %v", err)
        }

        driverID := uuid.NewString()
        benchCounter++
        _, err = integDBPool.Exec(ctx, `
                INSERT INTO identity.drivers (id, name, phone, password_hash, vehicle_type,
                    license_number, national_id, status, is_online, is_active, is_verified,
                    must_change_password, locale, timezone, created_at, updated_at)
                VALUES ($1, 'Bench Driver', $2, $3, 'motorcycle', $4, $5, 'offline',
                    false, true, true, false, 'ar', 'Africa/Cairo', NOW(), NOW())
        `, driverID, nextPhone(), hash, "LIC-BENCH", "NID-BENCH")
        if err != nil {
                b.Fatalf("seed driver: %v", err)
        }

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
                _, err := svc.GetDriverProfile(ctx, driverID)
                if err != nil {
                        b.Fatalf("GetDriverProfile: %v", err)
                }
        }
}

// BenchmarkVerifyUserExists measures the cross-module verification call.
// This will be called by orders/payments modules.
func BenchmarkVerifyUserExists(b *testing.B) {
        setupBench(b)
        ctx := context.Background()
        svc := benchModule.Service()

        result, err := svc.RegisterUser(ctx, port.RegisterUserInput{
                Name: "Bench Verify", Phone: nextPhone(), Password: "password123",
        })
        if err != nil {
                b.Fatalf("pre-register: %v", err)
        }
        userID := result.User.ID

        b.ResetTimer()
        b.ReportAllocs()

        for i := 0; i < b.N; i++ {
                _, err := svc.VerifyUserExists(ctx, userID)
                if err != nil {
                        b.Fatalf("VerifyUserExists: %v", err)
                }
        }
}

// suppress unused import
var _ = time.Second
