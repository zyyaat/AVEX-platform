// Package domain tests: OrderAssignment entity — transitions, expiry, attempt tracking.
package domain

import (
	"errors"
	"testing"
	"time"
)

var assignmentNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func validAssignmentParams() AssignmentParams {
	return AssignmentParams{
		ID:            "assign-001",
		OrderID:       "order-001",
		DriverID:      "driver-001",
		OfferTTL:      15 * time.Second,
		DistanceM:     intPtr(500),
		AttemptNumber: 1,
		Now:           assignmentNow,
	}
}

func intPtr(v int) *int { return &v }

func TestNewOrderAssignment_Success(t *testing.T) {
	a, err := NewOrderAssignment(validAssignmentParams())
	if err != nil {
		t.Fatalf("NewOrderAssignment error: %v", err)
	}
	if a.Status() != AssignmentPending {
		t.Errorf("Status = %q, want 'pending'", a.Status())
	}
	if a.AttemptNumber() != 1 {
		t.Errorf("AttemptNumber = %d, want 1", a.AttemptNumber())
	}
	if !a.OfferExpiresAt().Equal(assignmentNow.Add(15 * time.Second)) {
		t.Errorf("OfferExpiresAt = %v, want %v", a.OfferExpiresAt(), assignmentNow.Add(15*time.Second))
	}
	if a.IsPending() {
		// IsPending checks status, not time
	}
	if !a.IsPending() {
		t.Error("new assignment should be pending")
	}
}

func TestNewOrderAssignment_DefaultAttemptNumber(t *testing.T) {
	params := validAssignmentParams()
	params.AttemptNumber = 0
	a, err := NewOrderAssignment(params)
	if err != nil {
		t.Fatalf("NewOrderAssignment error: %v", err)
	}
	if a.AttemptNumber() != 1 {
		t.Errorf("AttemptNumber = %d, want 1 (default)", a.AttemptNumber())
	}
}

func TestNewOrderAssignment_EmptyID(t *testing.T) {
	params := validAssignmentParams()
	params.ID = ""
	_, err := NewOrderAssignment(params)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("error = %v, want ErrInvalidID", err)
	}
}

func TestNewOrderAssignment_EmptyOrderID(t *testing.T) {
	params := validAssignmentParams()
	params.OrderID = ""
	_, err := NewOrderAssignment(params)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

func TestNewOrderAssignment_EmptyDriverID(t *testing.T) {
	params := validAssignmentParams()
	params.DriverID = ""
	_, err := NewOrderAssignment(params)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

func TestNewOrderAssignment_ZeroOfferTTL(t *testing.T) {
	params := validAssignmentParams()
	params.OfferTTL = 0
	_, err := NewOrderAssignment(params)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

func TestOrderAssignment_Accept_Success(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	err := a.Accept(assignmentNow.Add(5 * time.Second))
	if err != nil {
		t.Fatalf("Accept error: %v", err)
	}
	if !a.IsAccepted() {
		t.Error("assignment should be accepted")
	}
	if a.AcceptedAt() == nil {
		t.Error("AcceptedAt should be set")
	}
	if a.RespondedAt() == nil {
		t.Error("RespondedAt should be set")
	}
}

func TestOrderAssignment_Accept_OfferExpired(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	err := a.Accept(assignmentNow.Add(20 * time.Second)) // 15s TTL exceeded
	if !errors.Is(err, ErrAssignmentOfferExpired) {
		t.Errorf("error = %v, want ErrAssignmentOfferExpired", err)
	}
}

func TestOrderAssignment_Accept_AlreadyAccepted(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Accept(assignmentNow.Add(5 * time.Second))
	err := a.Accept(assignmentNow.Add(6 * time.Second))
	if !errors.Is(err, ErrAssignmentAlreadyAccepted) {
		t.Errorf("error = %v, want ErrAssignmentAlreadyAccepted", err)
	}
}

func TestOrderAssignment_Reject_Success(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	err := a.Reject("too far", assignmentNow.Add(5*time.Second))
	if err != nil {
		t.Fatalf("Reject error: %v", err)
	}
	if a.Status() != AssignmentRejected {
		t.Errorf("Status = %q, want 'rejected'", a.Status())
	}
	if a.RejectedReason() != "too far" {
		t.Errorf("RejectedReason = %q", a.RejectedReason())
	}
	if a.RejectedAt() == nil {
		t.Error("RejectedAt should be set")
	}
}

func TestOrderAssignment_Reject_AlreadyRejected(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Reject("too far", assignmentNow.Add(5*time.Second))
	err := a.Reject("again", assignmentNow.Add(6*time.Second))
	if !errors.Is(err, ErrAssignmentAlreadyRejected) {
		t.Errorf("error = %v, want ErrAssignmentAlreadyRejected", err)
	}
}

func TestOrderAssignment_Expire_Success(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	err := a.Expire(assignmentNow.Add(20 * time.Second))
	if err != nil {
		t.Fatalf("Expire error: %v", err)
	}
	if a.Status() != AssignmentExpired {
		t.Errorf("Status = %q, want 'expired'", a.Status())
	}
	if a.ExpiredAt() == nil {
		t.Error("ExpiredAt should be set")
	}
}

func TestOrderAssignment_Expire_AlreadyExpired(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Expire(assignmentNow.Add(20 * time.Second))
	err := a.Expire(assignmentNow.Add(25 * time.Second))
	if !errors.Is(err, ErrAssignmentAlreadyExpired) {
		t.Errorf("error = %v, want ErrAssignmentAlreadyExpired", err)
	}
}

func TestOrderAssignment_Cancel_Success(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	err := a.Cancel(assignmentNow.Add(5 * time.Second))
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if a.Status() != AssignmentCancelled {
		t.Errorf("Status = %q, want 'cancelled'", a.Status())
	}
}

func TestOrderAssignment_Cancel_AlreadyCancelled(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Cancel(assignmentNow.Add(5 * time.Second))
	err := a.Cancel(assignmentNow.Add(6 * time.Second))
	if !errors.Is(err, ErrAssignmentAlreadyCancelled) {
		t.Errorf("error = %v, want ErrAssignmentAlreadyCancelled", err)
	}
}

func TestOrderAssignment_Accept_FromRejected(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Reject("no", assignmentNow.Add(5*time.Second))
	err := a.Accept(assignmentNow.Add(6 * time.Second))
	if !errors.Is(err, ErrInvalidAssignmentTransition) {
		t.Errorf("error = %v, want ErrInvalidAssignmentTransition", err)
	}
}

func TestOrderAssignment_IsOfferExpired(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())

	// Before expiry.
	if a.IsOfferExpired(assignmentNow.Add(10 * time.Second)) {
		t.Error("offer should not be expired at 10s (TTL=15s)")
	}

	// After expiry.
	if !a.IsOfferExpired(assignmentNow.Add(20 * time.Second)) {
		t.Error("offer should be expired at 20s (TTL=15s)")
	}
}

func TestOrderAssignment_IsOfferExpired_AfterAccepted(t *testing.T) {
	a, _ := NewOrderAssignment(validAssignmentParams())
	_ = a.Accept(assignmentNow.Add(5 * time.Second))

	// After acceptance, IsOfferExpired should return false even past deadline.
	if a.IsOfferExpired(assignmentNow.Add(20 * time.Second)) {
		t.Error("accepted assignment should not report offer expired")
	}
}

func TestAssignmentStatus_IsTerminal(t *testing.T) {
	terminal := []AssignmentStatus{AssignmentAccepted, AssignmentRejected, AssignmentExpired, AssignmentCancelled}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%q should be terminal", s)
		}
	}
	if AssignmentPending.IsTerminal() {
		t.Error("pending should not be terminal")
	}
}

func TestAssignmentStatus_CanTransitionTo(t *testing.T) {
	// Pending can transition to all terminal states.
	targets := []AssignmentStatus{AssignmentAccepted, AssignmentRejected, AssignmentExpired, AssignmentCancelled}
	for _, target := range targets {
		if !AssignmentPending.CanTransitionTo(target) {
			t.Errorf("pending should transition to %q", target)
		}
	}

	// Terminal states cannot transition.
	for _, from := range targets {
		for _, to := range targets {
			if from.CanTransitionTo(to) {
				t.Errorf("%q should not transition to %q (terminal)", from, to)
			}
		}
	}

	// No-op transitions not allowed.
	if AssignmentPending.CanTransitionTo(AssignmentPending) {
		t.Error("pending should not transition to pending (no-op)")
	}
}
