// Package port deps: dependency interfaces and the Deps struct.
package port

import (
	"context"
	"time"
)

// Clock provides the current time.
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

// ===== Mapbox / Distance Matrix Dependency =====

// DistanceMatrixProvider computes driving distances and durations between
// origins and destinations. Implemented by the mapbox adapter.
//
// This is a port interface (not a platform interface) so the dispatch module
// can swap providers without affecting the service layer.
type DistanceMatrixProvider interface {
	// GetDistanceMatrix returns a matrix of driving distances (meters) and
	// durations (seconds) from each origin to each destination.
	// Origins and destinations are [lat, lng] pairs.
	// Returns:
	//   distances[i][j] = driving distance from origin i to destination j (meters)
	//   durations[i][j]  = driving duration from origin i to destination j (seconds)
	GetDistanceMatrix(ctx context.Context, origins [][2]float64, destinations [][2]float64) (distances [][]int, durations [][]int, err error)
}

// ===== EventPublisher Dependency =====

// EventEnvelope is the wire format for events published to the outbox.
type EventEnvelope struct {
	EventID       string
	EventType     string
	EventVersion  int
	SchemaVersion int
	OccurredAt    time.Time
	Producer      string // always "dispatch"
	CorrelationID string
	TraceID       string
	ActorType     string
	ActorID       string
	ActorIP       string
	ActorUA       string
	Payload       []byte
}

// EventPublisher publishes dispatch events to the outbox within the
// current transaction.
type EventPublisher interface {
	Publish(ctx context.Context, exec Executor, envelope EventEnvelope) error
}

// ActorContext carries actor information for event metadata.
type ActorContext struct {
	Type      string
	ID        string
	IP        string
	UserAgent string
}

// ===== Deps Struct =====

type Deps struct {
	Clock                Clock
	IDGenerator          IDGenerator
	EventPublisher       EventPublisher
	Logger               Logger
	TxRunner             TxRunner
	Repos                RepositorySet
	DistanceMatrixProvider DistanceMatrixProvider
}

// suppress unused import
var _ = context.Background
