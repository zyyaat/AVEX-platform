// Package service pricing_ops: pricing engine + pricing rule + surge zone management.
package service

import (
        "context"
        "errors"
        "fmt"

        "avex-backend/internal/modules/financial/domain"
        "avex-backend/internal/modules/financial/events"
        "avex-backend/internal/modules/financial/port"
)

// ===== CalculateQuote =====

// CalculateQuote computes a delivery price quote.
//
// Algorithm:
//   1. Load active pricing rule for (zone_id, currency) at the current time.
//   2. Compute subtotal = base_fee + (per_km_rate * distance) + (per_min_rate * duration).
//   3. Clamp subtotal to [min_fee, max_fee].
//   4. If order_total >= free_delivery_threshold, set is_free_delivery=true and zero the subtotal.
//   5. Apply surge multiplier: surge_adj = subtotal * (multiplier - 1). post_surge = subtotal + surge_adj.
//   6. If promo_code is provided, validate + calculate discount. post_discount = post_surge - discount.
//      NOTE: free_delivery promos zero out the delivery fee completely.
//   7. Apply tax (VAT) to post_discount. total = post_discount + tax_amount.
func (s *Service) CalculateQuote(ctx context.Context, input port.CalculateQuoteInput) (*port.CalculateQuoteResult, error) {
        if input.Currency == "" {
                input.Currency = "EGP"
        }
        now := s.deps.Clock.Now()

        // 1. Load pricing rule
        rule, err := s.deps.Repos.PricingRules.GetActiveForZone(ctx, s.pool, input.ZoneID, input.Currency, now)
        if err != nil {
                return nil, err
        }

        // 2. Compute subtotal
        // distanceFee = per_km_rate * distance_km (in cents, using integer math)
        // To avoid float issues: scale distance to int (km * 100 = meters / 10)
        // Easier: per_km_rate.Amount() * int(distance_km_float * 100) / 100
        distMeters := int64(input.DistanceKM * 1000)
        if distMeters < 0 {
                distMeters = 0
        }
        distFeeCents := (rule.PerKmRate().Amount() * distMeters) / 1000
        timeFeeCents := rule.PerMinRate().Amount() * int64(input.DurationMin)

        subtotalCents := rule.BaseFee().Amount() + distFeeCents + timeFeeCents

        // 3. Clamp to min_fee
        if subtotalCents < rule.MinFee().Amount() {
                subtotalCents = rule.MinFee().Amount()
        }
        // Clamp to max_fee if set
        if rule.MaxFee() != nil && subtotalCents > rule.MaxFee().Amount() {
                subtotalCents = rule.MaxFee().Amount()
        }

        subtotal, _ := domain.NewMoney(subtotalCents, input.Currency)
        baseFee := rule.BaseFee()
        distFee, _ := domain.NewMoney(distFeeCents, input.Currency)
        timeFee, _ := domain.NewMoney(timeFeeCents, input.Currency)

        // 4. Free delivery threshold check
        isFreeDelivery := false
        freeDeliveryReason := ""
        if rule.FreeDeliveryThreshold() != nil && input.OrderTotal >= rule.FreeDeliveryThreshold().Amount() {
                isFreeDelivery = true
                freeDeliveryReason = "order_total_exceeds_threshold"
                subtotal = domain.ZeroMoney(input.Currency)
        }

        // 5. Surge
        // Find the highest applicable surge multiplier for this zone at this time.
        surgeMultiplier := 1.0
        surges, err := s.deps.Repos.SurgeZones.GetActiveForZone(ctx, s.pool, input.ZoneID)
        if err != nil {
                return nil, fmt.Errorf("load surge zones: %w", err)
        }
        for _, surge := range surges {
                if surge.IsApplicableAt(now) {
                        if surge.Multiplier() > surgeMultiplier {
                                surgeMultiplier = surge.Multiplier()
                        }
                }
        }

        // surgeAdjustment = subtotal * (multiplier - 1)
        // Use integer math: surge_adj_cents = subtotal_cents * (multiplier_num - multiplier_den) / multiplier_den
        // With multiplier = 1.5 -> num/den = 3/2, surge_adj = subtotal * 1 / 2
        // We use a simpler approach: scale to 100x for precision
        surgeAdjCents := int64(0)
        if surgeMultiplier > 1.0 {
                surgeAdjCents = (subtotal.Amount() * int64((surgeMultiplier-1.0)*100)) / 100
        }
        surgeAdj, _ := domain.NewMoney(surgeAdjCents, input.Currency)
        postSurge, _ := subtotal.Add(surgeAdj)

        // 6. Promotion
        discount := domain.ZeroMoney(input.Currency)
        var promoCode string
        if input.PromoCode != "" {
                promo, err := s.deps.Repos.Promotions.GetByCode(ctx, s.pool, input.PromoCode)
                if err == nil {
                        // Check per-user limit if userID provided
                        if input.UserID != "" {
                                count, err := s.deps.Repos.Redemptions.CountByUserAndPromotion(ctx, s.pool, promo.ID(), input.UserID)
                                if err != nil {
                                        return nil, fmt.Errorf("count user redemptions: %w", err)
                                }
                                if count >= promo.PerUserLimit() {
                                        // Skip discount — promo not eligible for this user
                                        goto buildResult
                                }
                        }
                        orderTotalM, _ := domain.NewMoney(input.OrderTotal, input.Currency)
                        disc, err := promo.CalculateDiscount(orderTotalM, postSurge, now)
                        if err == nil {
                                discount = disc
                                promoCode = promo.Code()
                                // If free_delivery promo, force postSurge to 0 effectively (discount = postSurge).
                                if promo.Type() == domain.PromoTypeFreeDelivery && !isFreeDelivery {
                                        // Cap discount at postSurge
                                        if discount.Amount() > postSurge.Amount() {
                                                discount, _ = domain.NewMoney(postSurge.Amount(), postSurge.Currency())
                                        }
                                }
                        }
                }
        }

buildResult:
        postDiscount, err := postSurge.Subtract(discount)
        if err != nil {
                // discount > postSurge (shouldn't happen given cap logic, but guard anyway)
                postDiscount = domain.ZeroMoney(input.Currency)
        }

        // 7. Tax
        taxRate := 0.0
        taxes, err := s.deps.Repos.Taxes.ListActive(ctx, s.pool)
        if err != nil {
                return nil, fmt.Errorf("list taxes: %w", err)
        }
        for _, tax := range taxes {
                // Apply all taxes on delivery fee (post_discount)
                if tax.AppliesTo() == "delivery_fee" {
                        taxRate += tax.Rate()
                }
        }
        taxAmount, err := domain.ApplyTax(postDiscount, taxRate)
        if err != nil {
                return nil, err
        }
        total, _ := postDiscount.Add(taxAmount)

        // Build PriceQuote value object (validates invariants)
        _, err = domain.NewPriceQuote(
                input.Currency, input.DistanceKM, input.DurationMin,
                baseFee, distFee, timeFee, subtotal,
                surgeMultiplier, surgeAdj, postSurge,
                discount, postDiscount, taxRate, taxAmount, total,
                isFreeDelivery, freeDeliveryReason,
        )
        if err != nil {
                return nil, err
        }

        result := &port.CalculateQuoteResult{
                Currency:           input.Currency,
                DistanceKM:         input.DistanceKM,
                DurationMin:        input.DurationMin,
                BaseFee:            baseFee.Amount(),
                DistanceFee:        distFee.Amount(),
                TimeFee:            timeFee.Amount(),
                Subtotal:           subtotal.Amount(),
                SurgeMultiplier:    surgeMultiplier,
                SurgeAdjustment:    surgeAdj.Amount(),
                PostSurge:          postSurge.Amount(),
                Discount:           discount.Amount(),
                PostDiscount:       postDiscount.Amount(),
                TaxRate:            taxRate,
                TaxAmount:          taxAmount.Amount(),
                Total:              total.Amount(),
                IsFreeDelivery:     isFreeDelivery,
                FreeDeliveryReason: freeDeliveryReason,
        }
        // Suppress promoCode unused warning if no promo was applied.
        _ = promoCode

        // Publish pricing.calculated event (best-effort, non-transactional since this is a query)
        // We don't wrap in a transaction because we don't mutate any state.
        ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
        if input.UserID != "" {
                ec.Actor.ID = input.UserID
        }
        envelope, err := events.PricingCalculatedEnvelope(port.PricingCalculatedPayload{
                ZoneID:          input.ZoneID,
                Currency:        input.Currency,
                DistanceKM:      input.DistanceKM,
                DurationMin:     input.DurationMin,
                SubtotalCents:   subtotal.Amount(),
                SurgeMultiplier: surgeMultiplier,
                DiscountCents:   discount.Amount(),
                TaxCents:        taxAmount.Amount(),
                TotalCents:      total.Amount(),
                IsFreeDelivery:  isFreeDelivery,
        }, ec)
        if err != nil {
                // Don't fail the request if event publishing fails — log only.
                s.deps.Logger.Warn("failed to build pricing event envelope", "error", err)
        } else {
                // Persist the event in a small best-effort transaction.
                // If it fails, we don't fail the API call.
                _ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                        return s.deps.EventPublisher.Publish(ctx, exec, envelope)
                })
        }

        return result, nil
}

// ===== Pricing Rule Management =====

func (s *Service) CreatePricingRule(ctx context.Context, input port.CreatePricingRuleInput) (*port.PricingRuleDTO, error) {
        if input.Currency == "" {
                input.Currency = "EGP"
        }
        now := s.deps.Clock.Now()
        if input.ValidFrom.IsZero() {
                input.ValidFrom = now
        }

        baseFee, _ := domain.NewMoney(input.BaseFee, input.Currency)
        perKm, _ := domain.NewMoney(input.PerKmRate, input.Currency)
        perMin, _ := domain.NewMoney(input.PerMinRate, input.Currency)
        minFee, _ := domain.NewMoney(input.MinFee, input.Currency)

        var maxFee *domain.Money
        if input.MaxFee != nil {
                m, _ := domain.NewMoney(*input.MaxFee, input.Currency)
                maxFee = &m
        }
        var threshold *domain.Money
        if input.FreeDeliveryThreshold != nil {
                m, _ := domain.NewMoney(*input.FreeDeliveryThreshold, input.Currency)
                threshold = &m
        }

        id := s.deps.IDGenerator.NewID()
        rule, err := domain.NewPricingRule(id, input.ZoneID, input.Currency, baseFee, perKm, perMin, minFee, maxFee, threshold, input.ValidFrom, input.ValidTo, now)
        if err != nil {
                return nil, err
        }

        if err := s.deps.Repos.PricingRules.Create(ctx, s.pool, rule); err != nil {
                return nil, err
        }

        dto := port.ToPricingRuleDTO(rule)
        return &dto, nil
}

func (s *Service) ListPricingRules(ctx context.Context, page port.PageQuery) (port.Page[port.PricingRuleDTO], error) {
        result, err := s.deps.Repos.PricingRules.ListAll(ctx, s.pool, page)
        if err != nil {
                return port.Page[port.PricingRuleDTO]{}, err
        }
        dtos := make([]port.PricingRuleDTO, 0, len(result.Items))
        for _, r := range result.Items {
                dtos = append(dtos, port.ToPricingRuleDTO(r))
        }
        return port.Page[port.PricingRuleDTO]{
                Items:  dtos,
                Total:  result.Total,
                Limit:  result.Limit,
                Offset: result.Offset,
        }, nil
}

// ===== Surge Zone Management =====

func (s *Service) CreateSurgeZone(ctx context.Context, input port.CreateSurgeZoneInput) (*port.SurgeZoneDTO, error) {
        now := s.deps.Clock.Now()
        id := s.deps.IDGenerator.NewID()
        surge, err := domain.NewSurgeZone(id, input.ZoneID, input.Multiplier, input.Reason, input.DayOfWeek, input.StartTime, input.EndTime, now)
        if err != nil {
                return nil, err
        }
        if err := s.deps.Repos.SurgeZones.Create(ctx, s.pool, surge); err != nil {
                return nil, err
        }
        dto := port.ToSurgeZoneDTO(surge)
        return &dto, nil
}

func (s *Service) ListSurgeZones(ctx context.Context, page port.PageQuery) (port.Page[port.SurgeZoneDTO], error) {
        result, err := s.deps.Repos.SurgeZones.ListAll(ctx, s.pool, page)
        if err != nil {
                return port.Page[port.SurgeZoneDTO]{}, err
        }
        dtos := make([]port.SurgeZoneDTO, 0, len(result.Items))
        for _, s := range result.Items {
                dtos = append(dtos, port.ToSurgeZoneDTO(s))
        }
        return port.Page[port.SurgeZoneDTO]{
                Items:  dtos,
                Total:  result.Total,
                Limit:  result.Limit,
                Offset: result.Offset,
        }, nil
}

func (s *Service) DeactivateSurgeZone(ctx context.Context, id string) error {
        return s.deps.Repos.SurgeZones.Deactivate(ctx, s.pool, id)
}

// suppress unused import
var _ = errors.Is
