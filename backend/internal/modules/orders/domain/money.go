// Package domain money: Money value object with integer arithmetic.
//
// Money is stored as integer cents (int64) to avoid floating-point errors.
// All arithmetic operations validate currency matching and non-negative results.
// Percentage calculations use integer math: result = (base.amount * percent) / 100.
//
// The value object is immutable — all operations return new Money instances.
//
// Imports stdlib only.
package domain

import "fmt"

// Money represents a monetary amount in a specific currency.
// Amount is stored in cents (smallest currency unit) to avoid float errors.
// For example, $12.99 is stored as 1299 cents.
type Money struct {
	amount   int64
	currency string // ISO 4217 code, e.g. "EGP", "USD"
}

// NewMoney creates a Money value with validation.
// amount is in cents (must be >= 0).
// currency must be a non-empty 3-letter code.
func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, ErrInvalidMoneyAmount
	}
	if len(currency) != 3 {
		return Money{}, fmt.Errorf("%w: currency must be a 3-letter code, got %q", ErrInvalidCurrency, currency)
	}
	return Money{amount: amount, currency: currency}, nil
}

// ZeroMoney returns a zero-amount Money in the given currency.
func ZeroMoney(currency string) Money {
	return Money{amount: 0, currency: currency}
}

// Amount returns the amount in cents.
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO currency code.
func (m Money) Currency() string {
	return m.currency
}

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive reports whether the amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// Equals reports whether two Money values have the same amount and currency.
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

// LessThan reports whether m is less than other.
// Returns an error if currencies differ.
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return m.amount < other.amount, nil
}

// GreaterThanOrEqual reports whether m >= other.
// Returns an error if currencies differ.
func (m Money) GreaterThanOrEqual(other Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return m.amount >= other.amount, nil
}

// Add returns a new Money that is the sum of m and other.
// Returns an error if currencies differ.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// Subtract returns a new Money that is m - other.
// Returns an error if currencies differ or if the result would be negative.
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	result := m.amount - other.amount
	if result < 0 {
		return Money{}, fmt.Errorf("%w: %d - %d", ErrNegativeMoneyResult, m.amount, other.amount)
	}
	return Money{amount: result, currency: m.currency}, nil
}

// Multiply returns a new Money with the amount multiplied by qty.
// qty must be positive.
func (m Money) Multiply(qty int) (Money, error) {
	if qty <= 0 {
		return Money{}, fmt.Errorf("%w: quantity must be positive, got %d", ErrInvalidInput, qty)
	}
	return Money{amount: m.amount * int64(qty), currency: m.currency}, nil
}

// Percentage returns a new Money that is percent% of m.
// Uses integer math: result = (m.amount * percent) / 100.
// percent must be between 0 and 100 inclusive.
//
// Example: 1000 cents * 15% = 150 cents
func Percentage(base Money, percent int) (Money, error) {
	if percent < 0 || percent > 100 {
		return Money{}, fmt.Errorf("%w: %d", ErrInvalidPercentage, percent)
	}
	result := (base.amount * int64(percent)) / 100
	return Money{amount: result, currency: base.currency}, nil
}

// String returns a human-readable representation for logging.
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
