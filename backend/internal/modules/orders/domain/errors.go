// Package domain contains pure domain entities for the orders module.
// This file: typed domain errors for order lifecycle, validation, and assignments.
//
// Imports stdlib only — domain has zero external dependencies.
package domain

import (
	"errors"
	"fmt"
)

// ===== Order Not Found / Already Exists =====

var ErrOrderNotFound = errors.New("order not found")
var ErrOrderAlreadyExists = errors.New("order already exists")

// ===== Order Status Errors =====

var ErrInvalidStatusTransition = errors.New("invalid order status transition")
var ErrOrderNotPending = errors.New("order is not pending")
var ErrOrderNotConfirmed = errors.New("order is not confirmed")
var ErrOrderNotPreparing = errors.New("order is not preparing")
var ErrOrderNotReadyForPickup = errors.New("order is not ready for pickup")
var ErrOrderNotDispatching = errors.New("order is not dispatching")
var ErrOrderNotAssigned = errors.New("order is not assigned")
var ErrOrderNotPickedUp = errors.New("order is not picked up")
var ErrOrderAlreadyDelivered = errors.New("order is already delivered")
var ErrOrderAlreadyCancelled = errors.New("order is already cancelled")
var ErrOrderIsTerminal = errors.New("order is in a terminal state (delivered or cancelled)")

// ===== Cancellation Errors =====

var ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current state")
var ErrCancelReasonRequired = errors.New("cancel reason is required")

// ===== Assignment Errors =====

var ErrAssignmentNotFound = errors.New("order assignment not found")
var ErrAssignmentAlreadyAccepted = errors.New("assignment already accepted")
var ErrAssignmentAlreadyRejected = errors.New("assignment already rejected")
var ErrAssignmentAlreadyExpired = errors.New("assignment already expired")
var ErrAssignmentAlreadyCancelled = errors.New("assignment already cancelled")
var ErrAssignmentOfferExpired = errors.New("assignment offer has expired")
var ErrInvalidAssignmentTransition = errors.New("invalid assignment status transition")
var ErrDriverNotAssigned = errors.New("no driver assigned to this order")

// ===== Validation Errors =====

var ErrInvalidOrderNumber = errors.New("invalid order number")
var ErrEmptyOrderItems = errors.New("order must have at least one item")
var ErrInvalidQuantity = errors.New("quantity must be greater than zero")
var ErrInvalidPaymentMethod = errors.New("invalid payment method")
var ErrInvalidMoneyAmount = errors.New("invalid money amount")
var ErrInvalidCurrency = errors.New("invalid currency code")
var ErrCurrencyMismatch = errors.New("currency mismatch between money values")
var ErrNegativeMoneyResult = errors.New("money operation resulted in a negative amount")
var ErrDeliveryInfoRequired = errors.New("delivery info is required")
var ErrInvalidLatitude = errors.New("invalid latitude (must be between -90 and 90)")
var ErrInvalidLongitude = errors.New("invalid longitude (must be between -180 and 180)")
var ErrDeliveryAddressRequired = errors.New("delivery address is required")
var ErrRestaurantIDRequired = errors.New("restaurant id is required")
var ErrUserIDRequired = errors.New("user id is required")
var ErrCustomerNameRequired = errors.New("customer name is required")
var ErrCustomerPhoneRequired = errors.New("customer phone is required")
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
