// Package domain tests: DriverLocation + DispatchOffer.
package domain

import (
	"testing"
	"time"
)

func TestNewDriverLocation(t *testing.T) {
	now := testNowDispatch()
	tests := []struct {
		name     string
		lat      float64
		lng      float64
		bearing  float64
		accuracy float64
		wantErr  error
	}{
		{"valid", 30.0444, 31.2357, 90.0, 5.0, nil},
		{"equator", 0, 0, 0, 0, nil},
		{"north pole", 90, 0, 0, 0, nil},
		{"south pole", -90, 0, 0, 0, nil},
		{"lat too high", 91, 0, 0, 0, ErrInvalidLatitude},
		{"lat too low", -91, 0, 0, 0, ErrInvalidLatitude},
		{"lng too high", 0, 181, 0, 0, ErrInvalidLongitude},
		{"lng too low", 0, -181, 0, 0, ErrInvalidLongitude},
		{"bearing too high", 0, 0, 361, 0, ErrInvalidBearing},
		{"bearing negative", 0, 0, -1, 0, ErrInvalidBearing},
		{"negative accuracy", 0, 0, 0, -1, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDriverLocation("l1", "d1", tt.lat, tt.lng, tt.bearing, 0, tt.accuracy, now, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDriverLocationIsStale(t *testing.T) {
	now := testNowDispatch()
	loc, _ := NewDriverLocation("l1", "d1", 30.0, 31.0, 0, 0, 5, now, now)

	// Fresh
	if loc.IsStale(now, 30*time.Second) {
		t.Errorf("location should not be stale")
	}

	// Stale after 60 seconds (TTL = 30s)
	future := now.Add(60 * time.Second)
	if !loc.IsStale(future, 30*time.Second) {
		t.Errorf("location should be stale")
	}
}

func TestHaversineDistance(t *testing.T) {
	// Cairo: 30.0444, 31.2357
	// Giza Pyramids: 29.9792, 31.1342
	// Expected distance: ~13 km
	loc, _ := NewDriverLocation("l1", "d1", 30.0444, 31.2357, 0, 0, 5, testNowDispatch(), testNowDispatch())
	dist := loc.DistanceToMeters(29.9792, 31.1342)

	// Should be around 13,000 meters (allow ±2km tolerance for haversine)
	if dist < 11000 || dist > 15000 {
		t.Errorf("Cairo to Giza expected ~13000m, got %.0f", dist)
	}

	// Same point — should be 0
	samePoint := loc.DistanceToMeters(30.0444, 31.2357)
	if samePoint > 1 {
		t.Errorf("same point should be 0, got %f", samePoint)
	}
}

// ===== DispatchOffer Tests =====

func TestNewDispatchOffer(t *testing.T) {
	now := testNowDispatch()
	dist := 1500
	dur := 300
	fare := int64(500)

	tests := []struct {
		name        string
		id          string
		orderID     string
		driverID    string
		pickupLat   float64
		pickupLng   float64
		offerTTL    time.Duration
		wantErr     error
	}{
		{"valid", "o1", "ord-1", "d1", 30.0, 31.0, 15 * time.Second, nil},
		{"empty id", "", "ord-1", "d1", 30.0, 31.0, 15 * time.Second, ErrInvalidID},
		{"empty order", "o2", "", "d1", 30.0, 31.0, 15 * time.Second, ErrInvalidInput},
		{"empty driver", "o3", "ord-1", "", 30.0, 31.0, 15 * time.Second, ErrInvalidInput},
		{"invalid lat", "o4", "ord-1", "d1", 91, 31.0, 15 * time.Second, ErrInvalidLatitude},
		{"invalid lng", "o5", "ord-1", "d1", 30.0, 181, 15 * time.Second, ErrInvalidLongitude},
		{"zero TTL", "o6", "ord-1", "d1", 30.0, 31.0, 0, ErrInvalidInput},
		{"negative TTL", "o7", "ord-1", "d1", 30.0, 31.0, -1, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDispatchOffer(tt.id, tt.orderID, tt.driverID, "zone-1",
				tt.pickupLat, tt.pickupLng, 30.05, 31.05,
				&dist, &dur, &fare, "EGP",
				tt.offerTTL, 1, "system", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestOfferAcceptReject(t *testing.T) {
	now := testNowDispatch()
	dist := 1500
	dur := 300
	fare := int64(500)

	offer, _ := NewDispatchOffer("o1", "ord-1", "d1", "zone-1",
		30.0, 31.0, 30.05, 31.05,
		&dist, &dur, &fare, "EGP",
		15*time.Second, 1, "system", now)

	if !offer.IsPending() {
		t.Fatalf("expected pending")
	}

	// Accept
	accepted, err := offer.Accept(now)
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	if !accepted.IsAccepted() {
		t.Errorf("expected accepted")
	}
	if accepted.AcceptedAt() == nil {
		t.Errorf("expected accepted_at set")
	}

	// Idempotent error
	_, err = accepted.Accept(now)
	if !errIs(err, ErrOfferAlreadyAccepted) {
		t.Fatalf("expected ErrOfferAlreadyAccepted, got %v", err)
	}
}

func TestOfferExpireAfterTTL(t *testing.T) {
	now := testNowDispatch()
	dist := 1500
	dur := 300
	fare := int64(500)

	offer, _ := NewDispatchOffer("o1", "ord-1", "d1", "zone-1",
		30.0, 31.0, 30.05, 31.05,
		&dist, &dur, &fare, "EGP",
		15*time.Second, 1, "system", now)

	// After 16 seconds (TTL = 15s)
	future := now.Add(16 * time.Second)

	// Accept should fail with ErrOfferExpired
	_, err := offer.Accept(future)
	if !errIs(err, ErrOfferExpired) {
		t.Fatalf("expected ErrOfferExpired, got %v", err)
	}

	// Explicitly expire it
	expired, err := offer.Expire(future)
	if err != nil {
		t.Fatalf("expire failed: %v", err)
	}
	if expired.Status() != OfferStatusExpired {
		t.Errorf("expected expired status, got %s", expired.Status())
	}
}

func TestOfferCancel(t *testing.T) {
	now := testNowDispatch()
	dist := 1500
	dur := 300
	fare := int64(500)

	offer, _ := NewDispatchOffer("o1", "ord-1", "d1", "zone-1",
		30.0, 31.0, 30.05, 31.05,
		&dist, &dur, &fare, "EGP",
		15*time.Second, 1, "system", now)

	cancelled, err := offer.Cancel(now)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if cancelled.Status() != OfferStatusCancelled {
		t.Errorf("expected cancelled, got %s", cancelled.Status())
	}
}

// errIs helper (re-declared since this is a separate test package).
func errIs(err, target error) bool {
	if err == target {
		return true
	}
	for {
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == target {
			return true
		}
		if err == nil {
			return false
		}
	}
}
