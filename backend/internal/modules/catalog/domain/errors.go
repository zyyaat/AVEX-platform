// Package domain errors: typed domain errors for the catalog module.
package domain

import (
	"errors"
	"fmt"
)

var ErrRestaurantNotFound = errors.New("restaurant not found")
var ErrRestaurantAlreadyExists = errors.New("restaurant already exists")
var ErrMenuItemNotFound = errors.New("menu item not found")
var ErrMenuItemAlreadyExists = errors.New("menu item already exists")
var ErrCategoryNotFound = errors.New("category not found")
var ErrCategoryAlreadyExists = errors.New("category already exists")
var ErrRestaurantInactive = errors.New("restaurant is inactive")
var ErrMenuItemUnavailable = errors.New("menu item is unavailable")
var ErrRestaurantClosed = errors.New("restaurant is currently closed")
var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrNameRequired = errors.New("name is required")
var ErrInvalidPrice = errors.New("invalid price")
var ErrInvalidLatitude = errors.New("invalid latitude")
var ErrInvalidLongitude = errors.New("invalid longitude")

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

func (e *ValidationError) Unwrap() error { return e.Wrapped }

func NewValidationError(field string, err error) *ValidationError {
	return &ValidationError{Field: field, Wrapped: err}
}
