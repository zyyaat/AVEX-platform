// Package domain tests: Promotion + Pricing.
package domain

import (
        "testing"
        "time"
)

func parseRFC(s string) time.Time {
        t, _ := time.Parse(time.RFC3339, s)
        return t
}

// ===== Promotion Tests =====

func TestNewPromotion(t *testing.T) {
        now := testNow()
        limit := 100
        maxDisc := int64(500)

        tests := []struct {
                name       string
                promoType  PromotionType
                value      int64
                currency   string
                validFrom  time.Time
                validTo    *time.Time
                perUser    int
                wantErr    error
        }{
                {"percent valid", PromoTypePercent, 20, "EGP", now, nil, 1, nil},
                {"fixed valid", PromoTypeFixed, 500, "EGP", now, nil, 1, nil},
                {"free_delivery valid", PromoTypeFreeDelivery, 0, "EGP", now, nil, 1, nil},
                {"percent > 100", PromoTypePercent, 150, "EGP", now, nil, 1, ErrInvalidDiscountValue},
                {"invalid type", PromotionType("bogus"), 10, "EGP", now, nil, 1, ErrInvalidPromoType},
                {"short currency", PromoTypePercent, 10, "EG", now, nil, 1, ErrInvalidCurrency},
                {"negative value", PromoTypeFixed, -5, "EGP", now, nil, 1, ErrInvalidDiscountValue},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        _, err := NewPromotion("p1", "CODE", "desc", tt.promoType, tt.value, tt.currency,
                                0, &maxDisc, &limit, tt.perUser, tt.validFrom, tt.validTo, now)
                        if tt.wantErr != nil {
                                if err == nil || !errIs(err, tt.wantErr) {
                                        t.Fatalf("expected %v, got %v", tt.wantErr, err)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                })
        }
}

func TestPromotionValidityWindow(t *testing.T) {
        now := testNow()
        validTo := now.Add(24 * time.Hour)
        limit := 100
        maxDisc := int64(500)

        // Valid now
        p, _ := NewPromotion("p1", "WELCOME", "desc", PromoTypePercent, 20, "EGP", 0, &maxDisc, &limit, 1, now.Add(-1*time.Hour), &validTo, now)
        if err := p.IsUsable(now); err != nil {
                t.Errorf("expected usable, got %v", err)
        }

        // Not yet valid (validFrom in future)
        future := now.Add(1 * time.Hour)
        p2, _ := NewPromotion("p2", "FUTURE", "desc", PromoTypePercent, 20, "EGP", 0, &maxDisc, &limit, 1, future, nil, now)
        if err := p2.IsUsable(now); err == nil || !errIs(err, ErrPromotionNotYetValid) {
                t.Errorf("expected ErrPromotionNotYetValid, got %v", err)
        }

        // Expired
        expired := now.Add(-1 * time.Hour)
        p3, _ := NewPromotion("p3", "OLD", "desc", PromoTypePercent, 20, "EGP", 0, &maxDisc, &limit, 1, now.Add(-2*time.Hour), &expired, now)
        if err := p3.IsUsable(now); err == nil || !errIs(err, ErrPromotionExpired) {
                t.Errorf("expected ErrPromotionExpired, got %v", err)
        }

        // Inactive
        p4, _ := NewPromotion("p4", "INACTIVE", "desc", PromoTypePercent, 20, "EGP", 0, &maxDisc, &limit, 1, now.Add(-1*time.Hour), nil, now)
        p4 = p4.SetActive(false, now)
        if err := p4.IsUsable(now); err == nil || !errIs(err, ErrPromotionInactive) {
                t.Errorf("expected ErrPromotionInactive, got %v", err)
        }

        // Usage limit reached
        p5, _ := NewPromotion("p5", "LIMIT", "desc", PromoTypePercent, 20, "EGP", 0, &maxDisc, &limit, 1, now.Add(-1*time.Hour), nil, now)
        // Set usageCount via IncrementUsage 100 times
        for i := 0; i < 100; i++ {
                p5 = p5.IncrementUsage()
        }
        if err := p5.IsUsable(now); err == nil || !errIs(err, ErrPromotionUsageLimitReached) {
                t.Errorf("expected ErrPromotionUsageLimitReached, got %v", err)
        }
}

func TestPromotionCalculateDiscountPercent(t *testing.T) {
        now := testNow()
        limit := 100
        maxDisc := int64(200) // cap at 200 cents

        // Promo with min_order = 1000
        p, _ := NewPromotion("p1", "PCT20", "20% off", PromoTypePercent, 20, "EGP", 1000, &maxDisc, &limit, 1, now.Add(-1*time.Hour), nil, now)

        orderTotal, _ := NewMoney(1000, "EGP")
        deliveryFee, _ := NewMoney(100, "EGP")

        disc, err := p.CalculateDiscount(orderTotal, deliveryFee, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        // 20% of 1000 = 200, which equals max cap
        if disc.Amount() != 200 {
                t.Errorf("expected 200, got %d", disc.Amount())
        }

        // Higher order total — should still be capped at 200
        bigOrder, _ := NewMoney(5000, "EGP")
        disc2, _ := p.CalculateDiscount(bigOrder, deliveryFee, now)
        if disc2.Amount() != 200 {
                t.Errorf("expected 200 (capped), got %d", disc2.Amount())
        }

        // Separate promo with no max cap and low min — test 20% of 100 = 20
        p2, _ := NewPromotion("p2", "PCT20_NOCAP", "20% off no cap", PromoTypePercent, 20, "EGP", 50, nil, &limit, 1, now.Add(-1*time.Hour), nil, now)
        smallOrder, _ := NewMoney(100, "EGP")
        disc3, _ := p2.CalculateDiscount(smallOrder, deliveryFee, now)
        if disc3.Amount() != 20 {
                t.Errorf("expected 20, got %d", disc3.Amount())
        }
}

func TestPromotionCalculateDiscountFixed(t *testing.T) {
        now := testNow()
        limit := 100
        // no max cap, min_order = 0 to test the "discount > order" clamp
        p, _ := NewPromotion("p1", "FLAT50", "50 cents off", PromoTypeFixed, 50, "EGP", 0, nil, &limit, 1, now.Add(-1*time.Hour), nil, now)

        orderTotal, _ := NewMoney(200, "EGP")
        deliveryFee, _ := NewMoney(50, "EGP")

        disc, err := p.CalculateDiscount(orderTotal, deliveryFee, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if disc.Amount() != 50 {
                t.Errorf("expected 50, got %d", disc.Amount())
        }

        // Order < discount — should clamp to order amount
        smallOrder, _ := NewMoney(30, "EGP")
        disc2, err := p.CalculateDiscount(smallOrder, deliveryFee, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if disc2.Amount() != 30 {
                t.Errorf("expected 30 (clamped to order), got %d", disc2.Amount())
        }
}

func TestPromotionCalculateDiscountFreeDelivery(t *testing.T) {
        now := testNow()
        limit := 100
        p, _ := NewPromotion("p1", "FREESHIP", "free delivery", PromoTypeFreeDelivery, 0, "EGP", 0, nil, &limit, 1, now.Add(-1*time.Hour), nil, now)

        orderTotal, _ := NewMoney(500, "EGP")
        deliveryFee, _ := NewMoney(150, "EGP")

        disc, err := p.CalculateDiscount(orderTotal, deliveryFee, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if disc.Amount() != 150 {
                t.Errorf("expected 150 (delivery fee), got %d", disc.Amount())
        }
}

func TestPromotionMinOrderNotMet(t *testing.T) {
        now := testNow()
        limit := 100
        p, _ := NewPromotion("p1", "MIN500", "min 500", PromoTypePercent, 10, "EGP", 500, nil, &limit, 1, now.Add(-1*time.Hour), nil, now)

        orderTotal, _ := NewMoney(300, "EGP")
        deliveryFee, _ := NewMoney(50, "EGP")

        _, err := p.CalculateDiscount(orderTotal, deliveryFee, now)
        if !errIs(err, ErrPromoMinOrderNotMet) {
                t.Fatalf("expected ErrPromoMinOrderNotMet, got %v", err)
        }
}

// ===== Pricing Tests =====

func TestNewPricingRule(t *testing.T) {
        now := testNow()
        base, _ := NewMoney(500, "EGP")
        perKm, _ := NewMoney(100, "EGP")
        perMin, _ := NewMoney(20, "EGP")
        minFee, _ := NewMoney(500, "EGP")
        maxFee, _ := NewMoney(5000, "EGP")
        threshold, _ := NewMoney(10000, "EGP")

        r, err := NewPricingRule("r1", "zone-1", "EGP", base, perKm, perMin, minFee, &maxFee, &threshold, now, nil, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if r.ID() != "r1" {
                t.Errorf("id mismatch")
        }
        if !r.IsActive() {
                t.Errorf("expected active")
        }
}

func TestPricingRuleCurrencyMismatch(t *testing.T) {
        now := testNow()
        base, _ := NewMoney(500, "USD") // wrong currency
        perKm, _ := NewMoney(100, "EGP")
        perMin, _ := NewMoney(20, "EGP")
        minFee, _ := NewMoney(500, "EGP")

        _, err := NewPricingRule("r1", "zone-1", "EGP", base, perKm, perMin, minFee, nil, nil, now, nil, now)
        if !errIs(err, ErrCurrencyMismatch) {
                t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
        }
}

func TestSurgeZone(t *testing.T) {
        now := testNow()
        monday := 1

        s, err := NewSurgeZone("s1", "zone-1", 1.5, "lunch rush", &monday, "12:00", "14:00", now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if s.Multiplier() != 1.5 {
                t.Errorf("expected 1.5, got %f", s.Multiplier())
        }

        // Invalid multiplier
        _, err = NewSurgeZone("s2", "zone-1", 0.5, "invalid", nil, "", "", now)
        if !errIs(err, ErrSurgeMultiplierInvalid) {
                t.Fatalf("expected ErrSurgeMultiplierInvalid, got %v", err)
        }

        // Invalid day of week
        badDay := 9
        _, err = NewSurgeZone("s3", "zone-1", 1.5, "bad", &badDay, "", "", now)
        if !errIs(err, ErrInvalidInput) {
                t.Fatalf("expected ErrInvalidInput, got %v", err)
        }
}

func TestApplyTax(t *testing.T) {
        // 14% VAT on 1000 = 140
        base, _ := NewMoney(1000, "EGP")
        tax, err := ApplyTax(base, 14.0)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if tax.Amount() != 140 {
                t.Errorf("expected 140, got %d", tax.Amount())
        }

        // 14% on 999 = 139.86 -> integer floor 139
        base2, _ := NewMoney(999, "EGP")
        tax2, _ := ApplyTax(base2, 14.0)
        if tax2.Amount() != 139 {
                t.Errorf("expected 139 (floor), got %d", tax2.Amount())
        }

        // 0% tax
        zero, _ := ApplyTax(base, 0.0)
        if zero.Amount() != 0 {
                t.Errorf("expected 0, got %d", zero.Amount())
        }

        // Negative rate
        _, err = ApplyTax(base, -1.0)
        if !errIs(err, ErrInvalidPercentage) {
                t.Fatalf("expected ErrInvalidPercentage, got %v", err)
        }
}

func TestNewTax(t *testing.T) {
        now := testNow()
        tests := []struct {
                name      string
                appliesTo TaxAppliesTo
                rate      float64
                wantErr   error
        }{
                {"valid VAT", TaxAppliesToOrderTotal, 14.0, nil},
                {"valid delivery", TaxAppliesToDeliveryFee, 5.0, nil},
                {"invalid applies_to", TaxAppliesTo("bogus"), 14.0, ErrInvalidInput},
                {"negative rate", TaxAppliesToOrderTotal, -1.0, ErrInvalidMoneyAmount},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        _, err := NewTax("t1", "VAT", tt.rate, tt.appliesTo, now)
                        if tt.wantErr != nil {
                                if err == nil || !errIs(err, tt.wantErr) {
                                        t.Fatalf("expected %v, got %v", tt.wantErr, err)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                })
        }
}

func TestNewPriceQuote(t *testing.T) {
        base, _ := NewMoney(500, "EGP")
        dist, _ := NewMoney(300, "EGP")
        timeFee, _ := NewMoney(100, "EGP")
        sub, _ := NewMoney(900, "EGP")
        postSurge, _ := NewMoney(1350, "EGP")
        surgeAdj, _ := NewMoney(450, "EGP")
        disc, _ := NewMoney(0, "EGP")
        postDisc, _ := NewMoney(1350, "EGP")
        taxAmt, _ := NewMoney(189, "EGP")
        total, _ := NewMoney(1539, "EGP")

        _, err := NewPriceQuote("EGP", 3.0, 5, base, dist, timeFee, sub, 1.5, surgeAdj, postSurge, disc, postDisc, 14.0, taxAmt, total, false, "")
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }

        // Negative distance
        _, err = NewPriceQuote("EGP", -1, 5, base, dist, timeFee, sub, 1.0, surgeAdj, postSurge, disc, postDisc, 14.0, taxAmt, total, false, "")
        if !errIs(err, ErrInvalidDistance) {
                t.Fatalf("expected ErrInvalidDistance, got %v", err)
        }

        // Negative duration
        _, err = NewPriceQuote("EGP", 3.0, -5, base, dist, timeFee, sub, 1.0, surgeAdj, postSurge, disc, postDisc, 14.0, taxAmt, total, false, "")
        if !errIs(err, ErrInvalidDuration) {
                t.Fatalf("expected ErrInvalidDuration, got %v", err)
        }
}

// Ensure parseRFC is used (silences unused import if all other tests are skipped).
var _ = parseRFC
