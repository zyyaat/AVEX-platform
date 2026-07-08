// Package postgres offer_repository: DispatchOfferRepository implementation.
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

// DispatchOfferRepository implements port.DispatchOfferRepository.
type DispatchOfferRepository struct{}

var _ port.DispatchOfferRepository = (*DispatchOfferRepository)(nil)

// Create inserts a new dispatch offer.
func (r *DispatchOfferRepository) Create(ctx context.Context, exec port.Executor, offer domain.DispatchOffer) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO dispatch.offers (
			id, order_id, driver_id, zone_id, status,
			pickup_lat, pickup_lng, delivery_lat, delivery_lng,
			est_distance_m, est_duration_s, est_fare_cents, currency,
			offer_ttl_ms, offered_at, expires_at,
			responded_at, accepted_at, rejected_at, expired_at, cancelled_at,
			reject_reason, attempt_number, created_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24
		)
	`,
		offer.ID(),
		offer.OrderID(),
		offer.DriverID(),
		offer.ZoneID(),
		string(offer.Status()),
		offer.PickupLat(),
		offer.PickupLng(),
		offer.DeliveryLat(),
		offer.DeliveryLng(),
		offer.EstDistanceM(),
		offer.EstDurationS(),
		offer.EstFareCents(),
		offer.Currency(),
		offer.OfferTTL().Milliseconds(),
		offer.OfferedAt(),
		offer.ExpiresAt(),
		offer.RespondedAt(),
		offer.AcceptedAt(),
		offer.RejectedAt(),
		offer.ExpiredAt(),
		offer.CancelledAt(),
		offer.RejectReason(),
		offer.AttemptNumber(),
		offer.CreatedBy(),
	)
	if err != nil {
		return mapOfferWriteError(err)
	}
	return nil
}

// GetByID retrieves an offer by UUID.
func (r *DispatchOfferRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.DispatchOffer, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+offerColumns+` FROM dispatch.offers WHERE id = $1`, id)
	o, err := scanOffer(row)
	if err != nil {
		return nil, mapOfferReadError(err)
	}
	return &o, nil
}

// GetActiveOfferForOrder retrieves the pending offer for an order.
func (r *DispatchOfferRepository) GetActiveOfferForOrder(ctx context.Context, exec port.Executor, orderID string) (*domain.DispatchOffer, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+offerColumns+` FROM dispatch.offers WHERE order_id = $1 AND status = 'pending' LIMIT 1`, orderID)
	o, err := scanOffer(row)
	if err != nil {
		return nil, mapOfferReadError(err)
	}
	return &o, nil
}

// Update saves all fields of an existing offer.
func (r *DispatchOfferRepository) Update(ctx context.Context, exec port.Executor, offer domain.DispatchOffer) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE dispatch.offers SET
			status = $2,
			responded_at = $3,
			accepted_at = $4,
			rejected_at = $5,
			expired_at = $6,
			cancelled_at = $7,
			reject_reason = $8
		WHERE id = $1
	`,
		offer.ID(),
		string(offer.Status()),
		offer.RespondedAt(),
		offer.AcceptedAt(),
		offer.RejectedAt(),
		offer.ExpiredAt(),
		offer.CancelledAt(),
		offer.RejectReason(),
	)
	if err != nil {
		return mapOfferWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOfferNotFound
	}
	return nil
}

// ListByDriver retrieves offers for a driver with pagination.
func (r *DispatchOfferRepository) ListByDriver(ctx context.Context, exec port.Executor, driverID string, page port.PageQuery) (port.Page[domain.DispatchOffer], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM dispatch.offers WHERE driver_id = $1`, driverID).Scan(&total)
	if err != nil {
		return port.Page[domain.DispatchOffer]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+offerColumns+`
		FROM dispatch.offers
		WHERE driver_id = $1
		ORDER BY offered_at DESC
		LIMIT $2 OFFSET $3
	`, driverID, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.DispatchOffer]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.DispatchOffer
	for rows.Next() {
		o, err := scanOffer(rows)
		if err != nil {
			return port.Page[domain.DispatchOffer]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.DispatchOffer]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.DispatchOffer]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ListByOrder retrieves all offers for an order (across attempts).
func (r *DispatchOfferRepository) ListByOrder(ctx context.Context, exec port.Executor, orderID string) ([]domain.DispatchOffer, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+offerColumns+`
		FROM dispatch.offers
		WHERE order_id = $1
		ORDER BY attempt_number ASC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list by order: %w", err)
	}
	defer rows.Close()

	var items []domain.DispatchOffer
	for rows.Next() {
		o, err := scanOffer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// CountAttemptsForOrder returns the number of offers created for an order.
func (r *DispatchOfferRepository) CountAttemptsForOrder(ctx context.Context, exec port.Executor, orderID string) (int, error) {
	dbtx := toDBTX(exec)
	var count int
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM dispatch.offers WHERE order_id = $1`, orderID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count attempts: %w", err)
	}
	return count, nil
}

// ===== Mapper Helpers =====

const offerColumns = `id, order_id, driver_id, zone_id, status, pickup_lat, pickup_lng, delivery_lat, delivery_lng, est_distance_m, est_duration_s, est_fare_cents, currency, offer_ttl_ms, offered_at, expires_at, responded_at, accepted_at, rejected_at, expired_at, cancelled_at, reject_reason, attempt_number, created_by`

func scanOffer(s scanner) (domain.DispatchOffer, error) {
	var (
		id, orderID, driverID, status      string
		zoneID                             *string
		pickupLat, pickupLng, delLat, delLng float64
		estDist                            *int
		estDur                             *int
		estFare                            *int64
		currency                           string
		offerTTLms                         int64
		offeredAt, expiresAt               time.Time
		respondedAt, acceptedAt            *time.Time
		rejectedAt, expiredAt, cancelledAt *time.Time
		rejectReason                       string
		attemptNumber                      int
		createdBy                          string
	)
	if err := s.Scan(
		&id, &orderID, &driverID, &zoneID, &status,
		&pickupLat, &pickupLng, &delLat, &delLng,
		&estDist, &estDur, &estFare, &currency,
		&offerTTLms, &offeredAt, &expiresAt,
		&respondedAt, &acceptedAt, &rejectedAt, &expiredAt, &cancelledAt,
		&rejectReason, &attemptNumber, &createdBy,
	); err != nil {
		return domain.DispatchOffer{}, err
	}

	var zoneIDStr string
	if zoneID != nil {
		zoneIDStr = *zoneID
	}
	if currency == "" {
		currency = "EGP"
	}
	if createdBy == "" {
		createdBy = "system"
	}

	return domain.RehydrateDispatchOffer(
		id, orderID, driverID, zoneIDStr,
		domain.OfferStatus(status),
		pickupLat, pickupLng, delLat, delLng,
		estDist, estDur, estFare, currency,
		time.Duration(offerTTLms)*time.Millisecond,
		offeredAt, expiresAt,
		respondedAt, acceptedAt, rejectedAt, expiredAt, cancelledAt,
		rejectReason,
		attemptNumber,
		createdBy,
	), nil
}

func mapOfferWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domain.ErrOfferAlreadyExists
		}
	}
	return fmt.Errorf("offer write: %w", err)
}

func mapOfferReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrOfferNotFound
	}
	return fmt.Errorf("offer read: %w", err)
}
