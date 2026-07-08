// Package domain store_hours: operating hours per day of week.
package domain

import "time"

type StoreHours struct {
	id           string
	restaurantID string
	dayOfWeek    int    // 0=Sunday, 1=Monday, ..., 6=Saturday
	openTime     string // "10:00"
	closeTime    string // "23:00"
	isOpen       bool
}

func NewStoreHours(id, restaurantID string, dayOfWeek int, openTime, closeTime string, isOpen bool) (StoreHours, error) {
	if id == "" {
		return StoreHours{}, NewValidationError("id", ErrInvalidID)
	}
	if restaurantID == "" {
		return StoreHours{}, NewValidationError("restaurant_id", ErrInvalidInput)
	}
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return StoreHours{}, NewValidationError("day_of_week", ErrInvalidInput)
	}
	return StoreHours{id: id, restaurantID: restaurantID, dayOfWeek: dayOfWeek, openTime: openTime, closeTime: closeTime, isOpen: isOpen}, nil
}

type StoreHoursRecord struct {
	ID, RestaurantID    string
	DayOfWeek           int
	OpenTime, CloseTime string
	IsOpen              bool
}

func ReconstructStoreHours(r StoreHoursRecord) StoreHours {
	return StoreHours{id: r.ID, restaurantID: r.RestaurantID, dayOfWeek: r.DayOfWeek, openTime: r.OpenTime, closeTime: r.CloseTime, isOpen: r.IsOpen}
}

func (s StoreHours) ID() string           { return s.id }
func (s StoreHours) RestaurantID() string { return s.restaurantID }
func (s StoreHours) DayOfWeek() int       { return s.dayOfWeek }
func (s StoreHours) OpenTime() string     { return s.openTime }
func (s StoreHours) CloseTime() string    { return s.closeTime }
func (s StoreHours) IsOpen() bool         { return s.isOpen }

// IsOpenNow checks if the store is currently open based on the given time.
func (s StoreHours) IsOpenNow(now time.Time) bool {
	if !s.isOpen {
		return false
	}
	weekday := int(now.Weekday())
	if weekday != s.dayOfWeek {
		return false
	}
	// Simple string comparison works for "HH:MM" format.
	currentTime := now.Format("15:04")
	return currentTime >= s.openTime && currentTime <= s.closeTime
}
