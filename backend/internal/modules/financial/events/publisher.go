// Package events implements the financial module's EventPublisher.
//
// The Publisher is STATELESS. Every Publish call saves an EventEnvelope
// to the financial.outbox table. The outbox worker (cmd/worker) is
// responsible for the actual Redis publish.
package events

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/financial/port"
)

// Publisher implements port.EventPublisher.
type Publisher struct {
	repos port.RepositorySet
	idGen port.IDGenerator
}

var _ port.EventPublisher = (*Publisher)(nil)

// NewPublisher creates a stateless Publisher backed by the given outbox repo.
func NewPublisher(repos port.RepositorySet, idGen port.IDGenerator) *Publisher {
	return &Publisher{repos: repos, idGen: idGen}
}

// NewEventPublisher is a convenience wrapper for module.go.
func NewEventPublisher(repos port.RepositorySet, idGen port.IDGenerator) port.EventPublisher {
	return NewPublisher(repos, idGen)
}

// Publish saves an event envelope to the outbox within the given transaction.
func (p *Publisher) Publish(ctx context.Context, exec port.Executor, envelope port.EventEnvelope) error {
	if envelope.EventID == "" {
		envelope.EventID = p.idGen.NewID()
	}
	if envelope.Producer == "" {
		envelope.Producer = "financial"
	}
	if err := p.repos.Outbox.Save(ctx, exec, envelope); err != nil {
		return fmt.Errorf("publish event to outbox: %w", err)
	}
	return nil
}
