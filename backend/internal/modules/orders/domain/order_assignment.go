// Package domain order_assignment: OrderAssignment entity.
//
// An OrderAssignment tracks a single offer of an order to a driver.
// The dispatch engine creates an assignment when it sends an offer to a driver.
// The driver can accept, reject, or let the offer expire.
//
// Key features:
//   - offer_expires_at: deadline for driver response (e.g. 15 seconds)
//   - attempt_number: tracks which attempt this is (1, 2, 3...) for analytics
//   - State machine: pending → accepted/rejected/expired/cancelled
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// OrderAssignment tracks a single dispatch offer to a driver.
type OrderAssignment struct {
	id             string
	orderID        string
	driverID       string
	status         AssignmentStatus
	assignedAt     time.Time
	offerExpiresAt time.Time
	respondedAt    *time.Time
	acceptedAt     *time.Time
	rejectedAt     *time.Time
	expiredAt      *time.Time
	rejectedReason string
	distanceM      *int
	attemptNumber  int
}

// AssignmentParams holds the parameters for creating a new assignment.
type AssignmentParams struct {
	ID            string
	OrderID       string
	DriverID      string
	OfferTTL      time.Duration // how long the driver has to respond
	DistanceM     *int
	AttemptNumber int
	Now           time.Time
}

// NewOrderAssignment creates a new pending assignment with validation.
func NewOrderAssignment(params AssignmentParams) (OrderAssignment, error) {
	if params.ID == "" {
		return OrderAssignment{}, NewValidationError("id", ErrInvalidID)
	}
	if params.OrderID == "" {
		return OrderAssignment{}, NewValidationError("order_id", ErrInvalidInput)
	}
	if params.DriverID == "" {
		return OrderAssignment{}, NewValidationError("driver_id", ErrInvalidInput)
	}
	if params.OfferTTL <= 0 {
		return OrderAssignment{}, NewValidationError("offer_ttl", ErrInvalidInput)
	}
	if params.AttemptNumber < 1 {
		params.AttemptNumber = 1
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return OrderAssignment{
		id:             params.ID,
		orderID:        params.OrderID,
		driverID:       params.DriverID,
		status:         AssignmentPending,
		assignedAt:     now,
		offerExpiresAt: now.Add(params.OfferTTL),
		distanceM:      params.DistanceM,
		attemptNumber:  params.AttemptNumber,
	}, nil
}

// ===== Reconstruction =====

// AssignmentRecord holds all fields to rebuild an OrderAssignment from persistence.
type AssignmentRecord struct {
	ID             string
	OrderID        string
	DriverID       string
	Status         AssignmentStatus
	AssignedAt     time.Time
	OfferExpiresAt time.Time
	RespondedAt    *time.Time
	AcceptedAt     *time.Time
	RejectedAt     *time.Time
	ExpiredAt      *time.Time
	RejectedReason string
	DistanceM      *int
	AttemptNumber  int
}

// ReconstructAssignment rebuilds an OrderAssignment from persistence (no validation).
func ReconstructAssignment(rec AssignmentRecord) OrderAssignment {
	return OrderAssignment{
		id:             rec.ID,
		orderID:        rec.OrderID,
		driverID:       rec.DriverID,
		status:         rec.Status,
		assignedAt:     rec.AssignedAt,
		offerExpiresAt: rec.OfferExpiresAt,
		respondedAt:    rec.RespondedAt,
		acceptedAt:     rec.AcceptedAt,
		rejectedAt:     rec.RejectedAt,
		expiredAt:      rec.ExpiredAt,
		rejectedReason: rec.RejectedReason,
		distanceM:      rec.DistanceM,
		attemptNumber:  rec.AttemptNumber,
	}
}

// ===== Getters =====

func (a OrderAssignment) ID() string                { return a.id }
func (a OrderAssignment) OrderID() string           { return a.orderID }
func (a OrderAssignment) DriverID() string          { return a.driverID }
func (a OrderAssignment) Status() AssignmentStatus  { return a.status }
func (a OrderAssignment) AssignedAt() time.Time     { return a.assignedAt }
func (a OrderAssignment) OfferExpiresAt() time.Time { return a.offerExpiresAt }
func (a OrderAssignment) RespondedAt() *time.Time   { return a.respondedAt }
func (a OrderAssignment) AcceptedAt() *time.Time    { return a.acceptedAt }
func (a OrderAssignment) RejectedAt() *time.Time    { return a.rejectedAt }
func (a OrderAssignment) ExpiredAt() *time.Time     { return a.expiredAt }
func (a OrderAssignment) RejectedReason() string    { return a.rejectedReason }
func (a OrderAssignment) DistanceM() *int           { return a.distanceM }
func (a OrderAssignment) AttemptNumber() int        { return a.attemptNumber }

// IsOfferExpired reports whether the offer deadline has passed.
func (a OrderAssignment) IsOfferExpired(now time.Time) bool {
	return a.status == AssignmentPending && now.After(a.offerExpiresAt)
}

// IsPending reports whether the assignment is still awaiting a response.
func (a OrderAssignment) IsPending() bool {
	return a.status == AssignmentPending
}

// IsAccepted reports whether the driver accepted the assignment.
func (a OrderAssignment) IsAccepted() bool {
	return a.status == AssignmentAccepted
}

// ===== Behavior (mutations) =====

// Accept marks the assignment as accepted by the driver.
// Returns an error if the assignment is not pending or the offer has expired.
func (a *OrderAssignment) Accept(now time.Time) error {
	if a.status != AssignmentPending {
		if a.status == AssignmentAccepted {
			return ErrAssignmentAlreadyAccepted
		}
		return fmt.Errorf("%w: cannot accept from %s", ErrInvalidAssignmentTransition, a.status)
	}
	if now.After(a.offerExpiresAt) {
		return ErrAssignmentOfferExpired
	}

	a.status = AssignmentAccepted
	a.respondedAt = &now
	a.acceptedAt = &now
	return nil
}

// Reject marks the assignment as rejected by the driver.
// reason is optional (empty string = no reason given).
func (a *OrderAssignment) Reject(reason string, now time.Time) error {
	if a.status != AssignmentPending {
		if a.status == AssignmentRejected {
			return ErrAssignmentAlreadyRejected
		}
		return fmt.Errorf("%w: cannot reject from %s", ErrInvalidAssignmentTransition, a.status)
	}

	a.status = AssignmentRejected
	a.respondedAt = &now
	a.rejectedAt = &now
	a.rejectedReason = reason
	return nil
}

// Expire marks the assignment as expired (driver didn't respond in time).
// Returns an error if the assignment is not pending.
func (a *OrderAssignment) Expire(now time.Time) error {
	if a.status != AssignmentPending {
		if a.status == AssignmentExpired {
			return ErrAssignmentAlreadyExpired
		}
		return fmt.Errorf("%w: cannot expire from %s", ErrInvalidAssignmentTransition, a.status)
	}

	a.status = AssignmentExpired
	a.expiredAt = &now
	return nil
}

// Cancel marks the assignment as cancelled by the system.
// Used when an order is cancelled or reassigned to a different driver.
func (a *OrderAssignment) Cancel(now time.Time) error {
	if a.status != AssignmentPending {
		if a.status == AssignmentCancelled {
			return ErrAssignmentAlreadyCancelled
		}
		return fmt.Errorf("%w: cannot cancel from %s", ErrInvalidAssignmentTransition, a.status)
	}

	a.status = AssignmentCancelled
	return nil
}

// ===== String =====

func (a OrderAssignment) String() string {
	return fmt.Sprintf("OrderAssignment{id=%s, order=%s, driver=%s, status=%s, attempt=%d}",
		a.id, a.orderID, a.driverID, a.status, a.attemptNumber)
}
