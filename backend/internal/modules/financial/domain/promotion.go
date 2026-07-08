// Package domain promotion: Promotion aggregate root + redemption logic.
//
// A Promotion is a discount code that customers can apply to orders.
// Types:
//   - percent:      discount = (order_amount * value) / 100, capped at max_discount
//   - fixed:        discount = value (flat amount, capped at order_amount)
//   - free_delivery: discount applied to delivery fee only, value is ignored
//
// Invariants enforced:
//   - usage_count <= usage_limit (NULL = unlimited)
//   - per-user redemptions <= per_user_limit
//   - valid_from <= now <= valid_to (NULL valid_to = no expiry)
//   - order_amount >= min_order_amount
//   - discount <= order_amount (no negative order totals)
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// PromotionType enumerates discount strategies.
type PromotionType string

const (
	PromoTypePercent      PromotionType = "percent"
	PromoTypeFixed        PromotionType = "fixed"
	PromoTypeFreeDelivery PromotionType = "free_delivery"
)

// Promotion is the aggregate root for a discount code.
type Promotion struct {
	id                string
	code              string
	description       string
	promoType         PromotionType
	value             int64    // percent (0-100) or fixed amount (cents)
	currency          string   // for fixed / free_delivery
	minOrderAmount    int64    // minimum order total in cents
	maxDiscountAmount *int64   // NULL = no cap
	usageLimit        *int     // NULL = unlimited
	usageCount        int      // current redemptions
	perUserLimit      int      // default 1
	validFrom         time.Time
	validTo           *time.Time // NULL = no expiry
	isActive          bool
	createdAt         time.Time
	updatedAt         time.Time
}

// NewPromotion creates a new Promotion with validation.
func NewPromotion(
	id, code, description string,
	promoType PromotionType,
	value int64,
	currency string,
	minOrderAmount int64,
	maxDiscountAmount *int64,
	usageLimit *int,
	perUserLimit int,
	validFrom time.Time,
	validTo *time.Time,
	now time.Time,
) (Promotion, error) {
	if id == "" {
		return Promotion{}, fmt.Errorf("%w: promotion id is required", ErrInvalidID)
	}
	if code == "" {
		return Promotion{}, fmt.Errorf("%w: promotion code is required", ErrInvalidInput)
	}
	if promoType != PromoTypePercent && promoType != PromoTypeFixed && promoType != PromoTypeFreeDelivery {
		return Promotion{}, fmt.Errorf("%w: %s", ErrInvalidPromoType, promoType)
	}
	if len(currency) != 3 {
		return Promotion{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
	if value < 0 {
		return Promotion{}, ErrInvalidDiscountValue
	}
	if promoType == PromoTypePercent && value > 100 {
		return Promotion{}, fmt.Errorf("%w: percent > 100", ErrInvalidDiscountValue)
	}
	if perUserLimit < 1 {
		perUserLimit = 1
	}
	if validTo != nil && !validFrom.Before(*validTo) {
		return Promotion{}, fmt.Errorf("%w: valid_to must be after valid_from", ErrInvalidInput)
	}
	return Promotion{
		id:                id,
		code:              code,
		description:       description,
		promoType:         promoType,
		value:             value,
		currency:          currency,
		minOrderAmount:    minOrderAmount,
		maxDiscountAmount: maxDiscountAmount,
		usageLimit:        usageLimit,
		usageCount:        0,
		perUserLimit:      perUserLimit,
		validFrom:         validFrom,
		validTo:           validTo,
		isActive:          true,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

// RehydratePromotion reconstructs a Promotion from persistence.
func RehydratePromotion(
	id, code, description string,
	promoType PromotionType,
	value int64,
	currency string,
	minOrderAmount int64,
	maxDiscountAmount *int64,
	usageLimit *int,
	usageCount, perUserLimit int,
	validFrom time.Time,
	validTo *time.Time,
	isActive bool,
	createdAt, updatedAt time.Time,
) Promotion {
	return Promotion{
		id:                id,
		code:              code,
		description:       description,
		promoType:         promoType,
		value:             value,
		currency:          currency,
		minOrderAmount:    minOrderAmount,
		maxDiscountAmount: maxDiscountAmount,
		usageLimit:        usageLimit,
		usageCount:        usageCount,
		perUserLimit:      perUserLimit,
		validFrom:         validFrom,
		validTo:           validTo,
		isActive:          isActive,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

// ===== Accessors =====

func (p Promotion) ID() string                  { return p.id }
func (p Promotion) Code() string                { return p.code }
func (p Promotion) Description() string         { return p.description }
func (p Promotion) Type() PromotionType         { return p.promoType }
func (p Promotion) Value() int64                { return p.value }
func (p Promotion) Currency() string            { return p.currency }
func (p Promotion) MinOrderAmount() int64       { return p.minOrderAmount }
func (p Promotion) MaxDiscountAmount() *int64   { return p.maxDiscountAmount }
func (p Promotion) UsageLimit() *int            { return p.usageLimit }
func (p Promotion) UsageCount() int             { return p.usageCount }
func (p Promotion) PerUserLimit() int           { return p.perUserLimit }
func (p Promotion) ValidFrom() time.Time        { return p.validFrom }
func (p Promotion) ValidTo() *time.Time         { return p.validTo }
func (p Promotion) IsActive() bool              { return p.isActive }
func (p Promotion) CreatedAt() time.Time        { return p.createdAt }
func (p Promotion) UpdatedAt() time.Time        { return p.updatedAt }

// IsValidityWindowOpen reports whether now is within the promotion's valid window.
func (p Promotion) IsValidityWindowOpen(now time.Time) bool {
	if now.Before(p.validFrom) {
		return false
	}
	if p.validTo != nil && now.After(*p.validTo) {
		return false
	}
	return true
}

// HasUsageLimitReached reports whether the global usage limit is exhausted.
func (p Promotion) HasUsageLimitReached() bool {
	if p.usageLimit == nil {
		return false
	}
	return p.usageCount >= *p.usageLimit
}

// IsUsable reports whether the promotion can be redeemed at the given time.
// Returns the specific domain error if not.
func (p Promotion) IsUsable(now time.Time) error {
	if !p.isActive {
		return ErrPromotionInactive
	}
	if now.Before(p.validFrom) {
		return ErrPromotionNotYetValid
	}
	if p.validTo != nil && now.After(*p.validTo) {
		return ErrPromotionExpired
	}
	if p.HasUsageLimitReached() {
		return ErrPromotionUsageLimitReached
	}
	return nil
}

// CalculateDiscount computes the discount amount for a given order total and
// (optionally) delivery fee. Returns the discount as a Money value in the
// promotion's currency.
//
// For 'percent' type: discount = min(orderTotal * value/100, maxDiscount)
// For 'fixed' type:   discount = min(value, orderTotal)
// For 'free_delivery': discount = deliveryFee (passed via the deliveryFee param)
//
// Returns ErrPromoMinOrderNotMet if orderTotal < minOrderAmount.
// Returns ErrCurrencyMismatch if orderTotal currency != promotion currency.
func (p Promotion) CalculateDiscount(orderTotal Money, deliveryFee Money, now time.Time) (Money, error) {
	if err := p.IsUsable(now); err != nil {
		return ZeroMoney(p.currency), err
	}
	if orderTotal.Currency() != p.currency {
		return ZeroMoney(p.currency), fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, orderTotal.Currency(), p.currency)
	}
	if orderTotal.Amount() < p.minOrderAmount {
		return ZeroMoney(p.currency), fmt.Errorf("%w: need %d, got %d", ErrPromoMinOrderNotMet, p.minOrderAmount, orderTotal.Amount())
	}

	var discount int64
	switch p.promoType {
	case PromoTypePercent:
		// Integer math: discount = (orderTotal.amount * value) / 100
		discount = (orderTotal.Amount() * p.value) / 100
	case PromoTypeFixed:
		discount = p.value
		if discount > orderTotal.Amount() {
			discount = orderTotal.Amount()
		}
	case PromoTypeFreeDelivery:
		// Discount = the delivery fee amount (so delivery becomes free).
		if deliveryFee.Currency() != p.currency {
			return ZeroMoney(p.currency), fmt.Errorf("%w: delivery fee currency %s vs %s", ErrCurrencyMismatch, deliveryFee.Currency(), p.currency)
		}
		discount = deliveryFee.Amount()
	}

	// Apply max cap.
	if p.maxDiscountAmount != nil && discount > *p.maxDiscountAmount {
		discount = *p.maxDiscountAmount
	}
	if discount < 0 {
		discount = 0
	}
	return Money{amount: discount, currency: p.currency}, nil
}

// IncrementUsage returns a copy with usageCount incremented.
// Does NOT persist — the repository handles persistence.
func (p Promotion) IncrementUsage() Promotion {
	p.usageCount++
	p.updatedAt = time.Now().UTC()
	return p
}

// SetActive toggles the active flag.
func (p Promotion) SetActive(active bool, now time.Time) Promotion {
	p.isActive = active
	p.updatedAt = now
	return p
}

// ===== Promotion Redemption Value Object =====

// PromotionRedemption records a single redemption event.
type PromotionRedemption struct {
	id             string
	promotionID    string
	userID         string
	orderID        string
	discountAmount Money
	currency       string
	redeemedAt     time.Time
}

// NewPromotionRedemption creates a new redemption record.
func NewPromotionRedemption(
	id, promotionID, userID, orderID string,
	discountAmount Money,
	now time.Time,
) (PromotionRedemption, error) {
	if id == "" {
		return PromotionRedemption{}, fmt.Errorf("%w: redemption id is required", ErrInvalidID)
	}
	if promotionID == "" {
		return PromotionRedemption{}, fmt.Errorf("%w: promotion id is required", ErrInvalidInput)
	}
	if userID == "" {
		return PromotionRedemption{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if !discountAmount.IsPositive() && discountAmount.Amount() != 0 {
		return PromotionRedemption{}, fmt.Errorf("%w: discount cannot be negative", ErrInvalidMoneyAmount)
	}
	return PromotionRedemption{
		id:             id,
		promotionID:    promotionID,
		userID:         userID,
		orderID:        orderID,
		discountAmount: discountAmount,
		currency:       discountAmount.Currency(),
		redeemedAt:     now,
	}, nil
}

// RehydratePromotionRedemption reconstructs from persistence.
func RehydratePromotionRedemption(
	id, promotionID, userID, orderID string,
	discountAmount Money,
	currency string,
	redeemedAt time.Time,
) PromotionRedemption {
	return PromotionRedemption{
		id:             id,
		promotionID:    promotionID,
		userID:         userID,
		orderID:        orderID,
		discountAmount: discountAmount,
		currency:       currency,
		redeemedAt:     redeemedAt,
	}
}

func (r PromotionRedemption) ID() string             { return r.id }
func (r PromotionRedemption) PromotionID() string    { return r.promotionID }
func (r PromotionRedemption) UserID() string         { return r.userID }
func (r PromotionRedemption) OrderID() string        { return r.orderID }
func (r PromotionRedemption) DiscountAmount() Money  { return r.discountAmount }
func (r PromotionRedemption) Currency() string       { return r.currency }
func (r PromotionRedemption) RedeemedAt() time.Time  { return r.redeemedAt }
