// Package domain payment_method: PaymentMethod value object.
//
// Supported methods: cash, card, wallet.
// The value object is immutable — construct via ParsePaymentMethod.
//
// Imports stdlib only.
package domain

import "fmt"

// PaymentMethod represents how the customer pays for the order.
type PaymentMethod string

const (
	PaymentCash   PaymentMethod = "cash"
	PaymentCard   PaymentMethod = "card"
	PaymentWallet PaymentMethod = "wallet"
)

// IsValid reports whether the payment method is recognized.
func (p PaymentMethod) IsValid() bool {
	switch p {
	case PaymentCash, PaymentCard, PaymentWallet:
		return true
	}
	return false
}

// String returns the string representation.
func (p PaymentMethod) String() string {
	return string(p)
}

// ParsePaymentMethod converts a string to a PaymentMethod.
// Returns an error if the string is not a valid payment method.
func ParsePaymentMethod(s string) (PaymentMethod, error) {
	pm := PaymentMethod(s)
	if !pm.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidPaymentMethod, s)
	}
	return pm, nil
}

// AllPaymentMethods returns all valid payment methods.
func AllPaymentMethods() []PaymentMethod {
	return []PaymentMethod{PaymentCash, PaymentCard, PaymentWallet}
}
