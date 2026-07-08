// Package domain driver_location: DriverLocation aggregate root.
//
// DriverLocation tracks the most recent GPS position of a driver. It is
// updated frequently (every few seconds when on duty) and is the basis for
// "find nearest driver" queries.
//
// Design decisions:
//   - Stored separately from Driver to avoid write contention on the Driver
//     row. Every GPS ping would otherwise lock the Driver row.
//   - The location is immutable after creation; updates insert a new row
//     (or upsert via the repository layer).
//   - Staleness check: a location older than X seconds (configurable) is
//     considered stale and the driver is treated as offline for matching.
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// DriverLocation is the current GPS position of a driver.
type DriverLocation struct {
	id          string
	driverID    string
	lat         float64
	lng         float64
	bearing     float64 // 0-360 degrees, 0 = north
	speed       float64 // km/h
	accuracy    float64 // meters
	capturedAt  time.Time // when the GPS reading was taken
	receivedAt  time.Time // when the server received it
}

// NewDriverLocation creates a new DriverLocation with validation.
func NewDriverLocation(
	id, driverID string,
	lat, lng, bearing, speed, accuracy float64,
	capturedAt, receivedAt time.Time,
) (DriverLocation, error) {
	if id == "" {
		return DriverLocation{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if driverID == "" {
		return DriverLocation{}, fmt.Errorf("%w: driver id is required", ErrInvalidInput)
	}
	if lat < -90 || lat > 90 {
		return DriverLocation{}, ErrInvalidLatitude
	}
	if lng < -180 || lng > 180 {
		return DriverLocation{}, ErrInvalidLongitude
	}
	if bearing < 0 || bearing > 360 {
		return DriverLocation{}, ErrInvalidBearing
	}
	if accuracy < 0 {
		return DriverLocation{}, fmt.Errorf("%w: accuracy must be >= 0", ErrInvalidInput)
	}
	if capturedAt.IsZero() {
		capturedAt = receivedAt
	}
	return DriverLocation{
		id:         id,
		driverID:   driverID,
		lat:        lat,
		lng:        lng,
		bearing:    bearing,
		speed:      speed,
		accuracy:   accuracy,
		capturedAt: capturedAt,
		receivedAt: receivedAt,
	}, nil
}

// RehydrateDriverLocation reconstructs a DriverLocation from persistence.
func RehydrateDriverLocation(
	id, driverID string,
	lat, lng, bearing, speed, accuracy float64,
	capturedAt, receivedAt time.Time,
) DriverLocation {
	return DriverLocation{
		id:         id,
		driverID:   driverID,
		lat:        lat,
		lng:        lng,
		bearing:    bearing,
		speed:      speed,
		accuracy:   accuracy,
		capturedAt: capturedAt,
		receivedAt: receivedAt,
	}
}

// ===== Accessors =====

func (l DriverLocation) ID() string          { return l.id }
func (l DriverLocation) DriverID() string    { return l.driverID }
func (l DriverLocation) Lat() float64        { return l.lat }
func (l DriverLocation) Lng() float64        { return l.lng }
func (l DriverLocation) Bearing() float64    { return l.bearing }
func (l DriverLocation) Speed() float64      { return l.speed }
func (l DriverLocation) Accuracy() float64   { return l.accuracy }
func (l DriverLocation) CapturedAt() time.Time { return l.capturedAt }
func (l DriverLocation) ReceivedAt() time.Time { return l.receivedAt }

// IsStale reports whether the location is older than the given TTL.
func (l DriverLocation) IsStale(now time.Time, ttl time.Duration) bool {
	return now.Sub(l.capturedAt) > ttl
}

// DistanceToMeters computes the great-circle distance from this location to
// the given (lat, lng) point using the Haversine formula.
// Returns the distance in meters.
func (l DriverLocation) DistanceToMeters(lat, lng float64) float64 {
	return haversineMeters(l.lat, l.lng, lat, lng)
}

// haversineMeters computes the great-circle distance between two points
// using the Haversine formula. Returns meters.
//
// Inputs:
//   lat1, lng1: latitude and longitude of point 1 (degrees)
//   lat2, lng2: latitude and longitude of point 2 (degrees)
//
// Earth radius: 6,371,000 meters.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0

	// Convert to radians.
	lat1Rad := lat1 * pi / 180
	lat2Rad := lat2 * pi / 180
	deltaLat := (lat2 - lat1) * pi / 180
	deltaLng := (lng2 - lng1) * pi / 180

	// Haversine formula.
	a := sin(deltaLat/2)*sin(deltaLat/2) +
		cos(lat1Rad)*cos(lat2Rad)*sin(deltaLng/2)*sin(deltaLng/2)
	c := 2 * atan2(sqrt(a), sqrt(1-a))

	return earthRadiusM * c
}

// math constants + wrappers (avoids importing "math" in tests).
const pi = 3.14159265358979323846

var (
	sin  = mathSin
	cos  = mathCos
	atan2 = mathAtan2
	sqrt = mathSqrt
)
