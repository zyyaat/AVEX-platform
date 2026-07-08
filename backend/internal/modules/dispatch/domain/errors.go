// Package domain contains pure domain entities for the dispatch module.
//
// This file: typed domain errors for drivers, locations, and dispatch offers.
//
// Imports stdlib only.
package domain

import (
	"errors"
	"fmt"
)

// ===== Driver Errors =====

var ErrDriverNotFound = errors.New("driver not found")
var ErrDriverAlreadyExists = errors.New("driver already exists")
var ErrDriverOffline = errors.New("driver is offline")
var ErrDriverOnDuty = errors.New("driver is on duty and cannot be modified")
var ErrDriverBusy = errors.New("driver is busy with another order")
var ErrDriverSuspended = errors.New("driver is suspended")
var ErrInvalidDriverStatus = errors.New("invalid driver status")
var ErrInvalidVehicleType = errors.New("invalid vehicle type")

// ===== Location Errors =====

var ErrLocationNotFound = errors.New("driver location not found")
var ErrLocationTooStale = errors.New("driver location is too stale")
var ErrInvalidLatitude = errors.New("invalid latitude (must be between -90 and 90)")
var ErrInvalidLongitude = errors.New("invalid longitude (must be between -180 and 180)")
var ErrInvalidBearing = errors.New("invalid bearing (must be between 0 and 360)")
var ErrNoDriversAvailable = errors.New("no drivers available in this area")

// ===== Dispatch Offer Errors =====

var ErrOfferNotFound = errors.New("dispatch offer not found")
var ErrOfferAlreadyExists = errors.New("dispatch offer already exists for this order")
var ErrOfferExpired = errors.New("dispatch offer has expired")
var ErrOfferAlreadyAccepted = errors.New("dispatch offer already accepted")
var ErrOfferAlreadyRejected = errors.New("dispatch offer already rejected")
var ErrOfferAlreadyCancelled = errors.New("dispatch offer already cancelled")
var ErrOfferNotPending = errors.New("dispatch offer is not pending")
var ErrMaxAttemptsReached = errors.New("maximum dispatch attempts reached for this order")
var ErrDriverNotEligible = errors.New("driver is not eligible for this order")

// ===== Generic Validation =====

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrInvalidRadius = errors.New("invalid radius (must be > 0)")
var ErrInvalidLimit = errors.New("invalid limit (must be > 0)")

// ===== Composite Error =====

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
