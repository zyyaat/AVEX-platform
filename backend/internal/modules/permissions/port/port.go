// Package port: repository + service interfaces + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/permissions/domain"
)

// ===== Executor / TxRunner =====
type Executor interface{}
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}
type Row interface{ Scan(dest ...any) error }
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}
type PageQuery struct{ Limit, Offset int }

const (DefaultPageLimit = 50; MaxPageLimit = 100)

func (p PageQuery) Normalize() PageQuery {
	if p.Limit <= 0 { p.Limit = DefaultPageLimit }
	if p.Limit > MaxPageLimit { p.Limit = MaxPageLimit }
	if p.Offset < 0 { p.Offset = 0 }
	return p
}
type Page[T any] struct{ Items []T; Total int64; Limit, Offset int }

// ===== Repository Interfaces =====
type RoleRepository interface {
	Create(ctx context.Context, exec Executor, role domain.Role) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Role, error)
	GetByName(ctx context.Context, exec Executor, name string) (*domain.Role, error)
	Update(ctx context.Context, exec Executor, role domain.Role) error
	Delete(ctx context.Context, exec Executor, id string) error
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Role], error)
}
type PermissionRepository interface {
	Create(ctx context.Context, exec Executor, p domain.Permission) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Permission, error)
	GetByName(ctx context.Context, exec Executor, name string) (*domain.Permission, error)
	GetByIDs(ctx context.Context, exec Executor, ids []string) ([]domain.Permission, error)
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Permission], error)
	ListByModule(ctx context.Context, exec Executor, module string) ([]domain.Permission, error)
}
type RolePermissionRepository interface {
	Grant(ctx context.Context, exec Executor, rp domain.RolePermission) error
	Revoke(ctx context.Context, exec Executor, roleID, permissionID string) error
	ListByRole(ctx context.Context, exec Executor, roleID string) ([]domain.RolePermission, error)
	ListPermissionIDsByRoles(ctx context.Context, exec Executor, roleIDs []string) ([]string, error)
}
type UserRoleRepository interface {
	Assign(ctx context.Context, exec Executor, ur domain.UserRole) error
	Unassign(ctx context.Context, exec Executor, userID, roleID string) error
	ListByUser(ctx context.Context, exec Executor, userID string) ([]domain.UserRole, error)
	ListRoleIDsByUser(ctx context.Context, exec Executor, userID string) ([]string, error)
	ListUsersByRole(ctx context.Context, exec Executor, roleID string, page PageQuery) (Page[domain.UserRole], error)
}
type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}
type RepositorySet struct {
	Roles            RoleRepository
	Permissions      PermissionRepository
	RolePermissions  RolePermissionRepository
	UserRoles        UserRoleRepository
	Outbox           OutboxRepository
}

// ===== Infra =====
type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type EventEnvelope struct {
	EventID, EventType string
	EventVersion, SchemaVersion int
	OccurredAt time.Time
	Producer, CorrelationID, TraceID, ActorType, ActorID, ActorIP, ActorUA string
	Payload []byte
}
type EventPublisher interface{ Publish(ctx context.Context, exec Executor, envelope EventEnvelope) error }
type ActorContext struct{ Type, ID, IP, UserAgent string }
type Deps struct {
	Clock Clock; IDGenerator IDGenerator; EventPublisher EventPublisher; Logger Logger
	TxRunner TxRunner; Repos RepositorySet
}
