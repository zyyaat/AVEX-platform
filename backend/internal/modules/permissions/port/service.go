// Package port service: ServicePort + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/permissions/domain"
)

// ===== Events =====
const (
	EventRoleCreated       = "permissions.role.created"
	EventRoleDeleted       = "permissions.role.deleted"
	EventPermissionGranted = "permissions.permission.granted"
	EventPermissionRevoked = "permissions.permission.revoked"
	EventRoleAssigned      = "permissions.role.assigned"
	EventRoleUnassigned    = "permissions.role.unassigned"
)
const (
	RoleCreatedEventVersion = 1; RoleCreatedSchemaVersion = 1
	RoleDeletedEventVersion = 1; RoleDeletedSchemaVersion = 1
	PermissionGrantedEventVersion = 1; PermissionGrantedSchemaVersion = 1
	PermissionRevokedEventVersion = 1; PermissionRevokedSchemaVersion = 1
	RoleAssignedEventVersion = 1; RoleAssignedSchemaVersion = 1
	RoleUnassignedEventVersion = 1; RoleUnassignedSchemaVersion = 1
)

type RoleCreatedPayload struct{ RoleID, Name string; IsSystem bool `json:"is_system"` }
type RoleDeletedPayload struct{ RoleID string }
type PermissionGrantedPayload struct{ RoleID, PermissionID, PermissionName string }
type PermissionRevokedPayload struct{ RoleID, PermissionID string }
type RoleAssignedPayload struct{ UserID, RoleID, RoleName, AssignedBy string }
type RoleUnassignedPayload struct{ UserID, RoleID string }

type EventMetadata struct{ CorrelationID, TraceID string; OccurredAt time.Time }
type EventContext struct{ Actor ActorContext; Metadata EventMetadata }

func BuildEnvelope(eventID, eventType string, eventVersion, schemaVersion int, payload []byte, ec EventContext) EventEnvelope {
	occurredAt := ec.Metadata.OccurredAt
	if occurredAt.IsZero() { occurredAt = time.Now().UTC() }
	return EventEnvelope{
		EventID: eventID, EventType: eventType, EventVersion: eventVersion, SchemaVersion: schemaVersion,
		OccurredAt: occurredAt, Producer: "permissions",
		CorrelationID: ec.Metadata.CorrelationID, TraceID: ec.Metadata.TraceID,
		ActorType: ec.Actor.Type, ActorID: ec.Actor.ID, ActorIP: ec.Actor.IP, ActorUA: ec.Actor.UserAgent,
		Payload: payload,
	}
}

// ===== DTOs =====
type CreateRoleInput struct{ Name, Description string; IsSystem bool }
type CreatePermissionInput struct{ Name, Description string }
type AssignRoleInput struct{ UserID, RoleID, AssignedBy string }
type GrantPermissionInput struct{ RoleID, PermissionID string }

type RoleDTO struct {
	ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`
	IsSystem bool `json:"is_system"`; CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
type PermissionDTO struct {
	ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`
	Module string `json:"module"`; Resource string `json:"resource"`; Action string `json:"action"`; CreatedAt time.Time `json:"created_at"`
}
type UserRoleDTO struct {
	ID string `json:"id"`; UserID string `json:"user_id"`; RoleID string `json:"role_id"`
	RoleName string `json:"role_name,omitempty"`; AssignedBy string `json:"assigned_by,omitempty"`; CreatedAt time.Time `json:"created_at"`
}
type CheckPermissionResult struct {
	Allowed bool `json:"allowed"`
	UserID  string `json:"user_id"`
	Permission string `json:"permission"`
	Roles   []string `json:"roles,omitempty"`
}

// ===== ServicePort =====
type ServicePort interface {
	// Roles
	CreateRole(ctx context.Context, input CreateRoleInput) (*RoleDTO, error)
	GetRole(ctx context.Context, id string) (*RoleDTO, error)
	GetRoleByName(ctx context.Context, name string) (*RoleDTO, error)
	ListRoles(ctx context.Context, page PageQuery) (Page[RoleDTO], error)
	DeleteRole(ctx context.Context, id string) error

	// Permissions
	CreatePermission(ctx context.Context, input CreatePermissionInput) (*PermissionDTO, error)
	GetPermission(ctx context.Context, id string) (*PermissionDTO, error)
	ListPermissions(ctx context.Context, page PageQuery) (Page[PermissionDTO], error)
	ListPermissionsByModule(ctx context.Context, module string) ([]PermissionDTO, error)

	// Role-Permission mapping
	GrantPermission(ctx context.Context, input GrantPermissionInput) error
	RevokePermission(ctx context.Context, roleID, permissionID string) error
	ListPermissionsByRole(ctx context.Context, roleID string) ([]PermissionDTO, error)

	// User-Role mapping
	AssignRole(ctx context.Context, input AssignRoleInput) (*UserRoleDTO, error)
	UnassignRole(ctx context.Context, userID, roleID string) error
	ListRolesByUser(ctx context.Context, userID string) ([]RoleDTO, error)
	ListPermissionsByUser(ctx context.Context, userID string) ([]PermissionDTO, error)
	ListUsersByRole(ctx context.Context, roleID string, page PageQuery) (Page[UserRoleDTO], error)

	// Permission checking
	HasPermission(ctx context.Context, userID, permission string) (CheckPermissionResult, error)
	HasAnyPermission(ctx context.Context, userID string, permissions []string) (bool, error)
	HasAllPermissions(ctx context.Context, userID string, permissions []string) (bool, error)
}

// ===== Mappers =====
func ToRoleDTO(r domain.Role) RoleDTO {
	return RoleDTO{ID: r.ID(), Name: r.Name(), Description: r.Description(), IsSystem: r.IsSystem(), CreatedAt: r.CreatedAt(), UpdatedAt: r.UpdatedAt()}
}
func ToPermissionDTO(p domain.Permission) PermissionDTO {
	return PermissionDTO{ID: p.ID(), Name: p.Name(), Description: p.Description(), Module: p.Module(), Resource: p.Resource(), Action: p.Action(), CreatedAt: p.CreatedAt()}
}
func ToUserRoleDTO(ur domain.UserRole, roleName string) UserRoleDTO {
	return UserRoleDTO{ID: ur.ID(), UserID: ur.UserID(), RoleID: ur.RoleID(), RoleName: roleName, AssignedBy: ur.AssignedBy(), CreatedAt: ur.CreatedAt()}
}
func ToRoleDTOPtr(r domain.Role) *RoleDTO { dto := ToRoleDTO(r); return &dto }
func ToPermissionDTOPtr(p domain.Permission) *PermissionDTO { dto := ToPermissionDTO(p); return &dto }
func ToUserRoleDTOPtr(ur domain.UserRole, roleName string) *UserRoleDTO { dto := ToUserRoleDTO(ur, roleName); return &dto }
