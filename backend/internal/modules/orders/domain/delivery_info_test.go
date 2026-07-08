// Package domain tests: DeliveryInfo value object.
package domain

import (
	"errors"
	"testing"
)

func TestNewDeliveryInfo_Success(t *testing.T) {
	info, err := NewDeliveryInfo(30.05, 31.36, "Nasr City, Cairo", "Apartment 3, Floor 2")
	if err != nil {
		t.Fatalf("NewDeliveryInfo error: %v", err)
	}
	if info.Lat() != 30.05 {
		t.Errorf("Lat = %f", info.Lat())
	}
	if info.Lng() != 31.36 {
		t.Errorf("Lng = %f", info.Lng())
	}
	if info.Address() != "Nasr City, Cairo" {
		t.Errorf("Address = %q", info.Address())
	}
	if info.Notes() != "Apartment 3, Floor 2" {
		t.Errorf("Notes = %q", info.Notes())
	}
	if !info.HasNotes() {
		t.Error("should have notes")
	}
}

func TestNewDeliveryInfo_NoNotes(t *testing.T) {
	info, err := NewDeliveryInfo(30.05, 31.36, "Maadi, Cairo", "")
	if err != nil {
		t.Fatalf("NewDeliveryInfo error: %v", err)
	}
	if info.HasNotes() {
		t.Error("should not have notes")
	}
}

func TestNewDeliveryInfo_InvalidLatitude(t *testing.T) {
	tests := []float64{-91, 91, -180, 180}
	for _, lat := range tests {
		_, err := NewDeliveryInfo(lat, 31.36, "Cairo", "")
		if !errors.Is(err, ErrInvalidLatitude) {
			t.Errorf("lat %f: error = %v, want ErrInvalidLatitude", lat, err)
		}
	}
}

func TestNewDeliveryInfo_InvalidLongitude(t *testing.T) {
	tests := []float64{-181, 181, -360, 360}
	for _, lng := range tests {
		_, err := NewDeliveryInfo(30.05, lng, "Cairo", "")
		if !errors.Is(err, ErrInvalidLongitude) {
			t.Errorf("lng %f: error = %v, want ErrInvalidLongitude", lng, err)
		}
	}
}

func TestNewDeliveryInfo_EmptyAddress(t *testing.T) {
	_, err := NewDeliveryInfo(30.05, 31.36, "", "")
	if !errors.Is(err, ErrDeliveryAddressRequired) {
		t.Errorf("error = %v, want ErrDeliveryAddressRequired", err)
	}
}

func TestNewDeliveryInfo_BoundaryCoordinates(t *testing.T) {
	// Valid boundaries: lat -90/90, lng -180/180.
	_, err := NewDeliveryInfo(-90, -180, "South Pole", "")
	if err != nil {
		t.Fatalf("boundary (-90, -180) error: %v", err)
	}
	_, err = NewDeliveryInfo(90, 180, "North Pole", "")
	if err != nil {
		t.Fatalf("boundary (90, 180) error: %v", err)
	}
}

func TestDeliveryInfo_IsZero(t *testing.T) {
	var zero DeliveryInfo
	if !zero.IsZero() {
		t.Error("zero DeliveryInfo should be IsZero")
	}
	info, _ := NewDeliveryInfo(30, 31, "Cairo", "")
	if info.IsZero() {
		t.Error("non-zero DeliveryInfo should not be IsZero")
	}
}
