// Package domain: Permissions module errors + types.
//
// RBAC (Role-Based Access Control) model:
//   User → UserRole → Role → RolePermission → Permission
//
// A user can have multiple roles. A role can have multiple permissions.
// The effective permissions of a user = union of all permissions from all
// their roles.
//
// Permissions use a hierarchical naming convention: "module.resource.action"
// Examples:
//   "orders.order.read"
//   "orders.order.write"
//   "financial.wallet.credit"
//   "dispatch.driver.suspend"
//   "support.ticket.assign"
//   "admin.*" (wildcard — grants all permissions in the admin module)
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ===== Errors =====

var ErrRoleNotFound = errors.New("role not found")
var ErrRoleAlreadyExists = errors.New("role already exists")
var ErrPermissionNotFound = errors.New("permission not found")
var ErrPermissionAlreadyExists = errors.New("permission already exists")
var ErrRolePermissionNotFound = errors.New("role-permission mapping not found")
var ErrRolePermissionAlreadyExists = errors.New("role already has this permission")
var ErrUserRoleNotFound = errors.New("user-role mapping not found")
var ErrUserRoleAlreadyExists = errors.New("user already has this role")
var ErrCannotModifySystemRole = errors.New("cannot modify system-defined roles")
var ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin role from a user")

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyRoleName = errors.New("role name is required")
var ErrEmptyPermissionName = errors.New("permission name is required")
var ErrEmptyUserID = errors.New("user id is required")
var ErrInvalidPermissionFormat = errors.New("invalid permission format (must be module.resource.action)")

// ===== Role =====

// Role represents a named collection of permissions (e.g. "admin", "agent", "driver").
type Role struct {
	id          string
	name        string
	description string
	isSystem    bool // system roles cannot be deleted
	createdAt   time.Time
	updatedAt   time.Time
}

// NewRole creates a new Role with validation.
func NewRole(id, name, description string, isSystem bool, now time.Time) (Role, error) {
	if id == "" {
		return Role{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if name == "" {
		return Role{}, ErrEmptyRoleName
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Role{
		id:          id,
		name:        name,
		description: description,
		isSystem:    isSystem,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func RehydrateRole(id, name, description string, isSystem bool, createdAt, updatedAt time.Time) Role {
	return Role{id: id, name: name, description: description, isSystem: isSystem, createdAt: createdAt, updatedAt: updatedAt}
}

func (r Role) ID() string          { return r.id }
func (r Role) Name() string        { return r.name }
func (r Role) Description() string { return r.description }
func (r Role) IsSystem() bool      { return r.isSystem }
func (r Role) CreatedAt() time.Time { return r.createdAt }
func (r Role) UpdatedAt() time.Time { return r.updatedAt }

// SetDescription updates the description.
func (r Role) SetDescription(desc string, now time.Time) Role {
	r.description = desc
	r.updatedAt = now
	return r
}

// ===== Permission =====

// Permission represents a single actionable capability (e.g. "orders.order.read").
type Permission struct {
	id          string
	name        string // e.g. "orders.order.read"
	description string
	module      string // e.g. "orders"
	resource    string // e.g. "order"
	action      string // e.g. "read"
	createdAt   time.Time
}

// NewPermission creates a new Permission with validation.
// The name must be in "module.resource.action" format.
func NewPermission(id, name, description string, now time.Time) (Permission, error) {
	if id == "" {
		return Permission{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if name == "" {
		return Permission{}, ErrEmptyPermissionName
	}
	module, resource, action, err := ParsePermissionName(name)
	if err != nil {
		return Permission{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Permission{
		id:          id,
		name:        name,
		description: description,
		module:      module,
		resource:    resource,
		action:      action,
		createdAt:   now,
	}, nil
}

func RehydratePermission(id, name, description, module, resource, action string, createdAt time.Time) Permission {
	return Permission{id: id, name: name, description: description, module: module, resource: resource, action: action, createdAt: createdAt}
}

func (p Permission) ID() string          { return p.id }
func (p Permission) Name() string        { return p.name }
func (p Permission) Description() string { return p.description }
func (p Permission) Module() string      { return p.module }
func (p Permission) Resource() string    { return p.resource }
func (p Permission) Action() string      { return p.action }
func (p Permission) CreatedAt() time.Time { return p.createdAt }

// ParsePermissionName splits a permission name into (module, resource, action).
// Format: "module.resource.action" (exactly 3 parts, lowercase, no spaces).
// Wildcards are allowed: "admin.*.*" or "orders.order.*".
func ParsePermissionName(name string) (module, resource, action string, err error) {
	parts := strings.Split(name, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("%w: %q (expected 3 parts, got %d)", ErrInvalidPermissionFormat, name, len(parts))
	}
	module = parts[0]
	resource = parts[1]
	action = parts[2]
	if module == "" || resource == "" || action == "" {
		return "", "", "", fmt.Errorf("%w: %q (empty parts)", ErrInvalidPermissionFormat, name)
	}
	return module, resource, action, nil
}

// Matches checks if this permission matches the given required permission.
// Wildcards (*) in the permission name match any value at that position.
// Example: permission "orders.*.*" matches required "orders.order.read".
func (p Permission) Matches(required string) bool {
	reqModule, reqResource, reqAction, err := ParsePermissionName(required)
	if err != nil {
		return false
	}
	if p.module != "*" && p.module != reqModule {
		return false
	}
	if p.resource != "*" && p.resource != reqResource {
		return false
	}
	if p.action != "*" && p.action != reqAction {
		return false
	}
	return true
}

// ===== UserRole =====

// UserRole represents the assignment of a role to a user.
type UserRole struct {
	id        string
	userID    string
	roleID    string
	assignedBy string
	createdAt time.Time
}

func NewUserRole(id, userID, roleID, assignedBy string, now time.Time) (UserRole, error) {
	if id == "" {
		return UserRole{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if userID == "" {
		return UserRole{}, ErrEmptyUserID
	}
	if roleID == "" {
		return UserRole{}, fmt.Errorf("%w: role id is required", ErrInvalidInput)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return UserRole{
		id:        id,
		userID:    userID,
		roleID:    roleID,
		assignedBy: assignedBy,
		createdAt: now,
	}, nil
}

func RehydrateUserRole(id, userID, roleID, assignedBy string, createdAt time.Time) UserRole {
	return UserRole{id: id, userID: userID, roleID: roleID, assignedBy: assignedBy, createdAt: createdAt}
}

func (ur UserRole) ID() string         { return ur.id }
func (ur UserRole) UserID() string     { return ur.userID }
func (ur UserRole) RoleID() string     { return ur.roleID }
func (ur UserRole) AssignedBy() string { return ur.assignedBy }
func (ur UserRole) CreatedAt() time.Time { return ur.createdAt }

// ===== RolePermission =====

// RolePermission represents the grant of a permission to a role.
type RolePermission struct {
	id           string
	roleID       string
	permissionID string
	createdAt    time.Time
}

func NewRolePermission(id, roleID, permissionID string, now time.Time) (RolePermission, error) {
	if id == "" {
		return RolePermission{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if roleID == "" {
		return RolePermission{}, fmt.Errorf("%w: role id is required", ErrInvalidInput)
	}
	if permissionID == "" {
		return RolePermission{}, fmt.Errorf("%w: permission id is required", ErrInvalidInput)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return RolePermission{id: id, roleID: roleID, permissionID: permissionID, createdAt: now}, nil
}

func RehydrateRolePermission(id, roleID, permissionID string, createdAt time.Time) RolePermission {
	return RolePermission{id: id, roleID: roleID, permissionID: permissionID, createdAt: createdAt}
}

func (rp RolePermission) ID() string         { return rp.id }
func (rp RolePermission) RoleID() string     { return rp.roleID }
func (rp RolePermission) PermissionID() string { return rp.permissionID }
func (rp RolePermission) CreatedAt() time.Time { return rp.createdAt }

// ===== Default Roles =====

// System-defined roles (created by migration seed data).
const (
	RoleAdmin    = "admin"
	RoleAgent    = "agent"
	RoleMerchant = "merchant"
	RoleDriver   = "driver"
	RoleUser     = "user"
)

// IsSystemRole reports whether the given role name is a system-defined role.
func IsSystemRole(name string) bool {
	switch name {
	case RoleAdmin, RoleAgent, RoleMerchant, RoleDriver, RoleUser:
		return true
	}
	return false
}
