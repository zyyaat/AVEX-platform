// Package domain order_item: OrderItem value object.
//
// An OrderItem is an immutable snapshot of a menu item at the time of ordering.
// It captures the item's ID, name (EN + AR), price, and quantity.
// The snapshot ensures historical accuracy — if the menu item's price changes
// later, existing orders retain the original price.
//
// Imports stdlib only.
package domain

import "fmt"

// OrderItem is an immutable snapshot of a menu item in an order.
type OrderItem struct {
	menuItemID string
	name       string
	nameAr     string
	price      Money
	quantity   int
}

// NewOrderItem creates an OrderItem with validation.
// menuItemID must be non-empty.
// name must be non-empty.
// price must be a valid Money (non-negative).
// quantity must be > 0.
func NewOrderItem(menuItemID, name, nameAr string, price Money, quantity int) (OrderItem, error) {
	if menuItemID == "" {
		return OrderItem{}, NewValidationError("menu_item_id", ErrInvalidID)
	}
	if name == "" {
		return OrderItem{}, NewValidationError("name", ErrInvalidInput)
	}
	if quantity <= 0 {
		return OrderItem{}, NewValidationError("quantity", fmt.Errorf("%w: %d", ErrInvalidQuantity, quantity))
	}
	// Price amount is already validated by NewMoney, but we double-check.
	if price.Amount() < 0 {
		return OrderItem{}, NewValidationError("price", ErrInvalidMoneyAmount)
	}

	return OrderItem{
		menuItemID: menuItemID,
		name:       name,
		nameAr:     nameAr,
		price:      price,
		quantity:   quantity,
	}, nil
}

// MenuItemID returns the menu item ID (soft ref to catalog.menu_items).
func (i OrderItem) MenuItemID() string {
	return i.menuItemID
}

// Name returns the English name.
func (i OrderItem) Name() string {
	return i.name
}

// NameAr returns the Arabic name.
func (i OrderItem) NameAr() string {
	return i.nameAr
}

// Price returns the unit price as a Money value.
func (i OrderItem) Price() Money {
	return i.price
}

// Quantity returns the ordered quantity.
func (i OrderItem) Quantity() int {
	return i.quantity
}

// LineTotal returns the total price for this item (price × quantity).
// Returns an error if the multiplication fails (should never happen with valid inputs).
func (i OrderItem) LineTotal() (Money, error) {
	return i.price.Multiply(i.quantity)
}

// String returns a log-safe representation.
func (i OrderItem) String() string {
	return fmt.Sprintf("OrderItem{item=%s, name=%s, qty=%d, price=%s}",
		i.menuItemID, i.name, i.quantity, i.price.String())
}
