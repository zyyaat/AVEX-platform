// Package port: repository + service interfaces + DTOs + provider interfaces.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/notifications/domain"
)

// ===== Executor / TxRunner =====

type Executor interface{}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

// ===== Pagination =====

type PageQuery struct {
	Limit  int
	Offset int
}

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

func (p PageQuery) Normalize() PageQuery {
	if p.Limit <= 0 {
		p.Limit = DefaultPageLimit
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

type Page[T any] struct {
	Items  []T
	Total  int64
	Limit  int
	Offset int
}

// ===== Repository Interfaces =====

type NotificationRepository interface {
	Create(ctx context.Context, exec Executor, n domain.Notification) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Notification, error)
	Update(ctx context.Context, exec Executor, n domain.Notification) error
	ListByRecipient(ctx context.Context, exec Executor, recipientType, recipientID string, page PageQuery) (Page[domain.Notification], error)
	ListPending(ctx context.Context, exec Executor, limit int) ([]domain.Notification, error)
}

type PreferenceRepository interface {
	GetByRecipient(ctx context.Context, exec Executor, recipientType, recipientID string) (*domain.UserNotificationPreferences, error)
	Upsert(ctx context.Context, exec Executor, prefs domain.UserNotificationPreferences) error
}

type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}

type RepositorySet struct {
	Notifications NotificationRepository
	Preferences   PreferenceRepository
	Outbox        OutboxRepository
}

// ===== Provider Interfaces =====

// PushProvider sends push notifications (FCM, APNs).
type PushProvider interface {
	Send(ctx context.Context, input PushInput) error
}

type PushInput struct {
	DeviceTokens []string
	Title        string
	Body         string
	Data         map[string]any
	Priority     string // normal | high
}

// SMSProvider sends SMS messages.
type SMSProvider interface {
	Send(ctx context.Context, input SMSInput) error
}

type SMSInput struct {
	To      string
	Message string
}

// EmailProvider sends emails.
type EmailProvider interface {
	Send(ctx context.Context, input EmailInput) error
}

type EmailInput struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// ===== Event Publisher =====

type EventEnvelope struct {
	EventID       string
	EventType     string
	EventVersion  int
	SchemaVersion int
	OccurredAt    time.Time
	Producer      string // always "notifications"
	CorrelationID string
	TraceID       string
	ActorType     string
	ActorID       string
	ActorIP       string
	ActorUA       string
	Payload       []byte
}

type EventPublisher interface {
	Publish(ctx context.Context, exec Executor, envelope EventEnvelope) error
}

type ActorContext struct {
	Type      string
	ID        string
	IP        string
	UserAgent string
}

// ===== Infra Dependencies =====

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Deps struct {
	Clock          Clock
	IDGenerator    IDGenerator
	EventPublisher EventPublisher
	Logger         Logger
	TxRunner       TxRunner
	Repos          RepositorySet
	PushProvider   PushProvider
	SMSProvider    SMSProvider
	EmailProvider  EmailProvider
}
