// Package domain tests: Restaurant entity.
package domain

import (
	"errors"
	"testing"
	"time"
)

var catNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func TestNewRestaurant_Success(t *testing.T) {
	r, err := NewRestaurant(RestaurantParams{
		ID: "rest-001", Name: "Burger House", Lat: 30.05, Lng: 31.36,
		ZoneID: "zone-nasr", IsActive: true, Now: catNow,
	})
	if err != nil {
		t.Fatalf("NewRestaurant: %v", err)
	}
	if r.ID() != "rest-001" {
		t.Errorf("ID = %q", r.ID())
	}
	if !r.IsActive() {
		t.Error("should be active")
	}
}

func TestNewRestaurant_EmptyID(t *testing.T) {
	_, err := NewRestaurant(RestaurantParams{Name: "Test", Lat: 30, Lng: 31, Now: catNow})
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("error = %v, want ErrInvalidID", err)
	}
}

func TestNewRestaurant_EmptyName(t *testing.T) {
	_, err := NewRestaurant(RestaurantParams{ID: "r1", Lat: 30, Lng: 31, Now: catNow})
	if !errors.Is(err, ErrNameRequired) {
		t.Errorf("error = %v, want ErrNameRequired", err)
	}
}

func TestNewRestaurant_InvalidLat(t *testing.T) {
	_, err := NewRestaurant(RestaurantParams{ID: "r1", Name: "Test", Lat: -91, Lng: 31, Now: catNow})
	if !errors.Is(err, ErrInvalidLatitude) {
		t.Errorf("error = %v", err)
	}
}

func TestRestaurant_ActivateDeactivate(t *testing.T) {
	r, _ := NewRestaurant(RestaurantParams{ID: "r1", Name: "Test", Lat: 30, Lng: 31, IsActive: false, Now: catNow})
	r.Activate(catNow)
	if !r.IsActive() {
		t.Error("should be active")
	}
	r.Deactivate(catNow)
	if r.IsActive() {
		t.Error("should be inactive")
	}
}

func TestNewMenuItem_Success(t *testing.T) {
	m, err := NewMenuItem(MenuItemParams{ID: "m1", RestaurantID: "r1", Name: "Burger", Price: 12.99, IsAvailable: true, Now: catNow})
	if err != nil {
		t.Fatalf("NewMenuItem: %v", err)
	}
	if m.Price() != 12.99 {
		t.Errorf("Price = %v", m.Price())
	}
}

func TestNewMenuItem_NegativePrice(t *testing.T) {
	_, err := NewMenuItem(MenuItemParams{ID: "m1", RestaurantID: "r1", Name: "Burger", Price: -1, Now: catNow})
	if !errors.Is(err, ErrInvalidPrice) {
		t.Errorf("error = %v", err)
	}
}

func TestMenuItem_SetAvailable(t *testing.T) {
	m, _ := NewMenuItem(MenuItemParams{ID: "m1", RestaurantID: "r1", Name: "Burger", Price: 10, IsAvailable: true, Now: catNow})
	m.SetAvailable(false, catNow)
	if m.IsAvailable() {
		t.Error("should be unavailable")
	}
}

func TestNewCategory_Success(t *testing.T) {
	c, err := NewCategory(CategoryParams{ID: "c1", Name: "Burgers", Now: catNow})
	if err != nil {
		t.Fatalf("NewCategory: %v", err)
	}
	if c.Name() != "Burgers" {
		t.Errorf("Name = %q", c.Name())
	}
}

func TestNewStoreHours_IsOpenNow(t *testing.T) {
	sh, _ := NewStoreHours("sh1", "r1", 4, "10:00", "23:00", true) // Thursday
	// catNow is 2026-01-15 which is Thursday
	if !sh.IsOpenNow(catNow) {
		t.Error("should be open at 12:00 on Thursday")
	}
	// Check closed time
	closedTime := time.Date(2026, 1, 15, 1, 0, 0, 0, time.UTC) // 1:00 AM Thursday
	if sh.IsOpenNow(closedTime) {
		t.Error("should be closed at 1:00 AM")
	}
}

func TestNewStoreHours_DifferentDay(t *testing.T) {
	sh, _ := NewStoreHours("sh1", "r1", 0, "10:00", "23:00", true) // Sunday
	// catNow is Thursday
	if sh.IsOpenNow(catNow) {
		t.Error("should not be open — different day")
	}
}
