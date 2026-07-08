// Package domain tests: Driver aggregate.
package domain

import (
	"testing"
	"time"
)

func testNowDispatch() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

func TestNewDriver(t *testing.T) {
	now := testNowDispatch()
	tests := []struct {
		name         string
		id           string
		userID       string
		vehicleType  VehicleType
		licensePlate string
		wantErr      error
	}{
		{"valid bike", "d1", "u1", VehicleTypeBike, "ABC-123", nil},
		{"valid scooter", "d2", "u2", VehicleTypeScooter, "XYZ-789", nil},
		{"valid car", "d3", "u3", VehicleTypeCar, "CAR-001", nil},
		{"empty id", "", "u1", VehicleTypeBike, "ABC", ErrInvalidID},
		{"empty user id", "d4", "", VehicleTypeBike, "ABC", ErrInvalidInput},
		{"invalid vehicle", "d5", "u1", VehicleType("truck"), "ABC", ErrInvalidVehicleType},
		{"empty license plate", "d6", "u1", VehicleTypeBike, "", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDriver(tt.id, tt.userID, tt.vehicleType, tt.licensePlate, nil, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Status() != DriverStatusOffline {
				t.Errorf("expected offline, got %s", d.Status())
			}
			if d.Rating() != 5.0 {
				t.Errorf("expected initial rating 5.0, got %f", d.Rating())
			}
			if d.Version() != 1 {
				t.Errorf("expected version 1, got %d", d.Version())
			}
		})
	}
}

func TestDriverGoOnlineOffline(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)

	// Go online
	online, err := d.GoOnline(now)
	if err != nil {
		t.Fatalf("go online failed: %v", err)
	}
	if !online.IsOnline() {
		t.Errorf("expected online")
	}
	if online.GoOnlineAt() == nil {
		t.Errorf("expected go_online_at set")
	}

	// Idempotent
	online2, _ := online.GoOnline(now)
	if !online2.IsOnline() {
		t.Errorf("expected still online")
	}

	// Go offline
	offline, err := online.GoOffline(now)
	if err != nil {
		t.Fatalf("go offline failed: %v", err)
	}
	if offline.Status() != DriverStatusOffline {
		t.Errorf("expected offline, got %s", offline.Status())
	}
}

func TestDriverSuspendBlockOnline(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)

	// Suspend
	suspended, err := d.Suspend("fraud", now)
	if err != nil {
		t.Fatalf("suspend failed: %v", err)
	}
	if !suspended.IsSuspended() {
		t.Errorf("expected suspended")
	}
	if suspended.SuspendedReason() != "fraud" {
		t.Errorf("reason: %s", suspended.SuspendedReason())
	}

	// Cannot go online while suspended
	_, err = suspended.GoOnline(now)
	if !errIs(err, ErrDriverSuspended) {
		t.Fatalf("expected ErrDriverSuspended, got %v", err)
	}

	// Unsuspend
	unsuspended, err := suspended.Unsuspend(now)
	if err != nil {
		t.Fatalf("unsuspend failed: %v", err)
	}
	if unsuspended.IsSuspended() {
		t.Errorf("expected not suspended")
	}
	if unsuspended.Status() != DriverStatusOffline {
		t.Errorf("expected offline after unsuspend")
	}
}

func TestDriverStartCompleteOrder(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)
	online, _ := d.GoOnline(now)

	// Start order
	busy, err := online.StartOrder("ord-1", now)
	if err != nil {
		t.Fatalf("start order failed: %v", err)
	}
	if !busy.IsBusy() {
		t.Errorf("expected busy")
	}
	if busy.CurrentOrderID() != "ord-1" {
		t.Errorf("current order: %s", busy.CurrentOrderID())
	}

	// Cannot start another order while busy
	_, err = busy.StartOrder("ord-2", now)
	if !errIs(err, ErrDriverBusy) {
		t.Fatalf("expected ErrDriverBusy, got %v", err)
	}

	// Cannot go offline while busy
	_, err = busy.GoOffline(now)
	if !errIs(err, ErrDriverOnDuty) {
		t.Fatalf("expected ErrDriverOnDuty, got %v", err)
	}

	// Complete order
	available, err := busy.CompleteOrder(now)
	if err != nil {
		t.Fatalf("complete order failed: %v", err)
	}
	if !available.IsOnline() {
		t.Errorf("expected online after completion")
	}
	if available.CurrentOrderID() != "" {
		t.Errorf("expected empty current order id")
	}
	if available.TotalDeliveries() != 1 {
		t.Errorf("expected 1 delivery, got %d", available.TotalDeliveries())
	}
}

func TestDriverStartOrderWhileOffline(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)

	_, err := d.StartOrder("ord-1", now)
	if !errIs(err, ErrDriverOffline) {
		t.Fatalf("expected ErrDriverOffline, got %v", err)
	}
}

func TestDriverUpdateRating(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)

	// Initial: rating=5.0, count=0
	// After 4-star: rating = (5.0*0 + 4) / 1 = 4.0, count=1
	d, _ = d.UpdateRating(4.0, now)
	if d.Rating() != 4.0 {
		t.Errorf("expected 4.0, got %f", d.Rating())
	}
	if d.RatingCount() != 1 {
		t.Errorf("expected count 1, got %d", d.RatingCount())
	}

	// After 5-star: rating = (4.0*1 + 5) / 2 = 4.5
	d, _ = d.UpdateRating(5.0, now)
	if d.Rating() != 4.5 {
		t.Errorf("expected 4.5, got %f", d.Rating())
	}

	// Invalid rating
	_, err := d.UpdateRating(6.0, now)
	if !errIs(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDriverRecordOfferDecision(t *testing.T) {
	now := testNowDispatch()
	d, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)
	// Initial acceptance rate = 100
	if d.AcceptanceRate() != 100 {
		t.Fatalf("expected 100, got %d", d.AcceptanceRate())
	}

	// Reject 1 offer: ar = (100*99 + 0)/100 = 99
	d = d.RecordOfferDecision(false, now)
	if d.AcceptanceRate() != 99 {
		t.Errorf("expected 99, got %d", d.AcceptanceRate())
	}

	// Accept 1 offer: ar = (99*99 + 100)/100 = 99.01 -> 99 (integer)
	d = d.RecordOfferDecision(true, now)
	// ar = (99*99 + 100)/100 = 9901/100 = 99 (integer division)
	if d.AcceptanceRate() != 99 {
		t.Errorf("expected 99, got %d", d.AcceptanceRate())
	}
}

func TestDriverServesZone(t *testing.T) {
	now := testNowDispatch()
	// Driver with no zone restrictions — serves all zones
	d1, _ := NewDriver("d1", "u1", VehicleTypeBike, "ABC", nil, now)
	if !d1.ServesZone("any-zone") {
		t.Errorf("driver with no zones should serve all")
	}

	// Driver with specific zones
	d2, _ := NewDriver("d2", "u1", VehicleTypeBike, "ABC", []string{"zone-1", "zone-2"}, now)
	if !d2.ServesZone("zone-1") {
		t.Errorf("expected to serve zone-1")
	}
	if d2.ServesZone("zone-3") {
		t.Errorf("did not expect to serve zone-3")
	}
}
