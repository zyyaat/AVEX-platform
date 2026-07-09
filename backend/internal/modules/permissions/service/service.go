// Package service: permissions service implementation.
package service

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/permissions/domain"
	"avex-backend/internal/modules/permissions/events"
	"avex-backend/internal/modules/permissions/port"
)

type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service { return &Service{deps: deps, pool: pool} }

func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
	return port.EventContext{Actor: actor, Metadata: port.EventMetadata{OccurredAt: s.deps.Clock.Now()}}
}

// ===== Roles =====

func (s *Service) CreateRole(ctx context.Context, input port.CreateRoleInput) (*port.RoleDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	role, err := domain.NewRole(id, input.Name, input.Description, input.IsSystem, now)
	if err != nil { return nil, err }
	if err := s.deps.Repos.Roles.Create(ctx, s.pool, role); err != nil { return nil, err }
	// Publish event
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
	env, _ := events.RoleCreatedEnvelope(port.RoleCreatedPayload{RoleID: role.ID(), Name: role.Name(), IsSystem: role.IsSystem()}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return port.ToRoleDTOPtr(role), nil
}

func (s *Service) GetRole(ctx context.Context, id string) (*port.RoleDTO, error) {
	r, err := s.deps.Repos.Roles.GetByID(ctx, s.pool, id)
	if err != nil { return nil, err }
	return port.ToRoleDTOPtr(*r), nil
}

func (s *Service) GetRoleByName(ctx context.Context, name string) (*port.RoleDTO, error) {
	r, err := s.deps.Repos.Roles.GetByName(ctx, s.pool, name)
	if err != nil { return nil, err }
	return port.ToRoleDTOPtr(*r), nil
}

func (s *Service) ListRoles(ctx context.Context, page port.PageQuery) (port.Page[port.RoleDTO], error) {
	result, err := s.deps.Repos.Roles.ListAll(ctx, s.pool, page)
	if err != nil { return port.Page[port.RoleDTO]{}, err }
	dtos := make([]port.RoleDTO, 0, len(result.Items))
	for _, r := range result.Items { dtos = append(dtos, port.ToRoleDTO(r)) }
	return port.Page[port.RoleDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

func (s *Service) DeleteRole(ctx context.Context, id string) error {
	role, err := s.deps.Repos.Roles.GetByID(ctx, s.pool, id)
	if err != nil { return err }
	if role.IsSystem() { return domain.ErrCannotModifySystemRole }
	if err := s.deps.Repos.Roles.Delete(ctx, s.pool, id); err != nil { return err }
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
	env, _ := events.RoleDeletedEnvelope(port.RoleDeletedPayload{RoleID: id}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return nil
}

// ===== Permissions =====

func (s *Service) CreatePermission(ctx context.Context, input port.CreatePermissionInput) (*port.PermissionDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	p, err := domain.NewPermission(id, input.Name, input.Description, now)
	if err != nil { return nil, err }
	if err := s.deps.Repos.Permissions.Create(ctx, s.pool, p); err != nil { return nil, err }
	return port.ToPermissionDTOPtr(p), nil
}

func (s *Service) GetPermission(ctx context.Context, id string) (*port.PermissionDTO, error) {
	p, err := s.deps.Repos.Permissions.GetByID(ctx, s.pool, id)
	if err != nil { return nil, err }
	return port.ToPermissionDTOPtr(*p), nil
}

func (s *Service) ListPermissions(ctx context.Context, page port.PageQuery) (port.Page[port.PermissionDTO], error) {
	result, err := s.deps.Repos.Permissions.ListAll(ctx, s.pool, page)
	if err != nil { return port.Page[port.PermissionDTO]{}, err }
	dtos := make([]port.PermissionDTO, 0, len(result.Items))
	for _, p := range result.Items { dtos = append(dtos, port.ToPermissionDTO(p)) }
	return port.Page[port.PermissionDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

func (s *Service) ListPermissionsByModule(ctx context.Context, module string) ([]port.PermissionDTO, error) {
	perms, err := s.deps.Repos.Permissions.ListByModule(ctx, s.pool, module)
	if err != nil { return nil, err }
	dtos := make([]port.PermissionDTO, 0, len(perms))
	for _, p := range perms { dtos = append(dtos, port.ToPermissionDTO(p)) }
	return dtos, nil
}

// ===== Role-Permission Mapping =====

func (s *Service) GrantPermission(ctx context.Context, input port.GrantPermissionInput) error {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	rp, err := domain.NewRolePermission(id, input.RoleID, input.PermissionID, now)
	if err != nil { return err }
	if err := s.deps.Repos.RolePermissions.Grant(ctx, s.pool, rp); err != nil { return err }
	// Get permission name for event
	perm, _ := s.deps.Repos.Permissions.GetByID(ctx, s.pool, input.PermissionID)
	permName := ""
	if perm != nil { permName = perm.Name() }
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
	env, _ := events.PermissionGrantedEnvelope(port.PermissionGrantedPayload{RoleID: input.RoleID, PermissionID: input.PermissionID, PermissionName: permName}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return nil
}

func (s *Service) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	if err := s.deps.Repos.RolePermissions.Revoke(ctx, s.pool, roleID, permissionID); err != nil { return err }
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
	env, _ := events.PermissionRevokedEnvelope(port.PermissionRevokedPayload{RoleID: roleID, PermissionID: permissionID}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return nil
}

func (s *Service) ListPermissionsByRole(ctx context.Context, roleID string) ([]port.PermissionDTO, error) {
	rps, err := s.deps.Repos.RolePermissions.ListByRole(ctx, s.pool, roleID)
	if err != nil { return nil, err }
	if len(rps) == 0 { return []port.PermissionDTO{}, nil }
	permIDs := make([]string, 0, len(rps))
	for _, rp := range rps { permIDs = append(permIDs, rp.PermissionID()) }
	perms, err := s.deps.Repos.Permissions.GetByIDs(ctx, s.pool, permIDs)
	if err != nil { return nil, err }
	dtos := make([]port.PermissionDTO, 0, len(perms))
	for _, p := range perms { dtos = append(dtos, port.ToPermissionDTO(p)) }
	return dtos, nil
}

// ===== User-Role Mapping =====

func (s *Service) AssignRole(ctx context.Context, input port.AssignRoleInput) (*port.UserRoleDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	ur, err := domain.NewUserRole(id, input.UserID, input.RoleID, input.AssignedBy, now)
	if err != nil { return nil, err }
	if err := s.deps.Repos.UserRoles.Assign(ctx, s.pool, ur); err != nil { return nil, err }
	// Get role name for DTO + event
	role, _ := s.deps.Repos.Roles.GetByID(ctx, s.pool, input.RoleID)
	roleName := ""
	if role != nil { roleName = role.Name() }
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin", ID: input.AssignedBy})
	env, _ := events.RoleAssignedEnvelope(port.RoleAssignedPayload{UserID: input.UserID, RoleID: input.RoleID, RoleName: roleName, AssignedBy: input.AssignedBy}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return port.ToUserRoleDTOPtr(ur, roleName), nil
}

func (s *Service) UnassignRole(ctx context.Context, userID, roleID string) error {
	if err := s.deps.Repos.UserRoles.Unassign(ctx, s.pool, userID, roleID); err != nil { return err }
	ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
	env, _ := events.RoleUnassignedEnvelope(port.RoleUnassignedPayload{UserID: userID, RoleID: roleID}, ec)
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
	return nil
}

func (s *Service) ListRolesByUser(ctx context.Context, userID string) ([]port.RoleDTO, error) {
	urs, err := s.deps.Repos.UserRoles.ListByUser(ctx, s.pool, userID)
	if err != nil { return nil, err }
	if len(urs) == 0 { return []port.RoleDTO{}, nil }
	roleIDs := make([]string, 0, len(urs))
	for _, ur := range urs { roleIDs = append(roleIDs, ur.RoleID()) }
	// Fetch roles — we don't have a GetByIDs on RoleRepository, so fetch one-by-one
	// (small N — users typically have 1-2 roles)
	dtos := make([]port.RoleDTO, 0, len(roleIDs))
	for _, rid := range roleIDs {
		r, err := s.deps.Repos.Roles.GetByID(ctx, s.pool, rid)
		if err != nil { continue }
		dtos = append(dtos, port.ToRoleDTO(*r))
	}
	return dtos, nil
}

func (s *Service) ListPermissionsByUser(ctx context.Context, userID string) ([]port.PermissionDTO, error) {
	roleIDs, err := s.deps.Repos.UserRoles.ListRoleIDsByUser(ctx, s.pool, userID)
	if err != nil { return nil, err }
	if len(roleIDs) == 0 { return []port.PermissionDTO{}, nil }
	permIDs, err := s.deps.Repos.RolePermissions.ListPermissionIDsByRoles(ctx, s.pool, roleIDs)
	if err != nil { return nil, err }
	if len(permIDs) == 0 { return []port.PermissionDTO{}, nil }
	perms, err := s.deps.Repos.Permissions.GetByIDs(ctx, s.pool, permIDs)
	if err != nil { return nil, err }
	dtos := make([]port.PermissionDTO, 0, len(perms))
	for _, p := range perms { dtos = append(dtos, port.ToPermissionDTO(p)) }
	return dtos, nil
}

func (s *Service) ListUsersByRole(ctx context.Context, roleID string, page port.PageQuery) (port.Page[port.UserRoleDTO], error) {
	result, err := s.deps.Repos.UserRoles.ListUsersByRole(ctx, s.pool, roleID, page)
	if err != nil { return port.Page[port.UserRoleDTO]{}, err }
	// Get role name
	role, _ := s.deps.Repos.Roles.GetByID(ctx, s.pool, roleID)
	roleName := ""
	if role != nil { roleName = role.Name() }
	dtos := make([]port.UserRoleDTO, 0, len(result.Items))
	for _, ur := range result.Items { dtos = append(dtos, port.ToUserRoleDTO(ur, roleName)) }
	return port.Page[port.UserRoleDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

// ===== Permission Checking =====

func (s *Service) HasPermission(ctx context.Context, userID, permission string) (port.CheckPermissionResult, error) {
	roleIDs, err := s.deps.Repos.UserRoles.ListRoleIDsByUser(ctx, s.pool, userID)
	if err != nil { return port.CheckPermissionResult{}, err }
	if len(roleIDs) == 0 {
		return port.CheckPermissionResult{Allowed: false, UserID: userID, Permission: permission}, nil
	}
	// Get role names for the result
	roleNames := make([]string, 0, len(roleIDs))
	for _, rid := range roleIDs {
		r, err := s.deps.Repos.Roles.GetByID(ctx, s.pool, rid)
		if err == nil { roleNames = append(roleNames, r.Name()) }
	}

	permIDs, err := s.deps.Repos.RolePermissions.ListPermissionIDsByRoles(ctx, s.pool, roleIDs)
	if err != nil { return port.CheckPermissionResult{}, err }
	if len(permIDs) == 0 {
		return port.CheckPermissionResult{Allowed: false, UserID: userID, Permission: permission, Roles: roleNames}, nil
	}
	perms, err := s.deps.Repos.Permissions.GetByIDs(ctx, s.pool, permIDs)
	if err != nil { return port.CheckPermissionResult{}, err }
	for _, p := range perms {
		if p.Matches(permission) {
			return port.CheckPermissionResult{Allowed: true, UserID: userID, Permission: permission, Roles: roleNames}, nil
		}
	}
	return port.CheckPermissionResult{Allowed: false, UserID: userID, Permission: permission, Roles: roleNames}, nil
}

func (s *Service) HasAnyPermission(ctx context.Context, userID string, permissions []string) (bool, error) {
	for _, perm := range permissions {
		result, err := s.HasPermission(ctx, userID, perm)
		if err != nil { return false, err }
		if result.Allowed { return true, nil }
	}
	return false, nil
}

func (s *Service) HasAllPermissions(ctx context.Context, userID string, permissions []string) (bool, error) {
	for _, perm := range permissions {
		result, err := s.HasPermission(ctx, userID, perm)
		if err != nil { return false, err }
		if !result.Allowed { return false, nil }
	}
	return true, nil
}

// suppress unused import
var _ = fmt.Sprintf
