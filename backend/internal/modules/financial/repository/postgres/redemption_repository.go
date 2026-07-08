// Package postgres redemption_repository: PromotionRedemptionRepository implementation.
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

// PromotionRedemptionRepository implements port.PromotionRedemptionRepository.
type PromotionRedemptionRepository struct{}

var _ port.PromotionRedemptionRepository = (*PromotionRedemptionRepository)(nil)

// Create inserts a new redemption record.
// Returns ErrPromoAlreadyRedeemed on UNIQUE(promotion_id, user_id, order_id) violation.
func (r *PromotionRedemptionRepository) Create(ctx context.Context, exec port.Executor, redemption domain.PromotionRedemption) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.promotion_redemptions (
			id, promotion_id, user_id, order_id,
			discount_amount, currency, redeemed_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7
		)
	`,
		redemption.ID(),
		redemption.PromotionID(),
		redemption.UserID(),
		redemption.OrderID(),
		redemption.DiscountAmount().Amount(),
		redemption.Currency(),
		redemption.RedeemedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrPromoAlreadyRedeemed
		}
		return fmt.Errorf("redemption create: %w", err)
	}
	return nil
}

// CountByUserAndPromotion returns the number of redemptions for a (promo, user) pair.
func (r *PromotionRedemptionRepository) CountByUserAndPromotion(ctx context.Context, exec port.Executor, promotionID, userID string) (int, error) {
	dbtx := toDBTX(exec)
	var count int
	err := dbtx.QueryRow(ctx, `
		SELECT COUNT(*) FROM financial.promotion_redemptions
		WHERE promotion_id = $1 AND user_id = $2
	`, promotionID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count redemptions: %w", err)
	}
	return count, nil
}

// ListByUser retrieves all redemptions for a user.
func (r *PromotionRedemptionRepository) ListByUser(ctx context.Context, exec port.Executor, userID string, page port.PageQuery) (port.Page[domain.PromotionRedemption], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM financial.promotion_redemptions WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return port.Page[domain.PromotionRedemption]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+redemptionColumns+`
		FROM financial.promotion_redemptions
		WHERE user_id = $1
		ORDER BY redeemed_at DESC
		LIMIT $2 OFFSET $3
	`, userID, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.PromotionRedemption]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.PromotionRedemption
	for rows.Next() {
		x, err := scanRedemption(rows)
		if err != nil {
			return port.Page[domain.PromotionRedemption]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.PromotionRedemption]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.PromotionRedemption]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ListByOrder retrieves all redemptions applied to a specific order.
func (r *PromotionRedemptionRepository) ListByOrder(ctx context.Context, exec port.Executor, orderID string) ([]domain.PromotionRedemption, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+redemptionColumns+`
		FROM financial.promotion_redemptions
		WHERE order_id = $1
		ORDER BY redeemed_at DESC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list by order: %w", err)
	}
	defer rows.Close()

	var items []domain.PromotionRedemption
	for rows.Next() {
		x, err := scanRedemption(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ===== Mapper Helpers =====

const redemptionColumns = `id, promotion_id, user_id, order_id, discount_amount, currency, redeemed_at`

func scanRedemption(s scanner) (domain.PromotionRedemption, error) {
	var (
		id, promotionID, userID string
		orderID                 *string
		discountAmount          int64
		currency                string
		redeemedAt              time.Time
	)
	if err := s.Scan(&id, &promotionID, &userID, &orderID, &discountAmount, &currency, &redeemedAt); err != nil {
		return domain.PromotionRedemption{}, err
	}
	var orderIDStr string
	if orderID != nil {
		orderIDStr = *orderID
	}
	amt, _ := domain.NewMoney(discountAmount, currency)
	return domain.RehydratePromotionRedemption(id, promotionID, userID, orderIDStr, amt, currency, redeemedAt), nil
}

// Ensure pgx import is used.
var _ = pgx.ErrNoRows
