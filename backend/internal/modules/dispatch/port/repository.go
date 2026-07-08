// Package port repository: persistence interfaces for the dispatch module.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/dispatch/domain"
)

// ===== DriverRepository =====

type DriverRepository interface {
	// Create inserts a new driver.
	Create(ctx context.Context, exec Executor, driver domain.Driver) error

	// GetByID retrieves a driver by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Driver, error)

	// GetByUserID retrieves a driver by their identity user ID.
	GetByUserID(ctx context.Context, exec Executor, userID string) (*domain.Driver, error)

	// Update saves all fields of an existing driver with optimistic locking.
	Update(ctx context.Context, exec Executor, driver domain.Driver) error

	// ListOnlineByZone retrieves all online drivers serving the given zone.
	// If zoneID is empty, returns all online drivers.
	ListOnlineByZone(ctx context.Context, exec Executor, zoneID string) ([]domain.Driver, error)

	// ListAll retrieves all drivers (admin view) with pagination.
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Driver], error)
}

// ===== DriverLocationRepository =====

// NearbyDriver is the result of a "find nearest drivers" query.
type NearbyDriver struct {
	DriverID      string
	Lat           float64
	Lng           float64
	DistanceM     int    // great-circle distance from the query point
	Bearing       float64
	Speed         float64
	LocationAge   time.Duration // how stale the location is
	CapturedAt    time.Time
}

type DriverLocationRepository interface {
	// Upsert inserts or updates the current location for a driver.
	// Each driver has at most one row in the locations table.
	Upsert(ctx context.Context, exec Executor, loc domain.DriverLocation) error

	// GetByDriver retrieves the current location for a driver.
	GetByDriver(ctx context.Context, exec Executor, driverID string) (*domain.DriverLocation, error)

	// FindNearestDrivers returns up to `limit` drivers within `radiusM` meters
	// of the given (lat, lng), ordered by distance ascending.
	// Only drivers with locations fresher than `maxAge` are considered.
	FindNearestDrivers(ctx context.Context, exec Executor, lat, lng float64, radiusM int, maxAge time.Duration, limit int) ([]NearbyDriver, error)

	// DeleteByDriver removes the location for a driver (used when driver goes offline).
	DeleteByDriver(ctx context.Context, exec Executor, driverID string) error
}

// ===== DispatchOfferRepository =====

type DispatchOfferRepository interface {
	// Create inserts a new dispatch offer.
	Create(ctx context.Context, exec Executor, offer domain.DispatchOffer) error

	// GetByID retrieves an offer by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.DispatchOffer, error)

	// GetActiveOfferForOrder retrieves the pending offer for an order, if any.
	// Returns ErrOfferNotFound if no pending offer exists.
	GetActiveOfferForOrder(ctx context.Context, exec Executor, orderID string) (*domain.DispatchOffer, error)

	// Update saves all fields of an existing offer.
	Update(ctx context.Context, exec Executor, offer domain.DispatchOffer) error

	// ListByDriver retrieves offers for a driver with pagination.
	ListByDriver(ctx context.Context, exec Executor, driverID string, page PageQuery) (Page[domain.DispatchOffer], error)

	// ListByOrder retrieves all offers for an order (across attempts).
	ListByOrder(ctx context.Context, exec Executor, orderID string) ([]domain.DispatchOffer, error)

	// CountAttemptsForOrder returns the number of offers created for an order.
	// Used to enforce max retry limit.
	CountAttemptsForOrder(ctx context.Context, exec Executor, orderID string) (int, error)
}

// ===== OutboxRepository =====

type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}

// ===== Aggregate =====

type RepositorySet struct {
	Drivers    DriverRepository
	Locations  DriverLocationRepository
	Offers     DispatchOfferRepository
	Outbox     OutboxRepository
}
