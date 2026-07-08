// Package domain tests: Money value object.
package domain

import "testing"

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		wantErr  error
	}{
		{"valid", 1000, "EGP", nil},
		{"zero", 0, "EGP", nil},
		{"negative", -1, "EGP", ErrInvalidMoneyAmount},
		{"short currency", 100, "EG", ErrInvalidCurrency},
		{"long currency", 100, "EGYPT", ErrInvalidCurrency},
		{"empty currency", 100, "", ErrInvalidCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMoney(tt.amount, tt.currency)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.Amount() != tt.amount {
				t.Errorf("amount: expected %d, got %d", tt.amount, m.Amount())
			}
			if m.Currency() != tt.currency {
				t.Errorf("currency: expected %s, got %s", tt.currency, m.Currency())
			}
		})
	}
}

func TestMoneyAdd(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(500, "EGP")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Amount() != 1500 {
		t.Errorf("expected 1500, got %d", sum.Amount())
	}
	// Immutability: original unchanged
	if a.Amount() != 1000 {
		t.Errorf("original a mutated: %d", a.Amount())
	}

	// Currency mismatch
	c, _ := NewMoney(1000, "USD")
	_, err = a.Add(c)
	if !errIs(err, ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestMoneySubtract(t *testing.T) {
	a, _ := NewMoney(1000, "EGP")
	b, _ := NewMoney(300, "EGP")
	diff, err := a.Subtract(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.Amount() != 700 {
		t.Errorf("expected 700, got %d", diff.Amount())
	}

	// Negative result
	c, _ := NewMoney(100, "EGP")
	_, err = c.Subtract(a)
	if !errIs(err, ErrNegativeMoneyResult) {
		t.Fatalf("expected ErrNegativeMoneyResult, got %v", err)
	}
}

func TestMoneyMultiply(t *testing.T) {
	m, _ := NewMoney(500, "EGP")
	prod, err := m.Multiply(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prod.Amount() != 1500 {
		t.Errorf("expected 1500, got %d", prod.Amount())
	}

	// Zero / negative quantity
	_, err = m.Multiply(0)
	if !errIs(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestMoneyMultiplyFloat(t *testing.T) {
	m, _ := NewMoney(1000, "EGP")
	// 1.5x surge: numerator=3, denominator=2
	surged, err := m.MultiplyFloat(3, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if surged.Amount() != 1500 {
		t.Errorf("expected 1500, got %d", surged.Amount())
	}

	// 1.25x surge: 5/4
	surged2, _ := m.MultiplyFloat(5, 4)
	if surged2.Amount() != 1250 {
		t.Errorf("expected 1250, got %d", surged2.Amount())
	}

	// Zero numerator
	zero, _ := m.MultiplyFloat(0, 1)
	if !zero.IsZero() {
		t.Errorf("expected zero, got %d", zero.Amount())
	}

	// Negative numerator
	_, err = m.MultiplyFloat(-1, 1)
	if !errIs(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPercentage(t *testing.T) {
	base, _ := NewMoney(1000, "EGP")
	// 15% of 1000 = 150
	discount, err := Percentage(base, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if discount.Amount() != 150 {
		t.Errorf("expected 150, got %d", discount.Amount())
	}

	// 0% and 100%
	zero, _ := Percentage(base, 0)
	if zero.Amount() != 0 {
		t.Errorf("expected 0, got %d", zero.Amount())
	}
	full, _ := Percentage(base, 100)
	if full.Amount() != 1000 {
		t.Errorf("expected 1000, got %d", full.Amount())
	}

	// Out of range
	_, err = Percentage(base, -1)
	if !errIs(err, ErrInvalidPercentage) {
		t.Fatalf("expected ErrInvalidPercentage, got %v", err)
	}
	_, err = Percentage(base, 101)
	if !errIs(err, ErrInvalidPercentage) {
		t.Fatalf("expected ErrInvalidPercentage, got %v", err)
	}
}

func TestMoneyLessThan(t *testing.T) {
	a, _ := NewMoney(100, "EGP")
	b, _ := NewMoney(200, "EGP")
	yes, err := a.LessThan(b)
	if err != nil || !yes {
		t.Fatalf("expected a < b, got yes=%v err=%v", yes, err)
	}

	c, _ := NewMoney(100, "USD")
	_, err = a.LessThan(c)
	if !errIs(err, ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestMin(t *testing.T) {
	a, _ := NewMoney(100, "EGP")
	b, _ := NewMoney(200, "EGP")
	m, err := Min(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Amount() != 100 {
		t.Errorf("expected 100, got %d", m.Amount())
	}
}

// errIs is a helper that uses errors.Is but avoids importing errors in domain tests.
func errIs(err, target error) bool {
	if err == target {
		return true
	}
	// Walk Unwrap chain manually
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
