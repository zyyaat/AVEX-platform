// Package domain status: OrderStatus enum and state machine.
//
// The order lifecycle has 9 states:
//
//	pending → confirmed → preparing → ready_for_pickup → dispatching → assigned → picked_up → delivered
//	                                                                                           ↗
//	                                                 cancelled ←──────────────────────────────
//
// Terminal states: delivered, cancelled.
// No transitions are allowed from terminal states.
//
// Imports stdlib only.
package domain

import "fmt"

// OrderStatus represents the current lifecycle state of an order.
type OrderStatus string

const (
	StatusPending        OrderStatus = "pending"
	StatusConfirmed      OrderStatus = "confirmed"
	StatusPreparing      OrderStatus = "preparing"
	StatusReadyForPickup OrderStatus = "ready_for_pickup"
	StatusDispatching    OrderStatus = "dispatching"
	StatusAssigned       OrderStatus = "assigned"
	StatusPickedUp       OrderStatus = "picked_up"
	StatusDelivered      OrderStatus = "delivered"
	StatusCancelled      OrderStatus = "cancelled"
)

// IsValid reports whether the status is a recognized value.
func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusConfirmed, StatusPreparing, StatusReadyForPickup,
		StatusDispatching, StatusAssigned, StatusPickedUp, StatusDelivered, StatusCancelled:
		return true
	}
	return false
}

// String returns the string representation.
func (s OrderStatus) String() string {
	return string(s)
}

// IsTerminal reports whether the status is a terminal state (no further transitions).
func (s OrderStatus) IsTerminal() bool {
	return s == StatusDelivered || s == StatusCancelled
}

// IsDelivered reports whether the order has been delivered.
func (s OrderStatus) IsDelivered() bool {
	return s == StatusDelivered
}

// IsCancelled reports whether the order has been cancelled.
func (s OrderStatus) IsCancelled() bool {
	return s == StatusCancelled
}

// IsActive reports whether the order is in a non-terminal, non-cancelled state.
func (s OrderStatus) IsActive() bool {
	return !s.IsTerminal()
}

// CanTransitionTo reports whether transitioning from the current status
// to the target status is allowed by the state machine.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	if s == target {
		return false
	}
	if s.IsTerminal() {
		return false
	}

	allowed := map[OrderStatus][]OrderStatus{
		StatusPending:        {StatusConfirmed, StatusCancelled},
		StatusConfirmed:      {StatusPreparing, StatusCancelled},
		StatusPreparing:      {StatusReadyForPickup, StatusCancelled},
		StatusReadyForPickup: {StatusDispatching, StatusCancelled},
		StatusDispatching:    {StatusAssigned, StatusReadyForPickup, StatusCancelled},
		StatusAssigned:       {StatusPickedUp, StatusCancelled},
		StatusPickedUp:       {StatusDelivered, StatusCancelled},
		StatusDelivered:      {},
		StatusCancelled:      {},
	}

	for _, t := range allowed[s] {
		if t == target {
			return true
		}
	}
	return false
}

// Transition attempts to transition to the target status.
// Returns the new status on success, or ErrInvalidStatusTransition on failure.
func (s OrderStatus) Transition(target OrderStatus) (OrderStatus, error) {
	if !s.CanTransitionTo(target) {
		return s, fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, s, target)
	}
	return target, nil
}

// AllOrderStatuses returns all valid order statuses in lifecycle order.
func AllOrderStatuses() []OrderStatus {
	return []OrderStatus{
		StatusPending,
		StatusConfirmed,
		StatusPreparing,
		StatusReadyForPickup,
		StatusDispatching,
		StatusAssigned,
		StatusPickedUp,
		StatusDelivered,
		StatusCancelled,
	}
}

// Validate checks that the status is a recognized value.
// Returns an error if the status is invalid.
func (s OrderStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidInput, s)
	}
	return nil
}

// ParseOrderStatus converts a string to an OrderStatus.
// Returns an error if the string is not a valid status.
func ParseOrderStatus(s string) (OrderStatus, error) {
	status := OrderStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidInput, s)
	}
	return status, nil
}
