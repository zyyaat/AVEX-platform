// Package domain contains pure domain entities for the financial module.
//
// This file: typed domain errors for wallets, transactions, promotions,
// and pricing. All errors are sentinel-style vars that callers can match
// with errors.Is().
//
// Imports stdlib only — domain has zero external dependencies.
package domain

import (
	"errors"
	"fmt"
)

// ===== Wallet Errors =====

var ErrWalletNotFound = errors.New("wallet not found")
var ErrWalletAlreadyExists = errors.New("wallet already exists")
var ErrWalletFrozen = errors.New("wallet is frozen")
var ErrWalletClosed = errors.New("wallet is closed")
var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrInvalidOwnerType = errors.New("invalid owner type (must be user, driver, or merchant)")
var ErrOwnerIDRequired = errors.New("owner id is required")
var ErrInvalidCurrency = errors.New("invalid currency code (must be 3 letters)")

// ===== Transaction Errors =====

var ErrTransactionNotFound = errors.New("transaction not found")
var ErrTransactionAlreadyExists = errors.New("transaction already exists")
var ErrInvalidTransactionType = errors.New("invalid transaction type (must be credit or debit)")
var ErrInvalidTransactionCategory = errors.New("invalid transaction category")
var ErrTransactionAlreadyCompleted = errors.New("transaction is already completed")
var ErrTransactionAlreadyFailed = errors.New("transaction is already failed")
var ErrTransactionCannotBeReversed = errors.New("transaction cannot be reversed in its current state")
var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

// ===== Promotion Errors =====

var ErrPromotionNotFound = errors.New("promotion not found")
var ErrPromotionCodeAlreadyExists = errors.New("promotion code already exists")
var ErrPromotionInactive = errors.New("promotion is inactive")
var ErrPromotionExpired = errors.New("promotion has expired")
var ErrPromotionNotYetValid = errors.New("promotion is not yet valid")
var ErrPromotionUsageLimitReached = errors.New("promotion usage limit reached")
var ErrPromotionPerUserLimitReached = errors.New("promotion per-user limit reached")
var ErrPromoMinOrderNotMet = errors.New("order amount does not meet promotion minimum")
var ErrPromoAlreadyRedeemed = errors.New("promotion already redeemed for this order")
var ErrInvalidPromoType = errors.New("invalid promotion type")
var ErrInvalidDiscountValue = errors.New("invalid discount value")

// ===== Pricing Errors =====

var ErrPricingRuleNotFound = errors.New("pricing rule not found")
var ErrPricingRuleAlreadyExists = errors.New("pricing rule already exists for this zone")
var ErrSurgeMultiplierInvalid = errors.New("surge multiplier must be >= 1.0")
var ErrInvalidDistance = errors.New("distance must be >= 0")
var ErrInvalidDuration = errors.New("duration must be >= 0")

// ===== Generic Validation =====

var ErrInvalidMoneyAmount = errors.New("invalid money amount (must be >= 0)")
var ErrNegativeMoneyResult = errors.New("money operation resulted in a negative amount")
var ErrCurrencyMismatch = errors.New("currency mismatch between money values")
var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrInvalidPercentage = errors.New("percentage must be between 0 and 100")

// ===== Composite Error =====

// ValidationError wraps a domain error with field-level context.
type ValidationError struct {
	Field   string
	Wrapped error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %v", e.Field, e.Wrapped)
	}
	return e.Wrapped.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Wrapped
}

func NewValidationError(field string, err error) *ValidationError {
	return &ValidationError{Field: field, Wrapped: err}
}
