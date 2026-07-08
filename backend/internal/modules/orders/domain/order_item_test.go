// Package domain tests: OrderItem value object.
package domain

import (
	"errors"
	"testing"
)

func TestNewOrderItem_Success(t *testing.T) {
	price, _ := NewMoney(1299, "EGP")
	item, err := NewOrderItem("item-001", "Burger", "برجر", price, 2)
	if err != nil {
		t.Fatalf("NewOrderItem error: %v", err)
	}
	if item.MenuItemID() != "item-001" {
		t.Errorf("MenuItemID = %q", item.MenuItemID())
	}
	if item.Name() != "Burger" {
		t.Errorf("Name = %q", item.Name())
	}
	if item.NameAr() != "برجر" {
		t.Errorf("NameAr = %q", item.NameAr())
	}
	if item.Quantity() != 2 {
		t.Errorf("Quantity = %d", item.Quantity())
	}
	if item.Price().Amount() != 1299 {
		t.Errorf("Price = %d", item.Price().Amount())
	}
}

func TestNewOrderItem_LineTotal(t *testing.T) {
	price, _ := NewMoney(500, "EGP")
	item, _ := NewOrderItem("item-001", "Fries", "بطاطس", price, 3)
	total, err := item.LineTotal()
	if err != nil {
		t.Fatalf("LineTotal error: %v", err)
	}
	if total.Amount() != 1500 {
		t.Errorf("LineTotal = %d, want 1500", total.Amount())
	}
}

func TestNewOrderItem_EmptyMenuItemID(t *testing.T) {
	price, _ := NewMoney(500, "EGP")
	_, err := NewOrderItem("", "Fries", "بطاطس", price, 1)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("error = %v, want ErrInvalidID", err)
	}
}

func TestNewOrderItem_EmptyName(t *testing.T) {
	price, _ := NewMoney(500, "EGP")
	_, err := NewOrderItem("item-001", "", "بطاطس", price, 1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

func TestNewOrderItem_ZeroQuantity(t *testing.T) {
	price, _ := NewMoney(500, "EGP")
	_, err := NewOrderItem("item-001", "Fries", "بطاطس", price, 0)
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("error = %v, want ErrInvalidQuantity", err)
	}
}

func TestNewOrderItem_NegativeQuantity(t *testing.T) {
	price, _ := NewMoney(500, "EGP")
	_, err := NewOrderItem("item-001", "Fries", "بطاطس", price, -1)
	if !errors.Is(err, ErrInvalidQuantity) {
		t.Errorf("error = %v, want ErrInvalidQuantity", err)
	}
}
