// Package port: repository + service interfaces + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/settings/domain"
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

// ===== Repository Interfaces =====
type SettingRepository interface {
	Create(ctx context.Context, exec Executor, s domain.Setting) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Setting, error)
	GetByKey(ctx context.Context, exec Executor, key string) (*domain.Setting, error)
	Update(ctx context.Context, exec Executor, s domain.Setting) error
	Delete(ctx context.Context, exec Executor, id string) error
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Setting], error)
	ListByType(ctx context.Context, exec Executor, settingType string) ([]domain.Setting, error)
}
type RevisionRepository interface {
	Create(ctx context.Context, exec Executor, r domain.SettingRevision) error
	ListBySetting(ctx context.Context, exec Executor, settingID string, page PageQuery) (Page[domain.SettingRevision], error)
	GetBySettingAndVersion(ctx context.Context, exec Executor, settingID string, version int) (*domain.SettingRevision, error)
}
type FeatureFlagRepository interface {
	Create(ctx context.Context, exec Executor, f domain.FeatureFlag) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.FeatureFlag, error)
	GetByName(ctx context.Context, exec Executor, name string) (*domain.FeatureFlag, error)
	Update(ctx context.Context, exec Executor, f domain.FeatureFlag) error
	Delete(ctx context.Context, exec Executor, id string) error
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.FeatureFlag], error)
	ListEnabled(ctx context.Context, exec Executor) ([]domain.FeatureFlag, error)
}
type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}
type RepositorySet struct {
	Settings  SettingRepository
	Revisions RevisionRepository
	Flags     FeatureFlagRepository
	Outbox    OutboxRepository
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
