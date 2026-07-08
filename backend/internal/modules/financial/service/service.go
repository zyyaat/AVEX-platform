// Package service is the financial module's service layer.
package service

import (
	"context"

	"avex-backend/internal/modules/financial/port"
)

// Service implements port.ServicePort.
type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

// New creates a new financial Service.
func New(deps port.Deps, pool port.Executor) *Service {
	return &Service{deps: deps, pool: pool}
}

// eventContext builds an EventContext from the request context + actor.
func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
	return port.EventContext{
		Actor: actor,
		Metadata: port.EventMetadata{
			OccurredAt: s.deps.Clock.Now(),
		},
	}
}
