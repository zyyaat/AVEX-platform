// Package postgres promotion_repository: PromotionRepository implementation.
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

// PromotionRepository implements port.PromotionRepository using pgx/v5.
type PromotionRepository struct{}

var _ port.PromotionRepository = (*PromotionRepository)(nil)

// Create inserts a new promotion. Returns ErrPromotionCodeAlreadyExists on unique violation.
func (r *PromotionRepository) Create(ctx context.Context, exec port.Executor, promo domain.Promotion) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.promotions (
			id, code, description, promo_type, value, currency,
			min_order_amount, max_discount_amount,
			usage_limit, usage_count, per_user_limit,
			valid_from, valid_to, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16
		)
	`,
		promo.ID(),
		promo.Code(),
		promo.Description(),
		string(promo.Type()),
		promo.Value(),
		promo.Currency(),
		promo.MinOrderAmount(),
		promo.MaxDiscountAmount(),
		promo.UsageLimit(),
		promo.UsageCount(),
		promo.PerUserLimit(),
		promo.ValidFrom(),
		promo.ValidTo(),
		promo.IsActive(),
		promo.CreatedAt(),
		promo.UpdatedAt(),
	)
	if err != nil {
		return mapPromotionWriteError(err)
	}
	return nil
}

// GetByID retrieves a promotion by UUID.
func (r *PromotionRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Promotion, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+promotionColumns+` FROM financial.promotions WHERE id = $1`, id)
	p, err := scanPromotion(row)
	if err != nil {
		return nil, mapPromotionReadError(err)
	}
	return &p, nil
}

// GetByCode retrieves a promotion by its (case-insensitive) code.
func (r *PromotionRepository) GetByCode(ctx context.Context, exec port.Executor, code string) (*domain.Promotion, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+promotionColumns+` FROM financial.promotions WHERE UPPER(code) = UPPER($1)`, code)
	p, err := scanPromotion(row)
	if err != nil {
		return nil, mapPromotionReadError(err)
	}
	return &p, nil
}

// Update saves all fields of a promotion.
func (r *PromotionRepository) Update(ctx context.Context, exec port.Executor, promo domain.Promotion) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE financial.promotions SET
			description = $2,
			value = $3,
			min_order_amount = $4,
			max_discount_amount = $5,
			usage_limit = $6,
			usage_count = $7,
			per_user_limit = $8,
			valid_from = $9,
			valid_to = $10,
			is_active = $11,
			updated_at = $12
		WHERE id = $1
	`,
		promo.ID(),
		promo.Description(),
		promo.Value(),
		promo.MinOrderAmount(),
		promo.MaxDiscountAmount(),
		promo.UsageLimit(),
		promo.UsageCount(),
		promo.PerUserLimit(),
		promo.ValidFrom(),
		promo.ValidTo(),
		promo.IsActive(),
		promo.UpdatedAt(),
	)
	if err != nil {
		return mapPromotionWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPromotionNotFound
	}
	return nil
}

// IncrementUsage atomically increments usage_count by 1 with a guard against the limit.
// Returns ErrPromotionUsageLimitReached if the limit would be exceeded.
func (r *PromotionRepository) IncrementUsage(ctx context.Context, exec port.Executor, id string) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE financial.promotions
		SET usage_count = usage_count + 1, updated_at = NOW()
		WHERE id = $1 AND (usage_limit IS NULL OR usage_count < usage_limit)
	`, id)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPromotionUsageLimitReached
	}
	return nil
}

// ListActive retrieves all active promotions valid at the given time.
func (r *PromotionRepository) ListActive(ctx context.Context, exec port.Executor, now time.Time) ([]domain.Promotion, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+promotionColumns+`
		FROM financial.promotions
		WHERE is_active = TRUE
		  AND valid_from <= $1
		  AND (valid_to IS NULL OR valid_to >= $1)
		ORDER BY valid_from DESC
	`, now)
	if err != nil {
		return nil, fmt.Errorf("list active promotions: %w", err)
	}
	defer rows.Close()

	var items []domain.Promotion
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan promotion: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ListAll retrieves all promotions (admin view) with pagination.
func (r *PromotionRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Promotion], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM financial.promotions`).Scan(&total)
	if err != nil {
		return port.Page[domain.Promotion]{}, fmt.Errorf("count promotions: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+promotionColumns+`
		FROM financial.promotions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.Promotion]{}, fmt.Errorf("list promotions: %w", err)
	}
	defer rows.Close()

	var items []domain.Promotion
	for rows.Next() {
		p, err := scanPromotion(rows)
		if err != nil {
			return port.Page[domain.Promotion]{}, fmt.Errorf("scan promotion: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.Promotion]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.Promotion]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ===== Mapper Helpers =====

const promotionColumns = `id, code, description, promo_type, value, currency, min_order_amount, max_discount_amount, usage_limit, usage_count, per_user_limit, valid_from, valid_to, is_active, created_at, updated_at`

func scanPromotion(s scanner) (domain.Promotion, error) {
	var (
		id, code, description     string
		promoType, currency       string
		value, minOrderAmount     int64
		maxDiscountAmount         *int64
		usageLimit                *int
		usageCount, perUserLimit  int
		validFrom                 time.Time
		validTo                   *time.Time
		isActive                  bool
		createdAt, updatedAt      time.Time
	)
	var descPtr *string
	if err := s.Scan(
		&id, &code, &descPtr, &promoType, &value, &currency,
		&minOrderAmount, &maxDiscountAmount,
		&usageLimit, &usageCount, &perUserLimit,
		&validFrom, &validTo, &isActive,
		&createdAt, &updatedAt,
	); err != nil {
		return domain.Promotion{}, err
	}
	if descPtr != nil {
		description = *descPtr
	}

	return domain.RehydratePromotion(
		id, code, description,
		domain.PromotionType(promoType),
		value, currency,
		minOrderAmount, maxDiscountAmount,
		usageLimit, usageCount, perUserLimit,
		validFrom, validTo, isActive,
		createdAt, updatedAt,
	), nil
}

func mapPromotionWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domain.ErrPromotionCodeAlreadyExists
		}
	}
	return fmt.Errorf("promotion write: %w", err)
}

func mapPromotionReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPromotionNotFound
	}
	return fmt.Errorf("promotion read: %w", err)
}
