// Package domain tests: OrderStatus state machine.
package domain

import (
	"errors"
	"testing"
)

func TestOrderStatus_IsValid(t *testing.T) {
	valid := AllOrderStatuses()
	if len(valid) != 9 {
		t.Errorf("expected 9 statuses, got %d", len(valid))
	}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("%q should be valid", s)
		}
	}
	invalid := []OrderStatus{"", "unknown", "PENDING", "delivered ", "active"}
	for _, s := range invalid {
		if s.IsValid() {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestOrderStatus_IsTerminal(t *testing.T) {
	terminal := []OrderStatus{StatusDelivered, StatusCancelled}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%q should be terminal", s)
		}
	}
	nonTerminal := []OrderStatus{
		StatusPending, StatusConfirmed, StatusPreparing, StatusReadyForPickup,
		StatusDispatching, StatusAssigned, StatusPickedUp,
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%q should not be terminal", s)
		}
	}
}

func TestOrderStatus_CanTransitionTo_AllowedTransitions(t *testing.T) {
	tests := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		{StatusPending, StatusConfirmed},
		{StatusPending, StatusCancelled},
		{StatusConfirmed, StatusPreparing},
		{StatusConfirmed, StatusCancelled},
		{StatusPreparing, StatusReadyForPickup},
		{StatusPreparing, StatusCancelled},
		{StatusReadyForPickup, StatusDispatching},
		{StatusReadyForPickup, StatusCancelled},
		{StatusDispatching, StatusAssigned},
		{StatusDispatching, StatusReadyForPickup}, // retry
		{StatusDispatching, StatusCancelled},
		{StatusAssigned, StatusPickedUp},
		{StatusAssigned, StatusCancelled},
		{StatusPickedUp, StatusDelivered},
		{StatusPickedUp, StatusCancelled},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if !tt.from.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%q -> %q) = false, want true", tt.from, tt.to)
			}
		})
	}
}

func TestOrderStatus_CanTransitionTo_ForbiddenTransitions(t *testing.T) {
	tests := []struct {
		from OrderStatus
		to   OrderStatus
	}{
		// No skipping states
		{StatusPending, StatusPreparing},
		{StatusPending, StatusDelivered},
		{StatusConfirmed, StatusReadyForPickup},
		{StatusConfirmed, StatusDelivered},
		{StatusPreparing, StatusPickedUp},
		// Reverse transitions
		{StatusConfirmed, StatusPending},
		{StatusPreparing, StatusConfirmed},
		{StatusReadyForPickup, StatusPreparing},
		{StatusAssigned, StatusDispatching},
		{StatusPickedUp, StatusAssigned},
		// Terminal states
		{StatusDelivered, StatusCancelled},
		{StatusDelivered, StatusPending},
		{StatusCancelled, StatusDelivered},
		{StatusCancelled, StatusPending},
		// No-op (same state)
		{StatusPending, StatusPending},
		{StatusDelivered, StatusDelivered},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if tt.from.CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%q -> %q) = true, want false", tt.from, tt.to)
			}
		})
	}
}

func TestOrderStatus_Transition_Success(t *testing.T) {
	newStatus, err := StatusPending.Transition(StatusConfirmed)
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}
	if newStatus != StatusConfirmed {
		t.Errorf("Transition result = %q, want %q", newStatus, StatusConfirmed)
	}
}

func TestOrderStatus_Transition_Forbidden(t *testing.T) {
	_, err := StatusPending.Transition(StatusDelivered)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("Transition error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestOrderStatus_Transition_FromTerminal(t *testing.T) {
	_, err := StatusDelivered.Transition(StatusCancelled)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("Transition from terminal error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestParseOrderStatus(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"pending", true},
		{"confirmed", true},
		{"preparing", true},
		{"ready_for_pickup", true},
		{"dispatching", true},
		{"assigned", true},
		{"picked_up", true},
		{"delivered", true},
		{"cancelled", true},
		{"", false},
		{"unknown", false},
		{"PENDING", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseOrderStatus(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ParseOrderStatus(%q) error: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ParseOrderStatus(%q) should fail", tt.input)
			}
		})
	}
}
