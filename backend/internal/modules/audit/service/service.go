// Package service: audit service implementation.
package service

import (
	"context"
	"time"

	"avex-backend/internal/modules/audit/domain"
	"avex-backend/internal/modules/audit/port"
)

type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service { return &Service{deps: deps, pool: pool} }

// ===== Log =====

func (s *Service) Log(ctx context.Context, input port.LogActionInput) (*port.AuditEntryDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()

	entry, err := domain.NewAuditEntry(
		id,
		domain.ActorType(input.ActorType),
		input.ActorID,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		domain.Severity(input.Severity),
		input.Description,
		input.Metadata,
		input.IPAddress,
		input.UserAgent,
		input.CorrelationID,
		input.TraceID,
		now,
	)
	if err != nil { return nil, err }

	if err := s.deps.Repos.Audit.Create(ctx, s.pool, entry); err != nil { return nil, err }

	return port.ToAuditEntryDTOPtr(entry), nil
}

// ===== GetEntry =====

func (s *Service) GetEntry(ctx context.Context, id string) (*port.AuditEntryDTO, error) {
	entry, err := s.deps.Repos.Audit.GetByID(ctx, s.pool, id)
	if err != nil { return nil, err }
	return port.ToAuditEntryDTOPtr(*entry), nil
}

// ===== Query methods =====

func (s *Service) ListByActor(ctx context.Context, actorType, actorID string, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListByActor(ctx, s.pool, actorType, actorID, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) ListByResource(ctx context.Context, resourceType, resourceID string, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListByResource(ctx, s.pool, resourceType, resourceID, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) ListByAction(ctx context.Context, action string, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListByAction(ctx, s.pool, action, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) ListBySeverity(ctx context.Context, severity string, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListBySeverity(ctx, s.pool, severity, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) ListByTimeRange(ctx context.Context, from, to time.Time, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListByTimeRange(ctx, s.pool, from, to, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) ListAll(ctx context.Context, page port.PageQuery) (port.Page[port.AuditEntryDTO], error) {
	result, err := s.deps.Repos.Audit.ListAll(ctx, s.pool, page)
	if err != nil { return port.Page[port.AuditEntryDTO]{}, err }
	return s.toDTOPage(result), nil
}

func (s *Service) toDTOPage(p port.Page[domain.AuditEntry]) port.Page[port.AuditEntryDTO] {
	dtos := make([]port.AuditEntryDTO, 0, len(p.Items))
	for _, e := range p.Items { dtos = append(dtos, port.ToAuditEntryDTO(e)) }
	return port.Page[port.AuditEntryDTO]{Items: dtos, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

// ===== Stats =====

func (s *Service) GetStats(ctx context.Context, from, to time.Time) (*port.AuditStatsDTO, error) {
	byAction, err := s.deps.Repos.Audit.CountByAction(ctx, s.pool, from, to)
	if err != nil { return nil, err }
	var total int64
	for _, cnt := range byAction { total += cnt }
	return &port.AuditStatsDTO{
		TotalEntries: total,
		ByAction:     byAction,
		FromTime:     from,
		ToTime:       to,
	}, nil
}
