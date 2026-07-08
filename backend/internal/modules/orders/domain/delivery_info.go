// Package domain delivery_info: DeliveryInfo value object.
//
// Encapsulates the customer's delivery location: coordinates, address, and notes.
// Validation ensures valid geographic coordinates and a non-empty address.
//
// Imports stdlib only.
package domain

import "fmt"

// DeliveryInfo holds the customer's delivery location details.
type DeliveryInfo struct {
	lat     float64
	lng     float64
	address string
	notes   string
}

// NewDeliveryInfo creates a DeliveryInfo with validation.
// lat must be between -90 and 90.
// lng must be between -180 and 180.
// address must be non-empty.
func NewDeliveryInfo(lat, lng float64, address, notes string) (DeliveryInfo, error) {
	if lat < -90 || lat > 90 {
		return DeliveryInfo{}, fmt.Errorf("%w: %f", ErrInvalidLatitude, lat)
	}
	if lng < -180 || lng > 180 {
		return DeliveryInfo{}, fmt.Errorf("%w: %f", ErrInvalidLongitude, lng)
	}
	if address == "" {
		return DeliveryInfo{}, ErrDeliveryAddressRequired
	}
	return DeliveryInfo{
		lat:     lat,
		lng:     lng,
		address: address,
		notes:   notes,
	}, nil
}

// Lat returns the delivery latitude.
func (d DeliveryInfo) Lat() float64 {
	return d.lat
}

// Lng returns the delivery longitude.
func (d DeliveryInfo) Lng() float64 {
	return d.lng
}

// Address returns the delivery address text.
func (d DeliveryInfo) Address() string {
	return d.address
}

// Notes returns delivery instructions (e.g. "apartment 3, floor 2").
func (d DeliveryInfo) Notes() string {
	return d.notes
}

// HasNotes reports whether delivery notes are present.
func (d DeliveryInfo) HasNotes() bool {
	return d.notes != ""
}

// IsZero reports whether the delivery info is unset (all zero values).
func (d DeliveryInfo) IsZero() bool {
	return d.lat == 0 && d.lng == 0 && d.address == ""
}
