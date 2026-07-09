// Package events implements the permissions module's EventPublisher.
package events

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/permissions/port"
)

type Publisher struct {
	repos port.RepositorySet
	idGen port.IDGenerator
}

var _ port.EventPublisher = (*Publisher)(nil)

func NewPublisher(repos port.RepositorySet, idGen port.IDGenerator) *Publisher {
	return &Publisher{repos: repos, idGen: idGen}
}

func NewEventPublisher(repos port.RepositorySet, idGen port.IDGenerator) port.EventPublisher {
	return NewPublisher(repos, idGen)
}

func (p *Publisher) Publish(ctx context.Context, exec port.Executor, envelope port.EventEnvelope) error {
	if envelope.EventID == "" { envelope.EventID = p.idGen.NewID() }
	if envelope.Producer == "" { envelope.Producer = "permissions" }
	if err := p.repos.Outbox.Save(ctx, exec, envelope); err != nil {
		return fmt.Errorf("publish to outbox: %w", err)
	}
	return nil
}
