// Package service is the dispatch module's service layer.
package service

import (
	"context"
	"time"

	"avex-backend/internal/modules/dispatch/port"
)

// Config holds service-layer configuration.
type Config struct {
	// OfferTTL is how long a driver has to respond to an offer.
	OfferTTL time.Duration
	// LocationStaleTTL is how old a driver location can be before it's
	// considered stale (driver treated as offline for matching).
	LocationStaleTTL time.Duration
	// DefaultSearchRadiusM is the default search radius for nearest-driver queries.
	DefaultSearchRadiusM int
	// MaxAttempts is the maximum number of dispatch attempts per order
	// before giving up.
	MaxAttempts int
	// DefaultCurrency is the currency used for offer fares.
	DefaultCurrency string
}

// Service implements port.ServicePort.
type Service struct {
	deps port.Deps
	pool port.Executor
	cfg  Config
}

var _ port.ServicePort = (*Service)(nil)

// New creates a new dispatch Service.
func New(deps port.Deps, pool port.Executor, cfg Config) *Service {
	// Apply defaults.
	if cfg.OfferTTL <= 0 {
		cfg.OfferTTL = 15 * time.Second
	}
	if cfg.LocationStaleTTL <= 0 {
		cfg.LocationStaleTTL = 60 * time.Second
	}
	if cfg.DefaultSearchRadiusM <= 0 {
		cfg.DefaultSearchRadiusM = 5000 // 5km
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = "EGP"
	}
	return &Service{deps: deps, pool: pool, cfg: cfg}
}

// eventContext builds an EventContext from the request context + actor.
func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
	return port.EventContext{
		Actor: actor,
		Metadata: port.EventMetadata{
			OccurredAt: s.deps.Clock.Now(),
		},
	}
}
