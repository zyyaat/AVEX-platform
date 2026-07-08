// Package domain tests: Role + Permission + UserRole + RolePermission.
package domain

import (
	"testing"
	"time"
)

func testNowPerms() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

// ===== Role Tests =====

func TestNewRole(t *testing.T) {
	now := testNowPerms()
	tests := []struct {
		name        string
		id          string
		roleName    string
		description string
		isSystem    bool
		wantErr     error
	}{
		{"valid admin", "r1", "admin", "Administrator", true, nil},
		{"valid custom", "r2", "manager", "Manager role", false, nil},
		{"empty id", "", "x", "", false, ErrInvalidID},
		{"empty name", "r3", "", "", false, ErrEmptyRoleName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRole(tt.id, tt.roleName, tt.description, tt.isSystem, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.IsSystem() != tt.isSystem {
				t.Errorf("expected isSystem=%v, got %v", tt.isSystem, r.IsSystem())
			}
		})
	}
}

// ===== Permission Tests =====

func TestNewPermission(t *testing.T) {
	now := testNowPerms()
	tests := []struct {
		name     string
		id       string
		permName string
		wantErr  error
	}{
		{"valid", "p1", "orders.order.read", nil},
		{"valid wildcard", "p2", "admin.*.*", nil},
		{"valid action wildcard", "p3", "orders.order.*", nil},
		{"empty id", "", "x.y.z", ErrInvalidID},
		{"empty name", "p4", "", ErrEmptyPermissionName},
		{"two parts", "p5", "orders.read", ErrInvalidPermissionFormat},
		{"four parts", "p6", "a.b.c.d", ErrInvalidPermissionFormat},
		{"empty part", "p7", "orders..read", ErrInvalidPermissionFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPermission(tt.id, tt.permName, "desc", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name() != tt.permName {
				t.Errorf("expected name %q, got %q", tt.permName, p.Name())
			}
		})
	}
}

func TestPermissionMatches(t *testing.T) {
	now := testNowPerms()
	tests := []struct {
		name     string
		perm     string
		required string
		want     bool
	}{
		{"exact match", "orders.order.read", "orders.order.read", true},
		{"wildcard action", "orders.order.*", "orders.order.read", true},
		{"wildcard resource+action", "orders.*.*", "orders.order.write", true},
		{"full wildcard", "*.*.*", "anything.any.thing", true},
		{"wrong module", "orders.order.read", "financial.order.read", false},
		{"wrong resource", "orders.order.read", "orders.wallet.read", false},
		{"wrong action", "orders.order.read", "orders.order.write", false},
		{"wildcard module doesn't match", "admin.*.*", "orders.order.read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPermission("p1", tt.perm, "", now)
			if err != nil {
				t.Fatalf("create permission: %v", err)
			}
			if got := p.Matches(tt.required); got != tt.want {
				t.Errorf("Matches(%q, %q) = %v, want %v", tt.perm, tt.required, got, tt.want)
			}
		})
	}
}

// ===== UserRole Tests =====

func TestNewUserRole(t *testing.T) {
	now := testNowPerms()
	tests := []struct {
		name       string
		id         string
		userID     string
		roleID     string
		assignedBy string
		wantErr    error
	}{
		{"valid", "ur1", "u1", "r1", "admin-1", nil},
		{"empty id", "", "u1", "r1", "a1", ErrInvalidID},
		{"empty user id", "ur2", "", "r1", "a1", ErrEmptyUserID},
		{"empty role id", "ur3", "u1", "", "a1", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUserRole(tt.id, tt.userID, tt.roleID, tt.assignedBy, now)
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

// ===== RolePermission Tests =====

func TestNewRolePermission(t *testing.T) {
	now := testNowPerms()
	tests := []struct {
		name         string
		id           string
		roleID       string
		permissionID string
		wantErr      error
	}{
		{"valid", "rp1", "r1", "p1", nil},
		{"empty id", "", "r1", "p1", ErrInvalidID},
		{"empty role id", "rp2", "", "p1", ErrInvalidInput},
		{"empty permission id", "rp3", "r1", "", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRolePermission(tt.id, tt.roleID, tt.permissionID, now)
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

// ===== System Role Tests =====

func TestIsSystemRole(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"admin", true},
		{"agent", true},
		{"merchant", true},
		{"driver", true},
		{"user", true},
		{"manager", false},
		{"supervisor", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSystemRole(tt.name); got != tt.want {
				t.Errorf("IsSystemRole(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
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
