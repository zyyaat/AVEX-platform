// Package domain dispatch_offer: DispatchOffer aggregate root.
//
// A DispatchOffer represents a single dispatch attempt to assign an order
// to a driver. The dispatch engine creates an offer when it receives an
// OrderAssignmentRequested event from the orders module.
//
// Lifecycle:
//   pending → accepted → (order assigned in orders module)
//   pending → rejected (driver declined)
//   pending → expired (driver did not respond within TTL)
//   pending → cancelled (system cancelled, e.g. order cancelled)
//
// Multiple offers may be created for the same order (one per attempt), but
// only one offer can be in 'pending' state at a time for a given order.
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// OfferStatus enumerates dispatch offer lifecycle states.
type OfferStatus string

const (
	OfferStatusPending   OfferStatus = "pending"
	OfferStatusAccepted  OfferStatus = "accepted"
	OfferStatusRejected  OfferStatus = "rejected"
	OfferStatusExpired   OfferStatus = "expired"
	OfferStatusCancelled OfferStatus = "cancelled"
)

// DispatchOffer is the aggregate root for a single dispatch attempt.
type DispatchOffer struct {
	id            string
	orderID       string
	driverID      string
	zoneID        string
	status        OfferStatus
	pickupLat     float64
	pickupLng     float64
	deliveryLat   float64
	deliveryLng   float64
	estDistanceM  *int    // estimated distance driver → pickup
	estDurationS  *int    // estimated duration driver → pickup (seconds)
	estFareCents  *int64  // estimated fare for the driver (cents)
	currency      string
	offerTTL      time.Duration
	offeredAt     time.Time
	expiresAt     time.Time
	respondedAt   *time.Time
	acceptedAt    *time.Time
	rejectedAt    *time.Time
	expiredAt     *time.Time
	cancelledAt   *time.Time
	rejectReason  string
	attemptNumber int // 1-based; increments on retry
	createdBy     string // system | manual
}

// NewDispatchOffer creates a new pending offer with validation.
func NewDispatchOffer(
	id, orderID, driverID, zoneID string,
	pickupLat, pickupLng, deliveryLat, deliveryLng float64,
	estDistanceM *int, estDurationS *int, estFareCents *int64, currency string,
	offerTTL time.Duration,
	attemptNumber int,
	createdBy string,
	now time.Time,
) (DispatchOffer, error) {
	if id == "" {
		return DispatchOffer{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if orderID == "" {
		return DispatchOffer{}, fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}
	if driverID == "" {
		return DispatchOffer{}, fmt.Errorf("%w: driver id is required", ErrInvalidInput)
	}
	if pickupLat < -90 || pickupLat > 90 {
		return DispatchOffer{}, ErrInvalidLatitude
	}
	if pickupLng < -180 || pickupLng > 180 {
		return DispatchOffer{}, ErrInvalidLongitude
	}
	if deliveryLat < -90 || deliveryLat > 90 {
		return DispatchOffer{}, ErrInvalidLatitude
	}
	if deliveryLng < -180 || deliveryLng > 180 {
		return DispatchOffer{}, ErrInvalidLongitude
	}
	if offerTTL <= 0 {
		return DispatchOffer{}, fmt.Errorf("%w: offer TTL must be positive", ErrInvalidInput)
	}
	if attemptNumber < 1 {
		attemptNumber = 1
	}
	if createdBy == "" {
		createdBy = "system"
	}
	return DispatchOffer{
		id:            id,
		orderID:       orderID,
		driverID:      driverID,
		zoneID:        zoneID,
		status:        OfferStatusPending,
		pickupLat:     pickupLat,
		pickupLng:     pickupLng,
		deliveryLat:   deliveryLat,
		deliveryLng:   deliveryLng,
		estDistanceM:  estDistanceM,
		estDurationS:  estDurationS,
		estFareCents:  estFareCents,
		currency:      currency,
		offerTTL:      offerTTL,
		offeredAt:     now,
		expiresAt:     now.Add(offerTTL),
		attemptNumber: attemptNumber,
		createdBy:     createdBy,
	}, nil
}

// RehydrateDispatchOffer reconstructs from persistence.
func RehydrateDispatchOffer(
	id, orderID, driverID, zoneID string,
	status OfferStatus,
	pickupLat, pickupLng, deliveryLat, deliveryLng float64,
	estDistanceM *int, estDurationS *int, estFareCents *int64, currency string,
	offerTTL time.Duration,
	offeredAt, expiresAt time.Time,
	respondedAt, acceptedAt, rejectedAt, expiredAt, cancelledAt *time.Time,
	rejectReason string,
	attemptNumber int,
	createdBy string,
) DispatchOffer {
	return DispatchOffer{
		id:            id,
		orderID:       orderID,
		driverID:      driverID,
		zoneID:        zoneID,
		status:        status,
		pickupLat:     pickupLat,
		pickupLng:     pickupLng,
		deliveryLat:   deliveryLat,
		deliveryLng:   deliveryLng,
		estDistanceM:  estDistanceM,
		estDurationS:  estDurationS,
		estFareCents:  estFareCents,
		currency:      currency,
		offerTTL:      offerTTL,
		offeredAt:     offeredAt,
		expiresAt:     expiresAt,
		respondedAt:   respondedAt,
		acceptedAt:    acceptedAt,
		rejectedAt:    rejectedAt,
		expiredAt:     expiredAt,
		cancelledAt:   cancelledAt,
		rejectReason:  rejectReason,
		attemptNumber: attemptNumber,
		createdBy:     createdBy,
	}
}

// ===== Accessors =====

func (o DispatchOffer) ID() string            { return o.id }
func (o DispatchOffer) OrderID() string       { return o.orderID }
func (o DispatchOffer) DriverID() string      { return o.driverID }
func (o DispatchOffer) ZoneID() string        { return o.zoneID }
func (o DispatchOffer) Status() OfferStatus   { return o.status }
func (o DispatchOffer) PickupLat() float64    { return o.pickupLat }
func (o DispatchOffer) PickupLng() float64    { return o.pickupLng }
func (o DispatchOffer) DeliveryLat() float64  { return o.deliveryLat }
func (o DispatchOffer) DeliveryLng() float64  { return o.deliveryLng }
func (o DispatchOffer) EstDistanceM() *int    { return o.estDistanceM }
func (o DispatchOffer) EstDurationS() *int    { return o.estDurationS }
func (o DispatchOffer) EstFareCents() *int64  { return o.estFareCents }
func (o DispatchOffer) Currency() string      { return o.currency }
func (o DispatchOffer) OfferTTL() time.Duration { return o.offerTTL }
func (o DispatchOffer) OfferedAt() time.Time  { return o.offeredAt }
func (o DispatchOffer) ExpiresAt() time.Time  { return o.expiresAt }
func (o DispatchOffer) RespondedAt() *time.Time { return o.respondedAt }
func (o DispatchOffer) AcceptedAt() *time.Time  { return o.acceptedAt }
func (o DispatchOffer) RejectedAt() *time.Time  { return o.rejectedAt }
func (o DispatchOffer) ExpiredAt() *time.Time   { return o.expiredAt }
func (o DispatchOffer) CancelledAt() *time.Time  { return o.cancelledAt }
func (o DispatchOffer) RejectReason() string     { return o.rejectReason }
func (o DispatchOffer) AttemptNumber() int       { return o.attemptNumber }
func (o DispatchOffer) CreatedBy() string        { return o.createdBy }

// IsPending reports whether the offer is awaiting a response.
func (o DispatchOffer) IsPending() bool { return o.status == OfferStatusPending }

// IsAccepted reports whether the offer was accepted by the driver.
func (o DispatchOffer) IsAccepted() bool { return o.status == OfferStatusAccepted }

// IsExpired reports whether the offer deadline has passed.
func (o DispatchOffer) IsExpired(now time.Time) bool {
	return o.status == OfferStatusPending && now.After(o.expiresAt)
}

// ===== Status Transitions =====

// Accept marks the offer as accepted by the driver.
// Returns an error if the offer is not pending or has expired.
func (o DispatchOffer) Accept(now time.Time) (DispatchOffer, error) {
	if o.status != OfferStatusPending {
		if o.status == OfferStatusAccepted {
			return o, ErrOfferAlreadyAccepted
		}
		return o, fmt.Errorf("%w: cannot accept from %s", ErrOfferNotPending, o.status)
	}
	if now.After(o.expiresAt) {
		return o, ErrOfferExpired
	}
	o.status = OfferStatusAccepted
	o.respondedAt = &now
	o.acceptedAt = &now
	return o, nil
}

// Reject marks the offer as rejected by the driver.
// reason is optional.
func (o DispatchOffer) Reject(reason string, now time.Time) (DispatchOffer, error) {
	if o.status != OfferStatusPending {
		if o.status == OfferStatusRejected {
			return o, ErrOfferAlreadyRejected
		}
		return o, fmt.Errorf("%w: cannot reject from %s", ErrOfferNotPending, o.status)
	}
	o.status = OfferStatusRejected
	o.respondedAt = &now
	o.rejectedAt = &now
	o.rejectReason = reason
	return o, nil
}

// Expire marks the offer as expired (driver did not respond in time).
func (o DispatchOffer) Expire(now time.Time) (DispatchOffer, error) {
	if o.status != OfferStatusPending {
		if o.status == OfferStatusExpired {
			return o, nil // idempotent
		}
		return o, fmt.Errorf("%w: cannot expire from %s", ErrOfferNotPending, o.status)
	}
	o.status = OfferStatusExpired
	o.expiredAt = &now
	o.respondedAt = &now
	return o, nil
}

// Cancel marks the offer as cancelled by the system.
// Used when the order is cancelled or the offer is reassigned.
func (o DispatchOffer) Cancel(now time.Time) (DispatchOffer, error) {
	if o.status != OfferStatusPending {
		if o.status == OfferStatusCancelled {
			return o, nil // idempotent
		}
		return o, fmt.Errorf("%w: cannot cancel from %s", ErrOfferNotPending, o.status)
	}
	o.status = OfferStatusCancelled
	o.cancelledAt = &now
	return o, nil
}
