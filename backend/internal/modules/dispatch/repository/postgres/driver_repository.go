// Package postgres driver_repository: DriverRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"avex-backend/internal/modules/dispatch/domain"
	"avex-backend/internal/modules/dispatch/port"
)

// DriverRepository implements port.DriverRepository.
type DriverRepository struct{}

var _ port.DriverRepository = (*DriverRepository)(nil)

// Create inserts a new driver.
func (r *DriverRepository) Create(ctx context.Context, exec port.Executor, driver domain.Driver) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO dispatch.drivers (
			id, user_id, vehicle_type, license_plate,
			status, rating, rating_count, acceptance_rate, completion_rate, total_deliveries,
			zone_ids, current_order_id, go_online_at, go_offline_at, suspended_reason,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18
		)
	`,
		driver.ID(),
		driver.UserID(),
		string(driver.VehicleType()),
		driver.LicensePlate(),
		string(driver.Status()),
		driver.Rating(),
		driver.RatingCount(),
		driver.AcceptanceRate(),
		driver.CompletionRate(),
		driver.TotalDeliveries(),
		driver.ZoneIDs(),
		nilIfEmptyStr(driver.CurrentOrderID()),
		driver.GoOnlineAt(),
		driver.GoOfflineAt(),
		nilIfEmptyStr(driver.SuspendedReason()),
		driver.CreatedAt(),
		driver.UpdatedAt(),
		driver.Version(),
	)
	if err != nil {
		return mapDriverWriteError(err)
	}
	return nil
}

// GetByID retrieves a driver by UUID.
func (r *DriverRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Driver, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+driverColumns+` FROM dispatch.drivers WHERE id = $1`, id)
	d, err := scanDriver(row)
	if err != nil {
		return nil, mapDriverReadError(err)
	}
	return &d, nil
}

// GetByUserID retrieves a driver by their identity user ID.
func (r *DriverRepository) GetByUserID(ctx context.Context, exec port.Executor, userID string) (*domain.Driver, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+driverColumns+` FROM dispatch.drivers WHERE user_id = $1`, userID)
	d, err := scanDriver(row)
	if err != nil {
		return nil, mapDriverReadError(err)
	}
	return &d, nil
}

// Update saves all fields with optimistic locking.
func (r *DriverRepository) Update(ctx context.Context, exec port.Executor, driver domain.Driver) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE dispatch.drivers SET
			vehicle_type = $2,
			license_plate = $3,
			status = $4,
			rating = $5,
			rating_count = $6,
			acceptance_rate = $7,
			completion_rate = $8,
			total_deliveries = $9,
			zone_ids = $10,
			current_order_id = $11,
			go_online_at = $12,
			go_offline_at = $13,
			suspended_reason = $14,
			updated_at = $15,
			version = version + 1
		WHERE id = $1 AND version = $16
	`,
		driver.ID(),
		string(driver.VehicleType()),
		driver.LicensePlate(),
		string(driver.Status()),
		driver.Rating(),
		driver.RatingCount(),
		driver.AcceptanceRate(),
		driver.CompletionRate(),
		driver.TotalDeliveries(),
		driver.ZoneIDs(),
		nilIfEmptyStr(driver.CurrentOrderID()),
		driver.GoOnlineAt(),
		driver.GoOfflineAt(),
		nilIfEmptyStr(driver.SuspendedReason()),
		driver.UpdatedAt(),
		driver.Version(),
	)
	if err != nil {
		return mapDriverWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: optimistic lock failed for driver %s", domain.ErrDriverNotFound, driver.ID())
	}
	return nil
}

// ListOnlineByZone retrieves all online drivers serving the given zone.
// If zoneID is empty, returns all online drivers.
func (r *DriverRepository) ListOnlineByZone(ctx context.Context, exec port.Executor, zoneID string) ([]domain.Driver, error) {
	dbtx := toDBTX(exec)
	var rows pgx.Rows
	var err error
	if zoneID == "" {
		rows, err = dbtx.Query(ctx, `
			SELECT `+driverColumns+`
			FROM dispatch.drivers
			WHERE status = 'online'
			ORDER BY updated_at DESC
		`)
	} else {
		rows, err = dbtx.Query(ctx, `
			SELECT `+driverColumns+`
			FROM dispatch.drivers
			WHERE status = 'online' AND ($1 = ANY(zone_ids) OR zone_ids IS NULL)
			ORDER BY updated_at DESC
		`, zoneID)
	}
	if err != nil {
		return nil, fmt.Errorf("list online drivers: %w", err)
	}
	defer rows.Close()

	var items []domain.Driver
	for rows.Next() {
		d, err := scanDriver(rows)
		if err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ListAll retrieves all drivers (admin view) with pagination.
func (r *DriverRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Driver], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM dispatch.drivers`).Scan(&total)
	if err != nil {
		return port.Page[domain.Driver]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+driverColumns+`
		FROM dispatch.drivers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.Driver]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.Driver
	for rows.Next() {
		d, err := scanDriver(rows)
		if err != nil {
			return port.Page[domain.Driver]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.Driver]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.Driver]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ===== Mapper Helpers =====

const driverColumns = `id, user_id, vehicle_type, license_plate, status, rating, rating_count, acceptance_rate, completion_rate, total_deliveries, zone_ids, current_order_id, go_online_at, go_offline_at, suspended_reason, created_at, updated_at, version`

func scanDriver(s scanner) (domain.Driver, error) {
	var (
		id, userID, vehicleType, licensePlate, status string
		rating                                        float64
		ratingCount, acceptance, completion, total    int
		zoneIDs                                       []string
		currentOrderID                                *string
		goOnlineAt, goOfflineAt                       *time.Time
		suspendedReason                               *string
		createdAt, updatedAt                          time.Time
		version                                       int
	)
	if err := s.Scan(
		&id, &userID, &vehicleType, &licensePlate, &status,
		&rating, &ratingCount, &acceptance, &completion, &total,
		&zoneIDs, &currentOrderID, &goOnlineAt, &goOfflineAt, &suspendedReason,
		&createdAt, &updatedAt, &version,
	); err != nil {
		return domain.Driver{}, err
	}

	var currentOrderStr, suspendedStr string
	if currentOrderID != nil {
		currentOrderStr = *currentOrderID
	}
	if suspendedReason != nil {
		suspendedStr = *suspendedReason
	}

	return domain.RehydrateDriver(
		id, userID,
		domain.VehicleType(vehicleType),
		licensePlate,
		domain.DriverStatus(status),
		rating, ratingCount, acceptance, completion, total,
		zoneIDs,
		currentOrderStr,
		goOnlineAt, goOfflineAt,
		suspendedStr,
		createdAt, updatedAt,
		version,
	), nil
}

func mapDriverWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domain.ErrDriverAlreadyExists
		}
	}
	return fmt.Errorf("driver write: %w", err)
}

func mapDriverReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDriverNotFound
	}
	return fmt.Errorf("driver read: %w", err)
}
