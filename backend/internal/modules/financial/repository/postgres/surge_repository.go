// Package postgres surge_repository: SurgeZoneRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/port"
)

// SurgeZoneRepository implements port.SurgeZoneRepository.
type SurgeZoneRepository struct{}

var _ port.SurgeZoneRepository = (*SurgeZoneRepository)(nil)

// Create inserts a new surge zone.
func (r *SurgeZoneRepository) Create(ctx context.Context, exec port.Executor, surge domain.SurgeZone) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.surge_zones (
			id, zone_id, multiplier, reason,
			day_of_week, start_time, end_time,
			is_active, created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9
		)
	`,
		surge.ID(),
		surge.ZoneID(),
		surge.Multiplier(),
		surge.Reason(),
		surge.DayOfWeek(),
		surge.StartTime(),
		surge.EndTime(),
		surge.IsActive(),
		surge.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("surge create: %w", err)
	}
	return nil
}

// GetByID retrieves a surge zone by UUID.
func (r *SurgeZoneRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.SurgeZone, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+surgeColumns+` FROM financial.surge_zones WHERE id = $1`, id)
	s, err := scanSurge(row)
	if err != nil {
		return nil, mapSurgeReadError(err)
	}
	return &s, nil
}

// GetActiveForZone retrieves all active surge zones for a given zone.
func (r *SurgeZoneRepository) GetActiveForZone(ctx context.Context, exec port.Executor, zoneID string) ([]domain.SurgeZone, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+surgeColumns+`
		FROM financial.surge_zones
		WHERE zone_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
	`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("list surge: %w", err)
	}
	defer rows.Close()

	var items []domain.SurgeZone
	for rows.Next() {
		s, err := scanSurge(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ListAll retrieves all surge zones (admin view) with pagination.
func (r *SurgeZoneRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.SurgeZone], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM financial.surge_zones`).Scan(&total)
	if err != nil {
		return port.Page[domain.SurgeZone]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+surgeColumns+`
		FROM financial.surge_zones
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.SurgeZone]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.SurgeZone
	for rows.Next() {
		s, err := scanSurge(rows)
		if err != nil {
			return port.Page[domain.SurgeZone]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.SurgeZone]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.SurgeZone]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// Deactivate marks a surge zone as inactive.
func (r *SurgeZoneRepository) Deactivate(ctx context.Context, exec port.Executor, id string) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `UPDATE financial.surge_zones SET is_active = FALSE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: surge zone %s", domain.ErrInvalidID, id)
	}
	return nil
}

// ===== Mapper Helpers =====

const surgeColumns = `id, zone_id, multiplier, reason, day_of_week, start_time, end_time, is_active, created_at`

func scanSurge(s scanner) (domain.SurgeZone, error) {
	var (
		id, zoneID  string
		multiplier  float64
		reason      *string
		dayOfWeek   *int
		startTime   string
		endTime     string
		isActive    bool
		createdAt   time.Time
	)
	if err := s.Scan(&id, &zoneID, &multiplier, &reason, &dayOfWeek, &startTime, &endTime, &isActive, &createdAt); err != nil {
		return domain.SurgeZone{}, err
	}
	var reasonStr string
	if reason != nil {
		reasonStr = *reason
	}
	return domain.RehydrateSurgeZone(id, zoneID, multiplier, reasonStr, dayOfWeek, startTime, endTime, isActive, createdAt), nil
}

func mapSurgeReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidID
	}
	return fmt.Errorf("surge read: %w", err)
}
