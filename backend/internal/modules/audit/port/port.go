// Package port: repository + service interfaces + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/audit/domain"
)

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

// ===== Repository Interface =====
type AuditRepository interface {
	Create(ctx context.Context, exec Executor, entry domain.AuditEntry) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.AuditEntry, error)
	ListByActor(ctx context.Context, exec Executor, actorType, actorID string, page PageQuery) (Page[domain.AuditEntry], error)
	ListByResource(ctx context.Context, exec Executor, resourceType, resourceID string, page PageQuery) (Page[domain.AuditEntry], error)
	ListByAction(ctx context.Context, exec Executor, action string, page PageQuery) (Page[domain.AuditEntry], error)
	ListBySeverity(ctx context.Context, exec Executor, severity string, page PageQuery) (Page[domain.AuditEntry], error)
	ListByTimeRange(ctx context.Context, exec Executor, from, to time.Time, page PageQuery) (Page[domain.AuditEntry], error)
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.AuditEntry], error)
	CountByAction(ctx context.Context, exec Executor, from, to time.Time) (map[string]int64, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}

type RepositorySet struct {
	Audit  AuditRepository
	Outbox OutboxRepository
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
