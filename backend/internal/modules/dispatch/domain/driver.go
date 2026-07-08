// Package domain driver: Driver aggregate root.
//
// A Driver represents a delivery driver registered in the system.
// Drivers have a lifecycle: offline → online → busy → offline.
// A driver can only receive offers when online and not busy.
//
// Invariants:
//   - A driver with an active order cannot accept another offer.
//   - A suspended driver cannot go online.
//   - Rating must be in [0, 5].
//   - Acceptance rate must be in [0, 100].
//
// The driver entity does NOT store the current location — that's stored in
// DriverLocation (separate aggregate, frequently updated).
//
// Imports stdlib only.
package domain

import (
        "fmt"
        "time"
)

// DriverStatus enumerates driver lifecycle states.
type DriverStatus string

const (
        DriverStatusOffline    DriverStatus = "offline"
        DriverStatusOnline     DriverStatus = "online"
        DriverStatusBusy       DriverStatus = "busy"
        DriverStatusSuspended  DriverStatus = "suspended"
)

// VehicleType enumerates supported vehicle types.
type VehicleType string

const (
        VehicleTypeBike    VehicleType = "bike"
        VehicleTypeScooter VehicleType = "scooter"
        VehicleTypeCar     VehicleType = "car"
)

// Driver is the aggregate root for a delivery driver.
type Driver struct {
        id              string
        userID          string // links to identity module
        vehicleType     VehicleType
        licensePlate    string
        status          DriverStatus
        rating          float64
        ratingCount     int
        acceptanceRate  int     // 0-100, last 100 offers
        completionRate  int     // 0-100, last 100 orders
        totalDeliveries int
        zoneIDs         []string // zones the driver is willing to serve
        currentOrderID  string   // empty when not busy
        goOnlineAt      *time.Time
        goOfflineAt     *time.Time
        suspendedReason string
        createdAt       time.Time
        updatedAt       time.Time
        version         int // optimistic locking
}

// NewDriver creates a new Driver with validation.
// New drivers start in offline status.
func NewDriver(
        id, userID string,
        vehicleType VehicleType,
        licensePlate string,
        zoneIDs []string,
        now time.Time,
) (Driver, error) {
        if id == "" {
                return Driver{}, fmt.Errorf("%w: id is required", ErrInvalidID)
        }
        if userID == "" {
                return Driver{}, fmt.Errorf("%w: user_id is required", ErrInvalidInput)
        }
        if !isValidVehicleType(vehicleType) {
                return Driver{}, fmt.Errorf("%w: %s", ErrInvalidVehicleType, vehicleType)
        }
        if licensePlate == "" {
                return Driver{}, fmt.Errorf("%w: license plate is required", ErrInvalidInput)
        }
        return Driver{
                id:           id,
                userID:       userID,
                vehicleType:  vehicleType,
                licensePlate: licensePlate,
                status:       DriverStatusOffline,
                rating:       5.0,
                ratingCount:  0,
                acceptanceRate: 100,
                completionRate: 100,
                zoneIDs:      zoneIDs,
                createdAt:    now,
                updatedAt:    now,
                version:      1,
        }, nil
}

// RehydrateDriver reconstructs a Driver from persistence (no validation).
func RehydrateDriver(
        id, userID string,
        vehicleType VehicleType,
        licensePlate string,
        status DriverStatus,
        rating float64,
        ratingCount, acceptanceRate, completionRate, totalDeliveries int,
        zoneIDs []string,
        currentOrderID string,
        goOnlineAt, goOfflineAt *time.Time,
        suspendedReason string,
        createdAt, updatedAt time.Time,
        version int,
) Driver {
        return Driver{
                id:              id,
                userID:          userID,
                vehicleType:     vehicleType,
                licensePlate:    licensePlate,
                status:          status,
                rating:          rating,
                ratingCount:     ratingCount,
                acceptanceRate:  acceptanceRate,
                completionRate:  completionRate,
                totalDeliveries: totalDeliveries,
                zoneIDs:         zoneIDs,
                currentOrderID:  currentOrderID,
                goOnlineAt:      goOnlineAt,
                goOfflineAt:     goOfflineAt,
                suspendedReason: suspendedReason,
                createdAt:       createdAt,
                updatedAt:       updatedAt,
                version:         version,
        }
}

func isValidVehicleType(v VehicleType) bool {
        switch v {
        case VehicleTypeBike, VehicleTypeScooter, VehicleTypeCar:
                return true
        }
        return false
}

// ===== Accessors =====

func (d Driver) ID() string               { return d.id }
func (d Driver) UserID() string           { return d.userID }
func (d Driver) VehicleType() VehicleType { return d.vehicleType }
func (d Driver) LicensePlate() string     { return d.licensePlate }
func (d Driver) Status() DriverStatus     { return d.status }
func (d Driver) Rating() float64          { return d.rating }
func (d Driver) RatingCount() int         { return d.ratingCount }
func (d Driver) AcceptanceRate() int      { return d.acceptanceRate }
func (d Driver) CompletionRate() int      { return d.completionRate }
func (d Driver) TotalDeliveries() int     { return d.totalDeliveries }
func (d Driver) ZoneIDs() []string        { return d.zoneIDs }
func (d Driver) CurrentOrderID() string   { return d.currentOrderID }
func (d Driver) GoOnlineAt() *time.Time   { return d.goOnlineAt }
func (d Driver) GoOfflineAt() *time.Time  { return d.goOfflineAt }
func (d Driver) SuspendedReason() string  { return d.suspendedReason }
func (d Driver) CreatedAt() time.Time     { return d.createdAt }
func (d Driver) UpdatedAt() time.Time     { return d.updatedAt }
func (d Driver) Version() int             { return d.version }

// IsOnline reports whether the driver is currently online and available.
func (d Driver) IsOnline() bool { return d.status == DriverStatusOnline }

// IsBusy reports whether the driver is currently delivering an order.
func (d Driver) IsBusy() bool { return d.status == DriverStatusBusy }

// IsSuspended reports whether the driver is suspended.
func (d Driver) IsSuspended() bool { return d.status == DriverStatusSuspended }

// IsAvailableForOffer reports whether the driver can receive a new offer.
func (d Driver) IsAvailableForOffer() bool { return d.status == DriverStatusOnline }

// ServesZone reports whether the driver serves the given zone.
// If the driver has no zone restrictions (empty list), serves all zones.
func (d Driver) ServesZone(zoneID string) bool {
        if len(d.zoneIDs) == 0 {
                return true
        }
        for _, z := range d.zoneIDs {
                if z == zoneID {
                        return true
                }
        }
        return false
}

// ===== Mutations (immutable pattern) =====

// GoOnline transitions the driver from offline to online.
func (d Driver) GoOnline(now time.Time) (Driver, error) {
        if d.IsSuspended() {
                return d, ErrDriverSuspended
        }
        if d.IsBusy() {
                return d, fmt.Errorf("%w: cannot go online while busy", ErrDriverOnDuty)
        }
        if d.IsOnline() {
                return d, nil // idempotent
        }
        d.status = DriverStatusOnline
        d.goOnlineAt = &now
        d.goOfflineAt = nil
        d.updatedAt = now
        return d, nil
}

// GoOffline transitions the driver from online to offline.
// Drivers cannot go offline while busy with an active order.
func (d Driver) GoOffline(now time.Time) (Driver, error) {
        if d.IsBusy() {
                return d, fmt.Errorf("%w: cannot go offline while delivering an order", ErrDriverOnDuty)
        }
        if d.status == DriverStatusOffline {
                return d, nil // idempotent
        }
        d.status = DriverStatusOffline
        d.goOfflineAt = &now
        d.updatedAt = now
        return d, nil
}

// StartOrder transitions the driver from online to busy.
// Called when the driver accepts a dispatch offer.
func (d Driver) StartOrder(orderID string, now time.Time) (Driver, error) {
        if d.IsBusy() {
                return d, fmt.Errorf("%w: already delivering order %s", ErrDriverBusy, d.currentOrderID)
        }
        if !d.IsOnline() {
                return d, ErrDriverOffline
        }
        if orderID == "" {
                return d, fmt.Errorf("%w: order id is required", ErrInvalidInput)
        }
        d.status = DriverStatusBusy
        d.currentOrderID = orderID
        d.updatedAt = now
        return d, nil
}

// CompleteOrder transitions the driver from busy back to online.
// Called when the driver delivers the order.
func (d Driver) CompleteOrder(now time.Time) (Driver, error) {
        if !d.IsBusy() {
                return d, fmt.Errorf("%w: not currently busy", ErrDriverOffline)
        }
        d.status = DriverStatusOnline
        d.currentOrderID = ""
        d.totalDeliveries++
        d.updatedAt = now
        return d, nil
}

// Suspend transitions the driver to suspended status with a reason.
func (d Driver) Suspend(reason string, now time.Time) (Driver, error) {
        if d.IsBusy() {
                return d, fmt.Errorf("%w: cannot suspend while delivering", ErrDriverOnDuty)
        }
        d.status = DriverStatusSuspended
        d.suspendedReason = reason
        d.goOfflineAt = &now
        d.updatedAt = now
        return d, nil
}

// Unsuspend transitions the driver from suspended back to offline.
func (d Driver) Unsuspend(now time.Time) (Driver, error) {
        if !d.IsSuspended() {
                return d, fmt.Errorf("%w: not suspended", ErrInvalidDriverStatus)
        }
        d.status = DriverStatusOffline
        d.suspendedReason = ""
        d.updatedAt = now
        return d, nil
}

// UpdateRating updates the driver's rolling rating.
// newRating is the rating from a single completed order (1-5 stars).
// Uses a simple running average.
func (d Driver) UpdateRating(newRating float64, now time.Time) (Driver, error) {
        if newRating < 0 || newRating > 5 {
                return d, fmt.Errorf("%w: rating must be between 0 and 5", ErrInvalidInput)
        }
        // Running average: new_avg = (old_avg * count + new) / (count + 1)
        total := d.rating * float64(d.ratingCount)
        d.ratingCount++
        d.rating = (total + newRating) / float64(d.ratingCount)
        d.updatedAt = now
        return d, nil
}

// RecordOfferDecision updates the acceptance rate based on whether the driver
// accepted or rejected an offer.
func (d Driver) RecordOfferDecision(accepted bool, now time.Time) Driver {
        // Simple sliding window of last 100 offers.
        // For simplicity here, we use a weighted update.
        if accepted {
                d.acceptanceRate = (d.acceptanceRate*99 + 100) / 100
        } else {
                d.acceptanceRate = (d.acceptanceRate*99 + 0) / 100
        }
        d.updatedAt = now
        return d
}

// BumpVersion increments the version for optimistic locking.
func (d Driver) BumpVersion() Driver {
        d.version++
        return d
}
