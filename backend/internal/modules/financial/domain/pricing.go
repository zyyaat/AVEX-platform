// Package domain pricing: PricingRule, SurgeZone, and PriceQuote value objects.
//
// The pricing engine computes a delivery fee from:
//   base_fee + (per_km_rate * distance_km) + (per_min_rate * duration_min)
// then clamps to [min_fee, max_fee], applies surge multiplier, and
// finally applies tax (VAT).
//
// Pricing rules are zone-specific and time-bounded (valid_from..valid_to).
// Surge zones are time-windowed multipliers per zone.
//
// All monetary values use integer cents.
//
// Imports stdlib only.
package domain

import (
        "fmt"
        "time"
)

// PricingRule is a zone-specific delivery pricing rule.
type PricingRule struct {
        id                   string
        zoneID               string
        currency             string
        baseFee              Money
        perKmRate            Money // added per km of distance
        perMinRate           Money // added per minute of duration
        minFee               Money // floor after formula, before surge
        maxFee               *Money // ceiling after formula, before surge (NULL = no cap)
        freeDeliveryThreshold *Money // order total above which delivery is free (NULL = never)
        isActive             bool
        validFrom            time.Time
        validTo              *time.Time
        createdAt            time.Time
        updatedAt            time.Time
}

// NewPricingRule creates a new rule with validation.
func NewPricingRule(
        id, zoneID, currency string,
        baseFee, perKmRate, perMinRate, minFee Money,
        maxFee *Money,
        freeDeliveryThreshold *Money,
        validFrom time.Time,
        validTo *time.Time,
        now time.Time,
) (PricingRule, error) {
        if id == "" {
                return PricingRule{}, fmt.Errorf("%w: pricing rule id is required", ErrInvalidID)
        }
        if zoneID == "" {
                return PricingRule{}, fmt.Errorf("%w: zone id is required", ErrInvalidInput)
        }
        if len(currency) != 3 {
                return PricingRule{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
        }
        // Validate all Money currencies match.
        for i, m := range []Money{baseFee, perKmRate, perMinRate, minFee} {
                if m.Currency() != currency {
                        return PricingRule{}, fmt.Errorf("%w: money[%d] currency %s vs %s", ErrCurrencyMismatch, i, m.Currency(), currency)
                }
        }
        if maxFee != nil && maxFee.Currency() != currency {
                return PricingRule{}, fmt.Errorf("%w: maxFee currency %s vs %s", ErrCurrencyMismatch, maxFee.Currency(), currency)
        }
        if freeDeliveryThreshold != nil && freeDeliveryThreshold.Currency() != currency {
                return PricingRule{}, fmt.Errorf("%w: freeDeliveryThreshold currency %s vs %s", ErrCurrencyMismatch, freeDeliveryThreshold.Currency(), currency)
        }
        if validTo != nil && !validFrom.Before(*validTo) {
                return PricingRule{}, fmt.Errorf("%w: valid_to must be after valid_from", ErrInvalidInput)
        }
        return PricingRule{
                id:                    id,
                zoneID:                zoneID,
                currency:              currency,
                baseFee:               baseFee,
                perKmRate:             perKmRate,
                perMinRate:            perMinRate,
                minFee:                minFee,
                maxFee:                maxFee,
                freeDeliveryThreshold: freeDeliveryThreshold,
                isActive:              true,
                validFrom:             validFrom,
                validTo:               validTo,
                createdAt:             now,
                updatedAt:             now,
        }, nil
}

// RehydratePricingRule reconstructs from persistence.
func RehydratePricingRule(
        id, zoneID, currency string,
        baseFee, perKmRate, perMinRate, minFee Money,
        maxFee *Money,
        freeDeliveryThreshold *Money,
        isActive bool,
        validFrom time.Time,
        validTo *time.Time,
        createdAt, updatedAt time.Time,
) PricingRule {
        return PricingRule{
                id:                    id,
                zoneID:                zoneID,
                currency:              currency,
                baseFee:               baseFee,
                perKmRate:             perKmRate,
                perMinRate:            perMinRate,
                minFee:                minFee,
                maxFee:                maxFee,
                freeDeliveryThreshold: freeDeliveryThreshold,
                isActive:              isActive,
                validFrom:             validFrom,
                validTo:               validTo,
                createdAt:             createdAt,
                updatedAt:             updatedAt,
        }
}

// ===== Accessors =====

func (r PricingRule) ID() string                    { return r.id }
func (r PricingRule) ZoneID() string                { return r.zoneID }
func (r PricingRule) Currency() string              { return r.currency }
func (r PricingRule) BaseFee() Money                { return r.baseFee }
func (r PricingRule) PerKmRate() Money              { return r.perKmRate }
func (r PricingRule) PerMinRate() Money             { return r.perMinRate }
func (r PricingRule) MinFee() Money                 { return r.minFee }
func (r PricingRule) MaxFee() *Money                { return r.maxFee }
func (r PricingRule) FreeDeliveryThreshold() *Money { return r.freeDeliveryThreshold }
func (r PricingRule) IsActive() bool                { return r.isActive }
func (r PricingRule) ValidFrom() time.Time          { return r.validFrom }
func (r PricingRule) ValidTo() *time.Time           { return r.validTo }
func (r PricingRule) CreatedAt() time.Time          { return r.createdAt }
func (r PricingRule) UpdatedAt() time.Time          { return r.updatedAt }

// IsApplicableAt reports whether the rule is active and within its validity window.
func (r PricingRule) IsApplicableAt(now time.Time) bool {
        if !r.isActive {
                return false
        }
        if now.Before(r.validFrom) {
                return false
        }
        if r.validTo != nil && now.After(*r.validTo) {
                return false
        }
        return true
}

// ===== Surge Zone =====

// SurgeZone is a time-windowed surge multiplier for a zone.
type SurgeZone struct {
        id          string
        zoneID      string
        multiplier  float64 // e.g. 1.5 = +50%
        reason      string
        dayOfWeek   *int    // 0-6, NULL = all days
        startTime   string  // "HH:MM", empty = midnight
        endTime     string  // "HH:MM", empty = midnight
        isActive    bool
        createdAt   time.Time
}

// NewSurgeZone creates a new surge zone with validation.
func NewSurgeZone(
        id, zoneID string,
        multiplier float64,
        reason string,
        dayOfWeek *int,
        startTime, endTime string,
        now time.Time,
) (SurgeZone, error) {
        if id == "" {
                return SurgeZone{}, fmt.Errorf("%w: surge zone id is required", ErrInvalidID)
        }
        if zoneID == "" {
                return SurgeZone{}, fmt.Errorf("%w: zone id is required", ErrInvalidInput)
        }
        if multiplier < 1.0 {
                return SurgeZone{}, fmt.Errorf("%w: %f", ErrSurgeMultiplierInvalid, multiplier)
        }
        if dayOfWeek != nil && (*dayOfWeek < 0 || *dayOfWeek > 6) {
                return SurgeZone{}, fmt.Errorf("%w: day_of_week %d", ErrInvalidInput, *dayOfWeek)
        }
        return SurgeZone{
                id:         id,
                zoneID:     zoneID,
                multiplier: multiplier,
                reason:     reason,
                dayOfWeek:  dayOfWeek,
                startTime:  startTime,
                endTime:    endTime,
                isActive:   true,
                createdAt:  now,
        }, nil
}

// RehydrateSurgeZone reconstructs from persistence.
func RehydrateSurgeZone(
        id, zoneID string,
        multiplier float64,
        reason string,
        dayOfWeek *int,
        startTime, endTime string,
        isActive bool,
        createdAt time.Time,
) SurgeZone {
        return SurgeZone{
                id:         id,
                zoneID:     zoneID,
                multiplier: multiplier,
                reason:     reason,
                dayOfWeek:  dayOfWeek,
                startTime:  startTime,
                endTime:    endTime,
                isActive:   isActive,
                createdAt:  createdAt,
        }
}

func (s SurgeZone) ID() string         { return s.id }
func (s SurgeZone) ZoneID() string     { return s.zoneID }
func (s SurgeZone) Multiplier() float64 { return s.multiplier }
func (s SurgeZone) Reason() string     { return s.reason }
func (s SurgeZone) DayOfWeek() *int    { return s.dayOfWeek }
func (s SurgeZone) StartTime() string  { return s.startTime }
func (s SurgeZone) EndTime() string    { return s.endTime }
func (s SurgeZone) IsActive() bool     { return s.isActive }
func (s SurgeZone) CreatedAt() time.Time { return s.createdAt }

// IsApplicableAt reports whether the surge zone applies at the given time.
func (s SurgeZone) IsApplicableAt(now time.Time) bool {
        if !s.isActive {
                return false
        }
        if s.dayOfWeek != nil {
                // Go's time.Weekday: Sunday=0, Monday=1, ..., Saturday=6 — matches our 0-6 convention.
                if int(now.Weekday()) != *s.dayOfWeek {
                        return false
                }
        }
        // Time-of-day check (best-effort string comparison).
        if s.startTime != "" && s.endTime != "" {
                hhmm := now.Format("15:04")
                if hhmm < s.startTime || hhmm >= s.endTime {
                        return false
                }
        }
        return true
}

// ===== Tax =====

// Tax is a tax rule (e.g. 14% VAT).
type Tax struct {
        id         string
        name       string
        rate       float64 // e.g. 14.0 for 14%
        appliesTo  string  // 'delivery_fee' | 'order_total' | 'service_fee'
        isActive   bool
        createdAt  time.Time
}

const (
        TaxAppliesToDeliveryFee TaxAppliesTo = "delivery_fee"
        TaxAppliesToOrderTotal  TaxAppliesTo = "order_total"
        TaxAppliesToServiceFee  TaxAppliesTo = "service_fee"
)

// TaxAppliesTo is the type alias for tax application scope.
type TaxAppliesTo string

// NewTax creates a new tax rule.
func NewTax(id, name string, rate float64, appliesTo TaxAppliesTo, now time.Time) (Tax, error) {
        if id == "" {
                return Tax{}, fmt.Errorf("%w: tax id is required", ErrInvalidID)
        }
        if name == "" {
                return Tax{}, fmt.Errorf("%w: tax name is required", ErrInvalidInput)
        }
        if rate < 0 {
                return Tax{}, fmt.Errorf("%w: rate %f", ErrInvalidMoneyAmount, rate)
        }
        switch appliesTo {
        case TaxAppliesToDeliveryFee, TaxAppliesToOrderTotal, TaxAppliesToServiceFee:
        default:
                return Tax{}, fmt.Errorf("%w: applies_to %s", ErrInvalidInput, appliesTo)
        }
        return Tax{
                id:        id,
                name:      name,
                rate:      rate,
                appliesTo: string(appliesTo),
                isActive:  true,
                createdAt: now,
        }, nil
}

func (t Tax) ID() string         { return t.id }
func (t Tax) Name() string       { return t.name }
func (t Tax) Rate() float64      { return t.rate }
func (t Tax) AppliesTo() string  { return t.appliesTo }
func (t Tax) IsActive() bool     { return t.isActive }
func (t Tax) CreatedAt() time.Time { return t.createdAt }

// RehydrateTax reconstructs a Tax from persistence (bypasses constructor validation).
func RehydrateTax(id, name string, rate float64, appliesTo string, isActive bool, createdAt time.Time) Tax {
        return Tax{
                id:        id,
                name:      name,
                rate:      rate,
                appliesTo: appliesTo,
                isActive:  isActive,
                createdAt: createdAt,
        }
}

// ApplyTax returns the tax amount for a given base amount.
// Uses integer math: tax_amount = (base.amount * rate) / 100.
func ApplyTax(base Money, rate float64) (Money, error) {
        if rate < 0 {
                return ZeroMoney(base.Currency()), fmt.Errorf("%w: rate %f", ErrInvalidPercentage, rate)
        }
        // Convert float rate to integer math: numerator = rate * 100, denominator = 100 * 100
        // amount * (rate*100) / (100*100) = amount * rate / 100
        // To preserve precision, multiply first then divide.
        numerator := int64(rate * 100) // e.g. 14.0 -> 1400
        if numerator < 0 {
                return ZeroMoney(base.Currency()), fmt.Errorf("%w: negative rate", ErrInvalidPercentage)
        }
        result := (base.Amount() * numerator) / 10000
        return Money{amount: result, currency: base.Currency()}, nil
}

// ===== Price Quote =====

// PriceQuote is an immutable value object containing all components of a
// delivery price calculation. Returned by the pricing service to the caller.
type PriceQuote struct {
        currency          string
        distanceKM        float64
        durationMin       int
        baseFee           Money
        distanceFee       Money
        timeFee           Money
        subtotal          Money // base + distance + time (before surge)
        surgeMultiplier   float64
        surgeAdjustment   Money // additional amount due to surge
        postSurge         Money // subtotal + surgeAdjustment
        discount          Money // promotion discount (applied to delivery fee)
        postDiscount      Money // postSurge - discount
        taxRate           float64
        taxAmount         Money
        total             Money // postDiscount + taxAmount
        isFreeDelivery    bool  // true if free delivery threshold met
        freeDeliveryReason string
}

// NewPriceQuote constructs a PriceQuote with validation. Used by service layer.
func NewPriceQuote(
        currency string,
        distanceKM float64,
        durationMin int,
        baseFee, distanceFee, timeFee, subtotal Money,
        surgeMultiplier float64,
        surgeAdjustment, postSurge, discount, postDiscount Money,
        taxRate float64,
        taxAmount, total Money,
        isFreeDelivery bool,
        freeDeliveryReason string,
) (PriceQuote, error) {
        if distanceKM < 0 {
                return PriceQuote{}, ErrInvalidDistance
        }
        if durationMin < 0 {
                return PriceQuote{}, ErrInvalidDuration
        }
        if len(currency) != 3 {
                return PriceQuote{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
        }
        return PriceQuote{
                currency:           currency,
                distanceKM:         distanceKM,
                durationMin:        durationMin,
                baseFee:            baseFee,
                distanceFee:        distanceFee,
                timeFee:            timeFee,
                subtotal:           subtotal,
                surgeMultiplier:    surgeMultiplier,
                surgeAdjustment:    surgeAdjustment,
                postSurge:          postSurge,
                discount:           discount,
                postDiscount:       postDiscount,
                taxRate:            taxRate,
                taxAmount:          taxAmount,
                total:              total,
                isFreeDelivery:     isFreeDelivery,
                freeDeliveryReason: freeDeliveryReason,
        }, nil
}

// ===== Accessors =====

func (q PriceQuote) Currency() string          { return q.currency }
func (q PriceQuote) DistanceKM() float64       { return q.distanceKM }
func (q PriceQuote) DurationMin() int          { return q.durationMin }
func (q PriceQuote) BaseFee() Money            { return q.baseFee }
func (q PriceQuote) DistanceFee() Money        { return q.distanceFee }
func (q PriceQuote) TimeFee() Money            { return q.timeFee }
func (q PriceQuote) Subtotal() Money           { return q.subtotal }
func (q PriceQuote) SurgeMultiplier() float64  { return q.surgeMultiplier }
func (q PriceQuote) SurgeAdjustment() Money    { return q.surgeAdjustment }
func (q PriceQuote) PostSurge() Money          { return q.postSurge }
func (q PriceQuote) Discount() Money           { return q.discount }
func (q PriceQuote) PostDiscount() Money       { return q.postDiscount }
func (q PriceQuote) TaxRate() float64          { return q.taxRate }
func (q PriceQuote) TaxAmount() Money          { return q.taxAmount }
func (q PriceQuote) Total() Money              { return q.total }
func (q PriceQuote) IsFreeDelivery() bool      { return q.isFreeDelivery }
func (q PriceQuote) FreeDeliveryReason() string { return q.freeDeliveryReason }
