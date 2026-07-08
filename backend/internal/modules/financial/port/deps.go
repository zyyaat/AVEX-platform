// Package port deps: dependency interfaces and the Deps struct.
//
// The Deps struct holds all dependencies the financial service layer needs.
// Each dependency is an interface defined HERE (in port/) — not imported
// from platform/. This is true dependency inversion.
//
// Imports: stdlib only.
package port

import (
	"context"
	"time"
)

// ===== Infrastructure Dependencies =====

// Clock provides the current time. All service code depends on this
// interface, not on time.Now() directly, for testability.
type Clock interface {
	Now() time.Time
}

// IDGenerator generates unique IDs (UUIDs).
type IDGenerator interface {
	NewID() string
}

// Logger is a minimal logging interface.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// ===== EventPublisher Dependency =====

// EventEnvelope is the wire format for events published to the outbox.
type EventEnvelope struct {
	EventID       string
	EventType     string
	EventVersion  int
	SchemaVersion int
	OccurredAt    time.Time
	Producer      string // always "financial"
	CorrelationID string
	TraceID       string
	ActorType     string
	ActorID       string
	ActorIP       string
	ActorUA       string
	Payload       []byte // JSON-marshaled payload
}

// EventPublisher publishes financial events to the outbox within the
// current transaction.
type EventPublisher interface {
	Publish(ctx context.Context, exec Executor, envelope EventEnvelope) error
}

// ===== Actor Context =====

// ActorContext carries actor information for event metadata.
type ActorContext struct {
	Type      string // user | driver | merchant | support | system
	ID        string
	IP        string
	UserAgent string
}

// ===== Deps Struct =====

// Deps holds all dependencies the financial service layer needs.
type Deps struct {
	Clock          Clock
	IDGenerator    IDGenerator
	EventPublisher EventPublisher
	Logger         Logger
	TxRunner       TxRunner
	Repos          RepositorySet
}

// ===== helper to suppress unused import warning if all other time refs removed =====
var _ = context.Background
