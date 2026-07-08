// Package domain tests: Money value object — arithmetic, validation, percentage.
package domain

import (
	"errors"
	"testing"
)

func TestNewMoney_Valid(t *testing.T) {
	m, err := NewMoney(1299, "EGP")
	if err != nil {
		t.Fatalf("NewMoney error: %v", err)
	}
	if m.Amount() != 1299 {
		t.Errorf("Amount = %d, want 1299", m.Amount())
	}
	if m.Currency() != "EGP" {
		t.Errorf("Currency = %q", m.Currency())
	}
}

func TestNewMoney_NegativeAmount(t *testing.T) {
	_, err := NewMoney(-1, "EGP")
	if !errors.Is(err, ErrInvalidMoneyAmount) {
		t.Errorf("error = %v, want ErrInvalidMoneyAmount", err)
	}
}

func TestNewMoney_InvalidCurrency(t *testing.T) {
	tests := []string{"", "EG", "EGPPI", "egypt"}
	for _, c := range tests {
		_, err := NewMoney(100, c)
		if !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("currency %q: error = %v, want ErrInvalidCurrency", c, err)
		}
	}
}

func TestNewMoney_ZeroAmount(t *testing.T) {
	m, err := NewMoney(0, "USD")
	if err != nil {
		t.Fatalf("NewMoney(0) error: %v", err)
	}
	if !m.IsZero() {
		t.Error("should be zero")
	}
	if m.IsPositive() {
		t.Error("should not be positive")
	}
}

func TestZeroMoney(t *testing.T) {
	m := ZeroMoney("EGP")
	if !m.IsZero() {
		t.Error("ZeroMoney should be zero")
	}
	if m.Currency() != "EGP" {
		t.Errorf("Currency = %q", m.Currency())
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(500, "EGP")
	result, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	if result.Amount() != 1500 {
		t.Errorf("Add result = %d, want 1500", result.Amount())
	}
}

func TestMoney_Add_DifferentCurrency(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(500, "USD")
	_, err := a.Add(b)
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("error = %v, want ErrCurrencyMismatch", err)
	}
}

func TestMoney_Subtract_SameCurrency(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(300, "EGP")
	result, err := a.Subtract(b)
	if err != nil {
		t.Fatalf("Subtract error: %v", err)
	}
	if result.Amount() != 700 {
		t.Errorf("Subtract result = %d, want 700", result.Amount())
	}
}

func TestMoney_Subtract_NegativeResult(t *testing.T) {
	a, _ := NewMoney(100, "EGP")
	b, _ := NewMoney(300, "EGP")
	_, err := a.Subtract(b)
	if !errors.Is(err, ErrNegativeMoneyResult) {
		t.Errorf("error = %v, want ErrNegativeMoneyResult", err)
	}
}

func TestMoney_Multiply(t *testing.T) {
	m, _ := NewMoney(500, "EGP")
	result, err := m.Multiply(3)
	if err != nil {
		t.Fatalf("Multiply error: %v", err)
	}
	if result.Amount() != 1500 {
		t.Errorf("Multiply result = %d, want 1500", result.Amount())
	}
}

func TestMoney_Multiply_ZeroOrNegative(t *testing.T) {
	m, _ := NewMoney(500, "EGP")
	_, err := m.Multiply(0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Multiply(0) error = %v, want ErrInvalidInput", err)
	}
	_, err = m.Multiply(-1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Multiply(-1) error = %v, want ErrInvalidInput", err)
	}
}

func TestPercentage_Valid(t *testing.T) {
	base, _ := NewMoney(1000, "EGP")

	tests := []struct {
		percent int
		want    int64
	}{
		{0, 0},
		{10, 100},
		{15, 150},
		{50, 500},
		{100, 1000},
	}
	for _, tt := range tests {
		result, err := Percentage(base, tt.percent)
		if err != nil {
			t.Fatalf("Percentage(%d%%) error: %v", tt.percent, err)
		}
		if result.Amount() != tt.want {
			t.Errorf("Percentage(%d%%) = %d, want %d", tt.percent, result.Amount(), tt.want)
		}
	}
}

func TestPercentage_Invalid(t *testing.T) {
	base, _ := NewMoney(1000, "EGP")
	_, err := Percentage(base, -1)
	if !errors.Is(err, ErrInvalidPercentage) {
		t.Errorf("Percentage(-1) error = %v, want ErrInvalidPercentage", err)
	}
	_, err = Percentage(base, 101)
	if !errors.Is(err, ErrInvalidPercentage) {
		t.Errorf("Percentage(101) error = %v, want ErrInvalidPercentage", err)
	}
}

func TestPercentage_IntegerRounding(t *testing.T) {
	// 999 cents * 15% = 149.85 → should round down to 149 (integer math)
	base, _ := NewMoney(999, "EGP")
	result, err := Percentage(base, 15)
	if err != nil {
		t.Fatalf("Percentage error: %v", err)
	}
	if result.Amount() != 149 {
		t.Errorf("Percentage(999, 15%%) = %d, want 149 (integer rounding)", result.Amount())
	}
}

func TestMoney_Equals(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(1000, "EGP")
	c, _ := NewMoney(1000, "USD")
	d, _ := NewMoney(500, "EGP")

	if !a.Equals(b) {
		t.Error("a should equal b")
	}
	if a.Equals(c) {
		t.Error("a should not equal c (different currency)")
	}
	if a.Equals(d) {
		t.Error("a should not equal d (different amount)")
	}
}

func TestMoney_LessThan(t *testing.T) {
	a, _ := NewMoney(500, "EGP")
	b, _ := NewMoney(1000, "EGP")

	less, err := a.LessThan(b)
	if err != nil {
		t.Fatalf("LessThan error: %v", err)
	}
	if !less {
		t.Error("500 should be less than 1000")
	}

	less, err = b.LessThan(a)
	if err != nil {
		t.Fatalf("LessThan error: %v", err)
	}
	if less {
		t.Error("1000 should not be less than 500")
	}
}

func TestMoney_LessThan_DifferentCurrency(t *testing.T) {
	a, _ := NewMoney(500, "EGP")
	b, _ := NewMoney(1000, "USD")
	_, err := a.LessThan(b)
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("error = %v, want ErrCurrencyMismatch", err)
	}
}
