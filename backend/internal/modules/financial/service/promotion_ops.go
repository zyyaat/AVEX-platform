// Package service promotion_ops: promotion CRUD, validate, redeem.
package service

import (
        "context"
        "errors"
        "fmt"

        "avex-backend/internal/modules/financial/domain"
        "avex-backend/internal/modules/financial/events"
        "avex-backend/internal/modules/financial/port"
)

// ===== CreatePromotion =====

func (s *Service) CreatePromotion(ctx context.Context, input port.CreatePromotionInput) (*port.PromotionDTO, error) {
        if input.Currency == "" {
                input.Currency = "EGP"
        }
        now := s.deps.Clock.Now()
        if input.ValidFrom.IsZero() {
                input.ValidFrom = now
        }

        id := s.deps.IDGenerator.NewID()
        promo, err := domain.NewPromotion(
                id, input.Code, input.Description,
                domain.PromotionType(input.PromoType),
                input.Value, input.Currency,
                input.MinOrderAmount, input.MaxDiscountAmount,
                input.UsageLimit, input.PerUserLimit,
                input.ValidFrom, input.ValidTo, now,
        )
        if err != nil {
                return nil, err
        }

        if err := s.deps.Repos.Promotions.Create(ctx, s.pool, promo); err != nil {
                return nil, err
        }

        dto := port.ToPromotionDTO(promo)
        return &dto, nil
}

// ===== GetPromotion / ListActivePromotions =====

func (s *Service) GetPromotion(ctx context.Context, id string) (*port.PromotionDTO, error) {
        promo, err := s.deps.Repos.Promotions.GetByID(ctx, s.pool, id)
        if err != nil {
                return nil, err
        }
        dto := port.ToPromotionDTO(*promo)
        return &dto, nil
}

func (s *Service) ListActivePromotions(ctx context.Context) ([]port.PromotionDTO, error) {
        now := s.deps.Clock.Now()
        promos, err := s.deps.Repos.Promotions.ListActive(ctx, s.pool, now)
        if err != nil {
                return nil, err
        }
        dtos := make([]port.PromotionDTO, 0, len(promos))
        for _, p := range promos {
                dtos = append(dtos, port.ToPromotionDTO(p))
        }
        return dtos, nil
}

// ===== ValidatePromotion =====

// ValidatePromotion checks whether a promo code is valid for a given order
// without persisting any state. Returns the discount amount if valid, or the
// reason for failure.
func (s *Service) ValidatePromotion(ctx context.Context, input port.ValidatePromoInput) (port.ValidatePromoResult, error) {
        if input.Currency == "" {
                input.Currency = "EGP"
        }
        now := s.deps.Clock.Now()

        promo, err := s.deps.Repos.Promotions.GetByCode(ctx, s.pool, input.Code)
        if err != nil {
                if errors.Is(err, domain.ErrPromotionNotFound) {
                        return port.ValidatePromoResult{Valid: false, Reason: "promotion not found"}, nil
                }
                return port.ValidatePromoResult{}, err
        }

        // Check basic usability (active, validity window, usage limit)
        if err := promo.IsUsable(now); err != nil {
                return port.ValidatePromoResult{Valid: false, Reason: err.Error()}, nil
        }

        // Per-user limit check
        if input.UserID != "" {
                count, err := s.deps.Repos.Redemptions.CountByUserAndPromotion(ctx, s.pool, promo.ID(), input.UserID)
                if err != nil {
                        return port.ValidatePromoResult{}, fmt.Errorf("count user redemptions: %w", err)
                }
                if count >= promo.PerUserLimit() {
                        return port.ValidatePromoResult{Valid: false, Reason: "per-user limit reached"}, nil
                }
        }

        // Calculate discount
        orderTotal, _ := domain.NewMoney(input.OrderTotal, input.Currency)
        deliveryFee, _ := domain.NewMoney(input.DeliveryFee, input.Currency)

        discount, err := promo.CalculateDiscount(orderTotal, deliveryFee, now)
        if err != nil {
                return port.ValidatePromoResult{Valid: false, Reason: err.Error()}, nil
        }

        return port.ValidatePromoResult{
                Valid:          true,
                DiscountAmount: discount.Amount(),
                Currency:       discount.Currency(),
        }, nil
}

// ===== RedeemPromotion =====

// RedeemPromotion atomically:
//   1. Validates the promo (same as ValidatePromotion).
//   2. Inserts a redemption record with UNIQUE(promotion_id, user_id, order_id).
//   3. Increments the promotion's usage_count atomically (guarded by usage_limit).
//
// All within a single transaction. Returns the redemption record on success.
func (s *Service) RedeemPromotion(ctx context.Context, input port.RedeemPromoInput) (*port.RedeemPromoResult, error) {
        if input.Currency == "" {
                input.Currency = "EGP"
        }
        if input.UserID == "" {
                return nil, fmt.Errorf("%w: user id is required", domain.ErrInvalidInput)
        }
        if input.OrderID == "" {
                return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
        }
        now := s.deps.Clock.Now()

        var result *port.RedeemPromoResult

        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                // Re-read promo inside transaction (avoid TOCTOU).
                promo, err := s.deps.Repos.Promotions.GetByCode(ctx, exec, input.Code)
                if err != nil {
                        return err
                }
                if err := promo.IsUsable(now); err != nil {
                        return err
                }

                // Per-user limit check (inside tx)
                count, err := s.deps.Repos.Redemptions.CountByUserAndPromotion(ctx, exec, promo.ID(), input.UserID)
                if err != nil {
                        return fmt.Errorf("count user redemptions: %w", err)
                }
                if count >= promo.PerUserLimit() {
                        return domain.ErrPromotionPerUserLimitReached
                }

                // Calculate discount
                orderTotal, _ := domain.NewMoney(input.OrderTotal, input.Currency)
                deliveryFee, _ := domain.NewMoney(input.DeliveryFee, input.Currency)
                discount, err := promo.CalculateDiscount(orderTotal, deliveryFee, now)
                if err != nil {
                        return err
                }

                // Increment usage_count (atomic, guarded by usage_limit)
                if err := s.deps.Repos.Promotions.IncrementUsage(ctx, exec, promo.ID()); err != nil {
                        return err
                }

                // Create redemption record
                redemptionID := s.deps.IDGenerator.NewID()
                redemption, err := domain.NewPromotionRedemption(redemptionID, promo.ID(), input.UserID, input.OrderID, discount, now)
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Redemptions.Create(ctx, exec, redemption); err != nil {
                        return err
                }

                // Publish promotion.redeemed event
                ec := s.eventContext(ctx, port.ActorContext{Type: "user", ID: input.UserID})
                envelope, err := events.PromotionRedeemedEnvelope(port.PromotionRedeemedPayload{
                        RedemptionID:  redemption.ID(),
                        PromotionID:   promo.ID(),
                        PromotionCode: promo.Code(),
                        UserID:        input.UserID,
                        OrderID:       input.OrderID,
                        DiscountCents: discount.Amount(),
                        Currency:      discount.Currency(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }

                result = &port.RedeemPromoResult{
                        RedemptionID:   redemption.ID(),
                        PromotionID:    promo.ID(),
                        DiscountAmount: discount.Amount(),
                        Currency:       discount.Currency(),
                }
                return nil
        })
        if err != nil {
                return nil, err
        }
        return result, nil
}
