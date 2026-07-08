// Package postgres implements the dispatch module's repository interfaces
// using pgx/v5 against a PostgreSQL database.
//
// Schema: all tables live in the PostgreSQL schema "dispatch".
package postgres

import (
	"avex-backend/internal/modules/dispatch/port"
	"avex-backend/internal/platform/database"
)

// Repositories is the concrete implementation of port.RepositorySet.
type Repositories struct {
	drivers   *DriverRepository
	locations *DriverLocationRepository
	offers    *DispatchOfferRepository
	outbox    *OutboxRepository
}

// NewRepositories constructs a Repositories.
func NewRepositories() *Repositories {
	return &Repositories{
		drivers:   &DriverRepository{},
		locations: &DriverLocationRepository{},
		offers:    &DispatchOfferRepository{},
		outbox:    &OutboxRepository{},
	}
}

// RepositorySet returns a port.RepositorySet backed by this Repositories.
func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{
		Drivers:   r.drivers,
		Locations: r.locations,
		Offers:    r.offers,
		Outbox:    r.outbox,
	}
}

// toDBTX converts a port.Executor into a database.DBTX.
func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok {
		panic("postgres: port.Executor does not satisfy database.DBTX — check composition root wiring")
	}
	return dbtx
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// nilIfEmptyStr returns nil for empty strings (for nullable columns).
func nilIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
