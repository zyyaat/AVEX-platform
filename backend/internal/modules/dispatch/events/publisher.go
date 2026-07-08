// Package events implements the dispatch module's EventPublisher.
package events

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/dispatch/port"
)

// Publisher implements port.EventPublisher.
type Publisher struct {
	repos port.RepositorySet
	idGen port.IDGenerator
}

var _ port.EventPublisher = (*Publisher)(nil)

// NewPublisher creates a stateless Publisher.
func NewPublisher(repos port.RepositorySet, idGen port.IDGenerator) *Publisher {
	return &Publisher{repos: repos, idGen: idGen}
}

// NewEventPublisher is a convenience wrapper for module.go.
func NewEventPublisher(repos port.RepositorySet, idGen port.IDGenerator) port.EventPublisher {
	return NewPublisher(repos, idGen)
}

// Publish saves an event envelope to the outbox.
func (p *Publisher) Publish(ctx context.Context, exec port.Executor, envelope port.EventEnvelope) error {
	if envelope.EventID == "" {
		envelope.EventID = p.idGen.NewID()
	}
	if envelope.Producer == "" {
		envelope.Producer = "dispatch"
	}
	if err := p.repos.Outbox.Save(ctx, exec, envelope); err != nil {
		return fmt.Errorf("publish to outbox: %w", err)
	}
	return nil
}
