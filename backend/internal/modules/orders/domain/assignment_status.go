// Package domain assignment_status: AssignmentStatus enum for order assignments.
//
// An OrderAssignment tracks a single offer of an order to a driver.
// The assignment lifecycle:
//
//	pending → accepted   (driver accepts the offer)
//	pending → rejected   (driver explicitly rejects)
//	pending → expired    (driver doesn't respond before offer_expires_at)
//	pending → cancelled  (system cancels: order cancelled or reassigned)
//
// Terminal states: accepted, rejected, expired, cancelled.
//
// Imports stdlib only.
package domain

import "fmt"

// AssignmentStatus represents the status of a driver assignment offer.
type AssignmentStatus string

const (
	AssignmentPending   AssignmentStatus = "pending"
	AssignmentAccepted  AssignmentStatus = "accepted"
	AssignmentRejected  AssignmentStatus = "rejected"
	AssignmentExpired   AssignmentStatus = "expired"
	AssignmentCancelled AssignmentStatus = "cancelled"
)

// IsValid reports whether the status is a recognized value.
func (s AssignmentStatus) IsValid() bool {
	switch s {
	case AssignmentPending, AssignmentAccepted, AssignmentRejected, AssignmentExpired, AssignmentCancelled:
		return true
	}
	return false
}

// String returns the string representation.
func (s AssignmentStatus) String() string {
	return string(s)
}

// IsTerminal reports whether the status is terminal (no further transitions).
func (s AssignmentStatus) IsTerminal() bool {
	return s != AssignmentPending
}

// IsPending reports whether the assignment is still pending a driver response.
func (s AssignmentStatus) IsPending() bool {
	return s == AssignmentPending
}

// IsAccepted reports whether the driver accepted the assignment.
func (s AssignmentStatus) IsAccepted() bool {
	return s == AssignmentAccepted
}

// CanTransitionTo reports whether transitioning from the current status
// to the target status is allowed.
func (s AssignmentStatus) CanTransitionTo(target AssignmentStatus) bool {
	if s == target {
		return false
	}
	if s.IsTerminal() {
		return false
	}

	// Only pending can transition, and only to terminal states.
	allowed := map[AssignmentStatus][]AssignmentStatus{
		AssignmentPending: {AssignmentAccepted, AssignmentRejected, AssignmentExpired, AssignmentCancelled},
	}

	for _, t := range allowed[s] {
		if t == target {
			return true
		}
	}
	return false
}

// Transition attempts to transition to the target status.
// Returns the new status on success, or ErrInvalidAssignmentTransition on failure.
func (s AssignmentStatus) Transition(target AssignmentStatus) (AssignmentStatus, error) {
	if !s.CanTransitionTo(target) {
		return s, fmt.Errorf("%w: %s -> %s", ErrInvalidAssignmentTransition, s, target)
	}
	return target, nil
}

// AllAssignmentStatuses returns all valid assignment statuses.
func AllAssignmentStatuses() []AssignmentStatus {
	return []AssignmentStatus{
		AssignmentPending,
		AssignmentAccepted,
		AssignmentRejected,
		AssignmentExpired,
		AssignmentCancelled,
	}
}

// ParseAssignmentStatus converts a string to an AssignmentStatus.
// Returns an error if the string is not a valid status.
func ParseAssignmentStatus(s string) (AssignmentStatus, error) {
	status := AssignmentStatus(s)
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidInput, s)
	}
	return status, nil
}
