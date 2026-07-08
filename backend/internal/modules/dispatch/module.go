// Package dispatch is the composition root for the dispatch module.
package dispatch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/dispatch/events"
	"avex-backend/internal/modules/dispatch/mapbox"
	"avex-backend/internal/modules/dispatch/port"
	"avex-backend/internal/modules/dispatch/repository/postgres"
	"avex-backend/internal/modules/dispatch/service"
	httptransport "avex-backend/internal/modules/dispatch/transport/http"
	idp "avex-backend/internal/modules/identity/port"
	"avex-backend/internal/platform/config"
)

// Module is the wired dispatch module.
type Module struct {
	svc    port.ServicePort
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// New wires all dispatch dependencies and returns a ready-to-use Module.
// mapboxToken is the Mapbox API access token (empty = disable distance matrix).
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger, mapboxToken string) *Module {
	repos := postgres.NewRepositories()
	repoSet := repos.RepositorySet()

	// Event Publisher
	eventPublisher := events.NewEventPublisher(repoSet, &uuidIDGen{})

	// Mapbox DistanceMatrixProvider (nil if no token — falls back to Haversine)
	var distanceProvider port.DistanceMatrixProvider
	if mapboxToken != "" {
		distanceProvider = mapbox.New(mapboxToken)
		logger.Info("mapbox distance matrix enabled")
	} else {
		logger.Warn("mapbox distance matrix disabled — using Haversine fallback")
	}

	// TxRunner
	txRunner := &pgxTxRunner{pool: pool}

	// Service
	deps := port.Deps{
		Clock:                  &realClock{},
		IDGenerator:            &uuidIDGen{},
		EventPublisher:         eventPublisher,
		Logger:                 logger,
		TxRunner:               txRunner,
		Repos:                  repoSet,
		DistanceMatrixProvider: distanceProvider,
	}

	svc := service.New(deps, pool, service.Config{
		OfferTTL:              15 * time.Second,
		LocationStaleTTL:      60 * time.Second,
		DefaultSearchRadiusM:  5000,
		MaxAttempts:           5,
		DefaultCurrency:       "EGP",
	})

	return &Module{svc: svc, pool: pool, logger: logger}
}

// Service exposes the port.ServicePort for cross-module use.
func (m *Module) Service() port.ServicePort { return m.svc }

// RegisterRoutes wires the dispatch HTTP routes into the given mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux, jwtIssuer idp.JWTIssuer) {
	httptransport.RegisterRoutes(mux, m.svc, m.logger, jwtIssuer)
}

// Close releases resources held by the module.
func (m *Module) Close() {}

// ===== Adapters =====

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type uuidIDGen struct{}

func (*uuidIDGen) NewID() string { return newUUID() }

type pgxTxRunner struct{ pool *pgxpool.Pool }

func (r *pgxTxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, exec port.Executor) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// newUUID generates a UUIDv4 using crypto/rand.
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
