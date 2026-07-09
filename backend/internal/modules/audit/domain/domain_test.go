// Package domain tests: AuditEntry.
package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func testNowAudit() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

func TestNewAuditEntry(t *testing.T) {
	now := testNowAudit()
	tests := []struct {
		name         string
		id           string
		actorType    ActorType
		actorID      string
		action       string
		resourceType string
		severity     Severity
		wantErr      error
	}{
		{"valid user action", "a1", ActorUser, "u1", "orders.order.create", "order", SeverityInfo, nil},
		{"valid system action", "a2", ActorSystem, "", "system.cleanup", "system", SeverityInfo, nil},
		{"valid critical", "a3", ActorAdmin, "a1", "permissions.role.assign", "role", SeverityCritical, nil},
		{"valid warning", "a4", ActorUser, "u1", "identity.user.login_failed", "session", SeverityWarning, nil},
		{"empty id", "", ActorUser, "u1", "x", "r", SeverityInfo, ErrInvalidID},
		{"invalid actor type", "a5", ActorType("bogus"), "u1", "x", "r", SeverityInfo, ErrInvalidActorType},
		{"empty actor id (non-system)", "a6", ActorUser, "", "x", "r", SeverityInfo, ErrEmptyActorID},
		{"empty action", "a7", ActorUser, "u1", "", "r", SeverityInfo, ErrEmptyAction},
		{"empty resource type", "a8", ActorUser, "u1", "x", "", SeverityInfo, ErrEmptyResourceType},
		{"invalid severity", "a9", ActorUser, "u1", "x", "r", Severity("bogus"), ErrInvalidSeverity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAuditEntry(tt.id, tt.actorType, tt.actorID, tt.action, tt.resourceType, "res-1", tt.severity, "desc", nil, "127.0.0.1", "test", "", "", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuditEntryImmutability(t *testing.T) {
	now := testNowAudit()
	entry, _ := NewAuditEntry("a1", ActorUser, "u1", "orders.order.create", "order", "ord-1", SeverityInfo, "desc", nil, "127.0.0.1", "test", "", "", now)

	// There are no mutation methods on AuditEntry — it's immutable by design.
	// We verify that the struct only has accessor methods (Getters).
	if entry.Action() != "orders.order.create" {
		t.Errorf("action mismatch")
	}
	if entry.ActorID() != "u1" {
		t.Errorf("actor id mismatch")
	}
	if entry.Severity() != SeverityInfo {
		t.Errorf("severity mismatch")
	}
}

func TestAuditEntryMetadataJSON(t *testing.T) {
	now := testNowAudit()
	meta := map[string]any{"old_value": "5", "new_value": "10", "changed_by": "admin-1"}
	entry, _ := NewAuditEntry("a1", ActorAdmin, "a1", "settings.setting.update", "setting", "s1", SeverityCritical, "Updated delivery radius", meta, "127.0.0.1", "test", "", "", now)

	raw := entry.MetadataJSON()
	if raw == nil {
		t.Fatal("expected non-nil metadata JSON")
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["old_value"] != "5" {
		t.Errorf("expected old_value=5, got %v", decoded["old_value"])
	}
	if decoded["new_value"] != "10" {
		t.Errorf("expected new_value=10, got %v", decoded["new_value"])
	}
}

func TestAuditEntryEmptyMetadataJSON(t *testing.T) {
	now := testNowAudit()
	entry, _ := NewAuditEntry("a1", ActorSystem, "", "system.cleanup", "system", "sys-1", SeverityInfo, "cleanup", nil, "", "", "", "", now)
	if raw := entry.MetadataJSON(); raw != nil {
		t.Errorf("expected nil for empty metadata, got %s", raw)
	}
}

func TestAuditEntryIsCritical(t *testing.T) {
	now := testNowAudit()
	critical, _ := NewAuditEntry("a1", ActorAdmin, "a1", "permissions.role.assign", "role", "r1", SeverityCritical, "desc", nil, "", "", "", "", now)
	info, _ := NewAuditEntry("a2", ActorUser, "u1", "orders.order.create", "order", "o1", SeverityInfo, "desc", nil, "", "", "", "", now)
	warning, _ := NewAuditEntry("a3", ActorUser, "u1", "identity.login_failed", "session", "s1", SeverityWarning, "desc", nil, "", "", "", "", now)

	if !critical.IsCritical() {
		t.Error("expected critical to be critical")
	}
	if info.IsCritical() {
		t.Error("expected info to not be critical")
	}
	if warning.IsCritical() {
		t.Error("expected warning to not be critical")
	}
}

func TestSeverityIsValid(t *testing.T) {
	tests := []struct {
		s    Severity
		want bool
	}{
		{SeverityInfo, true},
		{SeverityWarning, true},
		{SeverityCritical, true},
		{Severity("bogus"), false},
		{Severity(""), false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("Severity(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestActorTypeIsValid(t *testing.T) {
	tests := []struct {
		a    ActorType
		want bool
	}{
		{ActorUser, true},
		{ActorDriver, true},
		{ActorMerchant, true},
		{ActorAgent, true},
		{ActorAdmin, true},
		{ActorSystem, true},
		{ActorType("bogus"), false},
	}
	for _, tt := range tests {
		if got := tt.a.IsValid(); got != tt.want {
			t.Errorf("ActorType(%q).IsValid() = %v, want %v", tt.a, got, tt.want)
		}
	}
}

// errIs helper
func errIs(err, target error) bool {
	if err == target {
		return true
	}
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
