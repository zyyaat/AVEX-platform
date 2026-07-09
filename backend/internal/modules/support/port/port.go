// Package port: repository + service interfaces + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/support/domain"
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

type TicketRepository interface {
	Create(ctx context.Context, exec Executor, t domain.Ticket) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Ticket, error)
	GetByTicketNo(ctx context.Context, exec Executor, ticketNo string) (*domain.Ticket, error)
	Update(ctx context.Context, exec Executor, t domain.Ticket) error
	ListByUser(ctx context.Context, exec Executor, userID string, page PageQuery) (Page[domain.Ticket], error)
	ListByAgent(ctx context.Context, exec Executor, agentID string, page PageQuery) (Page[domain.Ticket], error)
	ListByStatus(ctx context.Context, exec Executor, status string, page PageQuery) (Page[domain.Ticket], error)
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Ticket], error)
	ListUnassigned(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Ticket], error)
}

type MessageRepository interface {
	Create(ctx context.Context, exec Executor, m domain.TicketMessage) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.TicketMessage, error)
	Update(ctx context.Context, exec Executor, m domain.TicketMessage) error
	ListByTicket(ctx context.Context, exec Executor, ticketID string, page PageQuery) (Page[domain.TicketMessage], error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, exec Executor, a domain.TicketAttachment) error
	ListByMessage(ctx context.Context, exec Executor, messageID string) ([]domain.TicketAttachment, error)
	ListByTicket(ctx context.Context, exec Executor, ticketID string) ([]domain.TicketAttachment, error)
}

type OutboxRepository interface {
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}

type RepositorySet struct {
	Tickets     TicketRepository
	Messages    MessageRepository
	Attachments AttachmentRepository
	Outbox      OutboxRepository
}

// ===== Infra Dependencies =====

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type TicketNumberGenerator interface {
	Generate() string
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type EventEnvelope struct {
	EventID       string
	EventType     string
	EventVersion  int
	SchemaVersion int
	OccurredAt    time.Time
	Producer      string
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

type Deps struct {
	Clock             Clock
	IDGenerator       IDGenerator
	TicketNumberGen   TicketNumberGenerator
	EventPublisher    EventPublisher
	Logger            Logger
	TxRunner          TxRunner
	Repos             RepositorySet
}
