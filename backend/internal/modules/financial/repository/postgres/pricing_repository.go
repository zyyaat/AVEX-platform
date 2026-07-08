// Package postgres pricing_repository: PricingRuleRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/port"
)

// PricingRuleRepository implements port.PricingRuleRepository.
type PricingRuleRepository struct{}

var _ port.PricingRuleRepository = (*PricingRuleRepository)(nil)

// Create inserts a new pricing rule.
func (r *PricingRuleRepository) Create(ctx context.Context, exec port.Executor, rule domain.PricingRule) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.pricing_rules (
			id, zone_id, currency,
			base_fee, per_km_rate, per_min_rate, min_fee, max_fee, free_delivery_threshold,
			is_active, valid_from, valid_to, created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14
		)
	`,
		rule.ID(),
		rule.ZoneID(),
		rule.Currency(),
		rule.BaseFee().Amount(),
		rule.PerKmRate().Amount(),
		rule.PerMinRate().Amount(),
		rule.MinFee().Amount(),
		nullableMoneyAmount(rule.MaxFee()),
		nullableMoneyAmount(rule.FreeDeliveryThreshold()),
		rule.IsActive(),
		rule.ValidFrom(),
		rule.ValidTo(),
		rule.CreatedAt(),
		rule.UpdatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrPricingRuleAlreadyExists
		}
		return fmt.Errorf("pricing rule create: %w", err)
	}
	return nil
}

// GetByID retrieves a rule by UUID.
func (r *PricingRuleRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.PricingRule, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+pricingColumns+` FROM financial.pricing_rules WHERE id = $1`, id)
	rule, err := scanPricingRule(row)
	if err != nil {
		return nil, mapPricingReadError(err)
	}
	return &rule, nil
}

// GetActiveForZone retrieves the active pricing rule applicable to the given zone/currency.
func (r *PricingRuleRepository) GetActiveForZone(ctx context.Context, exec port.Executor, zoneID, currency string, now time.Time) (*domain.PricingRule, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `
		SELECT `+pricingColumns+`
		FROM financial.pricing_rules
		WHERE zone_id = $1 AND currency = $2 AND is_active = TRUE
		  AND valid_from <= $3
		  AND (valid_to IS NULL OR valid_to >= $3)
		ORDER BY valid_from DESC
		LIMIT 1
	`, zoneID, currency, now)
	rule, err := scanPricingRule(row)
	if err != nil {
		return nil, mapPricingReadError(err)
	}
	return &rule, nil
}

// ListAll retrieves all pricing rules (admin view) with pagination.
func (r *PricingRuleRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.PricingRule], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM financial.pricing_rules`).Scan(&total)
	if err != nil {
		return port.Page[domain.PricingRule]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+pricingColumns+`
		FROM financial.pricing_rules
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.PricingRule]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.PricingRule
	for rows.Next() {
		rule, err := scanPricingRule(rows)
		if err != nil {
			return port.Page[domain.PricingRule]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, rule)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.PricingRule]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.PricingRule]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ===== Mapper Helpers =====

const pricingColumns = `id, zone_id, currency, base_fee, per_km_rate, per_min_rate, min_fee, max_fee, free_delivery_threshold, is_active, valid_from, valid_to, created_at, updated_at`

func scanPricingRule(s scanner) (domain.PricingRule, error) {
	var (
		id, zoneID, currency            string
		baseFee, perKm, perMin, minFee  int64
		maxFee, freeThreshold           *int64
		isActive                        bool
		validFrom                       time.Time
		validTo                         *time.Time
		createdAt, updatedAt            time.Time
	)
	if err := s.Scan(
		&id, &zoneID, &currency,
		&baseFee, &perKm, &perMin, &minFee, &maxFee, &freeThreshold,
		&isActive, &validFrom, &validTo, &createdAt, &updatedAt,
	); err != nil {
		return domain.PricingRule{}, err
	}

	// Convert int64 amounts to Money
	baseM, _ := domain.NewMoney(baseFee, currency)
	perKmM, _ := domain.NewMoney(perKm, currency)
	perMinM, _ := domain.NewMoney(perMin, currency)
	minM, _ := domain.NewMoney(minFee, currency)
	var maxM *domain.Money
	if maxFee != nil {
		m, _ := domain.NewMoney(*maxFee, currency)
		maxM = &m
	}
	var thresholdM *domain.Money
	if freeThreshold != nil {
		m, _ := domain.NewMoney(*freeThreshold, currency)
		thresholdM = &m
	}

	return domain.RehydratePricingRule(
		id, zoneID, currency,
		baseM, perKmM, perMinM, minM, maxM, thresholdM,
		isActive, validFrom, validTo, createdAt, updatedAt,
	), nil
}

// nullableMoneyAmount returns the amount or nil if Money is nil.
func nullableMoneyAmount(m *domain.Money) any {
	if m == nil {
		return nil
	}
	return m.Amount()
}

func mapPricingReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPricingRuleNotFound
	}
	return fmt.Errorf("pricing rule read: %w", err)
}
