// Package postgres implements the permissions module's repository interfaces.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/permissions/domain"
	"avex-backend/internal/modules/permissions/port"
	"avex-backend/internal/platform/database"
)

type Repositories struct {
	roles  *RoleRepository
	perms  *PermissionRepository
	rps    *RolePermissionRepository
	urs    *UserRoleRepository
	outbox *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{roles: &RoleRepository{}, perms: &PermissionRepository{}, rps: &RolePermissionRepository{}, urs: &UserRoleRepository{}, outbox: &OutboxRepository{}}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{Roles: r.roles, Permissions: r.perms, RolePermissions: r.rps, UserRoles: r.urs, Outbox: r.outbox}
}

func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok { panic("postgres: port.Executor does not satisfy database.DBTX") }
	return dbtx
}
type scanner interface{ Scan(dest ...any) error }
func nilIfEmptyStr(s string) any { if s == "" { return nil }; return s }

// ===== RoleRepository =====
type RoleRepository struct{}
var _ port.RoleRepository = (*RoleRepository)(nil)

func (r *RoleRepository) Create(ctx context.Context, exec port.Executor, role domain.Role) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO permissions.roles (id, name, description, is_system, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		role.ID(), role.Name(), role.Description(), role.IsSystem(), role.CreatedAt(), role.UpdatedAt())
	if err != nil { return fmt.Errorf("create role: %w", err) }
	return nil
}
func (r *RoleRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Role, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, name, description, is_system, created_at, updated_at FROM permissions.roles WHERE id = $1`, id)
	role, err := scanRole(row)
	if err != nil { return nil, mapRoleErr(err) }
	return &role, nil
}
func (r *RoleRepository) GetByName(ctx context.Context, exec port.Executor, name string) (*domain.Role, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, name, description, is_system, created_at, updated_at FROM permissions.roles WHERE name = $1`, name)
	role, err := scanRole(row)
	if err != nil { return nil, mapRoleErr(err) }
	return &role, nil
}
func (r *RoleRepository) Update(ctx context.Context, exec port.Executor, role domain.Role) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE permissions.roles SET description=$2, updated_at=$3 WHERE id=$1`,
		role.ID(), role.Description(), role.UpdatedAt())
	return err
}
func (r *RoleRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `DELETE FROM permissions.roles WHERE id=$1 AND is_system=FALSE`, id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return domain.ErrCannotModifySystemRole }
	return nil
}
func (r *RoleRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Role], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)
	var total int64
	if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM permissions.roles`).Scan(&total); err != nil { return port.Page[domain.Role]{}, err }
	rows, err := dbtx.Query(ctx, `SELECT id, name, description, is_system, created_at, updated_at FROM permissions.roles ORDER BY name LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil { return port.Page[domain.Role]{}, err }
	defer rows.Close()
	var items []domain.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil { return port.Page[domain.Role]{}, err }
		items = append(items, role)
	}
	return port.Page[domain.Role]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}
func scanRole(s scanner) (domain.Role, error) {
	var id, name string; var desc *string; var isSystem bool; var createdAt, updatedAt time.Time
	if err := s.Scan(&id, &name, &desc, &isSystem, &createdAt, &updatedAt); err != nil { return domain.Role{}, err }
	var descStr string; if desc != nil { descStr = *desc }
	return domain.RehydrateRole(id, name, descStr, isSystem, createdAt, updatedAt), nil
}
func mapRoleErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) { return domain.ErrRoleNotFound }
	return err
}

// ===== PermissionRepository =====
type PermissionRepository struct{}
var _ port.PermissionRepository = (*PermissionRepository)(nil)

func (r *PermissionRepository) Create(ctx context.Context, exec port.Executor, p domain.Permission) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO permissions.permissions (id, name, description, module, resource, action, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID(), p.Name(), p.Description(), p.Module(), p.Resource(), p.Action(), p.CreatedAt())
	return err
}
func (r *PermissionRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Permission, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, name, description, module, resource, action, created_at FROM permissions.permissions WHERE id=$1`, id)
	p, err := scanPerm(row)
	if err != nil { return nil, mapPermErr(err) }
	return &p, nil
}
func (r *PermissionRepository) GetByName(ctx context.Context, exec port.Executor, name string) (*domain.Permission, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, name, description, module, resource, action, created_at FROM permissions.permissions WHERE name=$1`, name)
	p, err := scanPerm(row)
	if err != nil { return nil, mapPermErr(err) }
	return &p, nil
}
func (r *PermissionRepository) GetByIDs(ctx context.Context, exec port.Executor, ids []string) ([]domain.Permission, error) {
	if len(ids) == 0 { return nil, nil }
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, name, description, module, resource, action, created_at FROM permissions.permissions WHERE id = ANY($1)`, ids)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.Permission
	for rows.Next() {
		p, err := scanPerm(rows)
		if err != nil { return nil, err }
		items = append(items, p)
	}
	return items, rows.Err()
}
func (r *PermissionRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Permission], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)
	var total int64
	if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM permissions.permissions`).Scan(&total); err != nil { return port.Page[domain.Permission]{}, err }
	rows, err := dbtx.Query(ctx, `SELECT id, name, description, module, resource, action, created_at FROM permissions.permissions ORDER BY module, name LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil { return port.Page[domain.Permission]{}, err }
	defer rows.Close()
	var items []domain.Permission
	for rows.Next() {
		p, err := scanPerm(rows)
		if err != nil { return port.Page[domain.Permission]{}, err }
		items = append(items, p)
	}
	return port.Page[domain.Permission]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}
func (r *PermissionRepository) ListByModule(ctx context.Context, exec port.Executor, module string) ([]domain.Permission, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, name, description, module, resource, action, created_at FROM permissions.permissions WHERE module=$1 ORDER BY name`, module)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.Permission
	for rows.Next() {
		p, err := scanPerm(rows)
		if err != nil { return nil, err }
		items = append(items, p)
	}
	return items, rows.Err()
}
func scanPerm(s scanner) (domain.Permission, error) {
	var id, name, module, resource, action string; var desc *string; var createdAt time.Time
	if err := s.Scan(&id, &name, &desc, &module, &resource, &action, &createdAt); err != nil { return domain.Permission{}, err }
	var descStr string; if desc != nil { descStr = *desc }
	return domain.RehydratePermission(id, name, descStr, module, resource, action, createdAt), nil
}
func mapPermErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) { return domain.ErrPermissionNotFound }
	return err
}

// ===== RolePermissionRepository =====
type RolePermissionRepository struct{}
var _ port.RolePermissionRepository = (*RolePermissionRepository)(nil)

func (r *RolePermissionRepository) Grant(ctx context.Context, exec port.Executor, rp domain.RolePermission) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO permissions.role_permissions (id, role_id, permission_id, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT (role_id, permission_id) DO NOTHING`,
		rp.ID(), rp.RoleID(), rp.PermissionID(), rp.CreatedAt())
	return err
}
func (r *RolePermissionRepository) Revoke(ctx context.Context, exec port.Executor, roleID, permissionID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `DELETE FROM permissions.role_permissions WHERE role_id=$1 AND permission_id=$2`, roleID, permissionID)
	return err
}
func (r *RolePermissionRepository) ListByRole(ctx context.Context, exec port.Executor, roleID string) ([]domain.RolePermission, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, role_id, permission_id, created_at FROM permissions.role_permissions WHERE role_id=$1`, roleID)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.RolePermission
	for rows.Next() {
		var id, roleID, permID string; var createdAt time.Time
		if err := rows.Scan(&id, &roleID, &permID, &createdAt); err != nil { return nil, err }
		items = append(items, domain.RehydrateRolePermission(id, roleID, permID, createdAt))
	}
	return items, rows.Err()
}
func (r *RolePermissionRepository) ListPermissionIDsByRoles(ctx context.Context, exec port.Executor, roleIDs []string) ([]string, error) {
	if len(roleIDs) == 0 { return nil, nil }
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT DISTINCT permission_id FROM permissions.role_permissions WHERE role_id = ANY($1)`, roleIDs)
	if err != nil { return nil, err }
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { return nil, err }
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ===== UserRoleRepository =====
type UserRoleRepository struct{}
var _ port.UserRoleRepository = (*UserRoleRepository)(nil)

func (r *UserRoleRepository) Assign(ctx context.Context, exec port.Executor, ur domain.UserRole) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO permissions.user_roles (id, user_id, role_id, assigned_by, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (user_id, role_id) DO NOTHING`,
		ur.ID(), ur.UserID(), ur.RoleID(), nilIfEmptyStr(ur.AssignedBy()), ur.CreatedAt())
	return err
}
func (r *UserRoleRepository) Unassign(ctx context.Context, exec port.Executor, userID, roleID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `DELETE FROM permissions.user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID)
	return err
}
func (r *UserRoleRepository) ListByUser(ctx context.Context, exec port.Executor, userID string) ([]domain.UserRole, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, user_id, role_id, assigned_by, created_at FROM permissions.user_roles WHERE user_id=$1`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.UserRole
	for rows.Next() {
		var id, userID, roleID string; var assignedBy *string; var createdAt time.Time
		if err := rows.Scan(&id, &userID, &roleID, &assignedBy, &createdAt); err != nil { return nil, err }
		var assignedByStr string; if assignedBy != nil { assignedByStr = *assignedBy }
		items = append(items, domain.RehydrateUserRole(id, userID, roleID, assignedByStr, createdAt))
	}
	return items, rows.Err()
}
func (r *UserRoleRepository) ListRoleIDsByUser(ctx context.Context, exec port.Executor, userID string) ([]string, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT role_id FROM permissions.user_roles WHERE user_id=$1`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { return nil, err }
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *UserRoleRepository) ListUsersByRole(ctx context.Context, exec port.Executor, roleID string, page port.PageQuery) (port.Page[domain.UserRole], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)
	var total int64
	if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM permissions.user_roles WHERE role_id=$1`, roleID).Scan(&total); err != nil { return port.Page[domain.UserRole]{}, err }
	rows, err := dbtx.Query(ctx, `SELECT id, user_id, role_id, assigned_by, created_at FROM permissions.user_roles WHERE role_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, roleID, page.Limit, page.Offset)
	if err != nil { return port.Page[domain.UserRole]{}, err }
	defer rows.Close()
	var items []domain.UserRole
	for rows.Next() {
		var id, userID, roleID string; var assignedBy *string; var createdAt time.Time
		if err := rows.Scan(&id, &userID, &roleID, &assignedBy, &createdAt); err != nil { return port.Page[domain.UserRole]{}, err }
		var assignedByStr string; if assignedBy != nil { assignedByStr = *assignedBy }
		items = append(items, domain.RehydrateUserRole(id, userID, roleID, assignedByStr, createdAt))
	}
	return port.Page[domain.UserRole]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}

// ===== OutboxRepository =====
type OutboxRepository struct{}
var _ port.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Save(ctx context.Context, exec port.Executor, env port.EventEnvelope) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO permissions.outbox (event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent, next_retry_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW())`,
		env.EventID, env.EventType, env.EventVersion, env.SchemaVersion, env.Payload, env.OccurredAt, env.Producer,
		nilIfEmptyStr(env.CorrelationID), nilIfEmptyStr(env.TraceID), nilIfEmptyStr(env.ActorType), nilIfEmptyStr(env.ActorID), nilIfEmptyStr(env.ActorIP), nilIfEmptyStr(env.ActorUA))
	return err
}
func (r *OutboxRepository) GetPending(ctx context.Context, exec port.Executor, limit int) ([]port.EventEnvelope, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM permissions.outbox WHERE published_at IS NULL AND next_retry_at <= NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var envs []port.EventEnvelope
	for rows.Next() {
		var env port.EventEnvelope; var corID, traceID, aType, aID, aIP, aUA *string
		if err := rows.Scan(&env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion, &env.Payload, &env.OccurredAt, &env.Producer, &corID, &traceID, &aType, &aID, &aIP, &aUA); err != nil { return nil, err }
		if corID != nil { env.CorrelationID = *corID }; if traceID != nil { env.TraceID = *traceID }
		if aType != nil { env.ActorType = *aType }; if aID != nil { env.ActorID = *aID }
		if aIP != nil { env.ActorIP = *aIP }; if aUA != nil { env.ActorUA = *aUA }
		envs = append(envs, env)
	}
	return envs, rows.Err()
}
func (r *OutboxRepository) MarkPublished(ctx context.Context, exec port.Executor, eventID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE permissions.outbox SET published_at=NOW(), last_error=NULL WHERE event_id=$1`, eventID)
	return err
}

// Worker helpers
func (r *OutboxRepository) MarkFailed(ctx context.Context, pool *pgxpool.Pool, eventID string, err error) error {
	errMsg := ""; if err != nil { if len(err.Error()) > 2000 { errMsg = err.Error()[:2000] } else { errMsg = err.Error() } }
	_, execErr := pool.Exec(ctx, `UPDATE permissions.outbox SET retry_count=retry_count+1, last_error=$2, next_retry_at=NOW()+make_interval(secs=>LEAST(1*POWER(2,retry_count),3600)) WHERE event_id=$1`, eventID, errMsg)
	return execErr
}
func (r *OutboxRepository) FetchPendingWithIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxEntryWithID, error) {
	rows, err := pool.Query(ctx, `SELECT id, event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM permissions.outbox WHERE published_at IS NULL AND next_retry_at<=NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var entries []OutboxEntryWithID
	for rows.Next() {
		var entryID int64; var env port.EventEnvelope; var corID, traceID, aType, aID, aIP, aUA *string
		if err := rows.Scan(&entryID, &env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion, &env.Payload, &env.OccurredAt, &env.Producer, &corID, &traceID, &aType, &aID, &aIP, &aUA); err != nil { return nil, err }
		if corID != nil { env.CorrelationID = *corID }; if traceID != nil { env.TraceID = *traceID }
		if aType != nil { env.ActorType = *aType }; if aID != nil { env.ActorID = *aID }
		if aIP != nil { env.ActorIP = *aIP }; if aUA != nil { env.ActorUA = *aUA }
		entries = append(entries, OutboxEntryWithID{EntryID: entryID, Envelope: env})
	}
	return entries, rows.Err()
}
type OutboxEntryWithID struct{ EntryID int64; Envelope port.EventEnvelope }
func (r *OutboxRepository) MarkPublishedByID(ctx context.Context, pool *pgxpool.Pool, entryID int64) error {
	_, err := pool.Exec(ctx, `UPDATE permissions.outbox SET published_at=NOW(), last_error=NULL WHERE id=$1`, entryID)
	return err
}
