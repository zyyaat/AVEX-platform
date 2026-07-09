// Package domain tests: Language + Translation.
package domain

import (
	"testing"
	"time"
)

func testNowLoc() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

func TestNewLanguage(t *testing.T) {
	now := testNowLoc()
	tests := []struct {
		name      string
		id        string
		code      string
		nameStr   string
		isRTL     bool
		isDefault bool
		wantErr   error
	}{
		{"valid en", "l1", "en", "English", false, true, nil},
		{"valid ar RTL", "l2", "ar", "العربية", true, false, nil},
		{"valid fr", "l3", "fr", "Français", false, false, nil},
		{"empty id", "", "en", "English", false, false, ErrInvalidID},
		{"invalid code 1 char", "l4", "e", "English", false, false, ErrInvalidLanguageCode},
		{"invalid code 3 chars", "l5", "eng", "English", false, false, ErrInvalidLanguageCode},
		{"empty name", "l6", "en", "", false, false, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := NewLanguage(tt.id, tt.code, tt.nameStr, tt.isRTL, tt.isDefault, true, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil { t.Fatalf("unexpected error: %v", err) }
			if l.Code() != tt.code { t.Errorf("expected code %q, got %q", tt.code, l.Code()) }
			if l.IsRTL() != tt.isRTL { t.Errorf("expected isRTL=%v", tt.isRTL) }
		})
	}
}

func TestLanguageCodeLowercased(t *testing.T) {
	now := testNowLoc()
	l, _ := NewLanguage("l1", "EN", "English", false, true, true, now)
	if l.Code() != "en" {
		t.Errorf("expected lowercased 'en', got %q", l.Code())
	}
}

func TestNewTranslation(t *testing.T) {
	now := testNowLoc()
	tests := []struct {
		name      string
		id        string
		langCode  string
		key       string
		value     string
		wantErr   error
	}{
		{"valid", "t1", "en", "orders.status.pending", "Pending", nil},
		{"valid ar", "t2", "ar", "orders.status.pending", "قيد الانتظار", nil},
		{"empty id", "", "en", "key", "val", ErrInvalidID},
		{"empty lang code", "t3", "", "key", "val", ErrInvalidInput},
		{"empty key", "t4", "en", "", "val", ErrEmptyKey},
		{"empty value", "t5", "en", "key", "", ErrEmptyValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTranslation(tt.id, tt.langCode, tt.key, tt.value, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil { t.Fatalf("unexpected error: %v", err) }
		})
	}
}

func TestTranslationSetValue(t *testing.T) {
	now := testNowLoc()
	t1, _ := NewTranslation("t1", "en", "key", "old value", now)
	updated, err := t1.SetValue("new value", now)
	if err != nil { t.Fatalf("set value: %v", err) }
	if updated.Value() != "new value" { t.Errorf("expected 'new value', got %q", updated.Value()) }

	// Empty value
	_, err = t1.SetValue("", now)
	if !errIs(err, ErrEmptyValue) { t.Fatalf("expected ErrEmptyValue, got %v", err) }
}

func errIs(err, target error) bool {
	if err == target { return true }
	for {
		type u interface{ Unwrap() error }
		un, ok := err.(u)
		if !ok { return false }
		err = un.Unwrap()
		if err == target { return true }
		if err == nil { return false }
	}
}
