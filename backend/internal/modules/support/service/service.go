// Package service: support service implementation.
package service

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/support/domain"
	"avex-backend/internal/modules/support/events"
	"avex-backend/internal/modules/support/port"
)

type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service {
	return &Service{deps: deps, pool: pool}
}

func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
	return port.EventContext{
		Actor: actor,
		Metadata: port.EventMetadata{
			OccurredAt: s.deps.Clock.Now(),
		},
	}
}

// ===== CreateTicket =====

func (s *Service) CreateTicket(ctx context.Context, input port.CreateTicketInput) (*port.TicketDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	ticketNo := s.deps.TicketNumberGen.Generate()
	if input.CreatedBy == "" {
		input.CreatedBy = "user"
	}

	ticket, err := domain.NewTicket(
		id, ticketNo, input.UserID,
		input.OrderID, input.DriverID, input.RestaurantID,
		input.Subject, input.Description,
		domain.TicketCategory(input.Category),
		domain.TicketPriority(input.Priority),
		input.CreatedBy, now,
	)
	if err != nil {
		return nil, err
	}

	err = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		if err := s.deps.Repos.Tickets.Create(ctx, exec, ticket); err != nil {
			return err
		}
		// Publish ticket.created event
		ec := s.eventContext(ctx, port.ActorContext{Type: "user", ID: input.UserID})
		envelope, err := events.TicketCreatedEnvelope(port.TicketCreatedPayload{
			TicketID: ticket.ID(),
			TicketNo: ticket.TicketNo(),
			UserID:   ticket.UserID(),
			Subject:  ticket.Subject(),
			Category: string(ticket.Category()),
			Priority: string(ticket.Priority()),
			OrderID:  ticket.OrderID(),
		}, ec)
		if err != nil {
			return err
		}
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(ticket), nil
}

// ===== GetTicket / GetTicketByNumber =====

func (s *Service) GetTicket(ctx context.Context, id string) (*port.TicketDTO, error) {
	t, err := s.deps.Repos.Tickets.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*t), nil
}

func (s *Service) GetTicketByNumber(ctx context.Context, ticketNo string) (*port.TicketDTO, error) {
	t, err := s.deps.Repos.Tickets.GetByTicketNo(ctx, s.pool, ticketNo)
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*t), nil
}

// ===== List methods =====

func (s *Service) ListMyTickets(ctx context.Context, userID string, page port.PageQuery) (port.Page[port.TicketDTO], error) {
	result, err := s.deps.Repos.Tickets.ListByUser(ctx, s.pool, userID, page)
	if err != nil {
		return port.Page[port.TicketDTO]{}, err
	}
	return s.toTicketDTOPage(result), nil
}

func (s *Service) ListAgentTickets(ctx context.Context, agentID string, page port.PageQuery) (port.Page[port.TicketDTO], error) {
	result, err := s.deps.Repos.Tickets.ListByAgent(ctx, s.pool, agentID, page)
	if err != nil {
		return port.Page[port.TicketDTO]{}, err
	}
	return s.toTicketDTOPage(result), nil
}

func (s *Service) ListTicketsByStatus(ctx context.Context, status string, page port.PageQuery) (port.Page[port.TicketDTO], error) {
	result, err := s.deps.Repos.Tickets.ListByStatus(ctx, s.pool, status, page)
	if err != nil {
		return port.Page[port.TicketDTO]{}, err
	}
	return s.toTicketDTOPage(result), nil
}

func (s *Service) ListAllTickets(ctx context.Context, page port.PageQuery) (port.Page[port.TicketDTO], error) {
	result, err := s.deps.Repos.Tickets.ListAll(ctx, s.pool, page)
	if err != nil {
		return port.Page[port.TicketDTO]{}, err
	}
	return s.toTicketDTOPage(result), nil
}

func (s *Service) ListUnassignedTickets(ctx context.Context, page port.PageQuery) (port.Page[port.TicketDTO], error) {
	result, err := s.deps.Repos.Tickets.ListUnassigned(ctx, s.pool, page)
	if err != nil {
		return port.Page[port.TicketDTO]{}, err
	}
	return s.toTicketDTOPage(result), nil
}

func (s *Service) toTicketDTOPage(p port.Page[domain.Ticket]) port.Page[port.TicketDTO] {
	dtos := make([]port.TicketDTO, 0, len(p.Items))
	for _, t := range p.Items {
		dtos = append(dtos, port.ToTicketDTO(t))
	}
	return port.Page[port.TicketDTO]{Items: dtos, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

// ===== Assign / Unassign =====

func (s *Service) AssignTicket(ctx context.Context, input port.AssignTicketInput) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, input.TicketID)
		if err != nil {
			return err
		}
		assigned, err := t.Assign(input.AgentID, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, assigned); err != nil {
			return err
		}
		ec := s.eventContext(ctx, port.ActorContext{Type: "agent", ID: input.AgentID})
		envelope, err := events.TicketAssignedEnvelope(port.TicketAssignedPayload{
			TicketID: assigned.ID(), AgentID: input.AgentID,
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}
		updated = &assigned
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

func (s *Service) UnassignTicket(ctx context.Context, ticketID string) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, ticketID)
		if err != nil {
			return err
		}
		unassigned, err := t.Unassign(s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, unassigned); err != nil {
			return err
		}
		updated = &unassigned
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

// ===== SetStatus / SetPriority =====

func (s *Service) SetTicketStatus(ctx context.Context, ticketID, status string) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, ticketID)
		if err != nil {
			return err
		}
		oldStatus := t.Status()
		newStatus, err := t.SetStatus(domain.TicketStatus(status), s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, newStatus); err != nil {
			return err
		}
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.TicketStatusChangedEnvelope(port.TicketStatusChangedPayload{
			TicketID: newStatus.ID(), OldStatus: string(oldStatus), NewStatus: status,
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}
		updated = &newStatus
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

func (s *Service) SetTicketPriority(ctx context.Context, ticketID, priority string) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, ticketID)
		if err != nil {
			return err
		}
		updatedTicket, err := t.SetPriority(domain.TicketPriority(priority), s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, updatedTicket); err != nil {
			return err
		}
		updated = &updatedTicket
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

// ===== Close / Reopen =====

func (s *Service) CloseTicket(ctx context.Context, input port.CloseTicketInput) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, input.TicketID)
		if err != nil {
			return err
		}
		closed, err := t.Close(input.ClosedBy, input.Reason, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, closed); err != nil {
			return err
		}
		ec := s.eventContext(ctx, port.ActorContext{Type: input.ClosedBy})
		envelope, err := events.TicketClosedEnvelope(port.TicketClosedPayload{
			TicketID: closed.ID(), ClosedBy: input.ClosedBy, Reason: input.Reason,
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}
		updated = &closed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

func (s *Service) ReopenTicket(ctx context.Context, ticketID string) (*port.TicketDTO, error) {
	var updated *domain.Ticket
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, ticketID)
		if err != nil {
			return err
		}
		reopened, err := t.Reopen(s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, reopened); err != nil {
			return err
		}
		updated = &reopened
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketDTOPtr(*updated), nil
}

// ===== ReplyToTicket =====

func (s *Service) ReplyToTicket(ctx context.Context, input port.ReplyTicketInput) (*port.TicketMessageDTO, error) {
	var msg *domain.TicketMessage
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		t, err := s.deps.Repos.Tickets.GetByID(ctx, exec, input.TicketID)
		if err != nil {
			return err
		}
		if t.Status().IsTerminal() {
			return domain.ErrTicketClosed
		}
		now := s.deps.Clock.Now()
		msgID := s.deps.IDGenerator.NewID()
		newMsg, err := domain.NewTicketMessage(msgID, input.TicketID, domain.MessageType(input.SenderType), input.SenderID, input.Body, now)
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Messages.Create(ctx, exec, newMsg); err != nil {
			return err
		}
		// Increment message count + update ticket
		updated := t.IncrementMessageCount(now)
		// If agent replies, transition to waiting; if user replies, transition to in_progress
		if input.SenderType == "agent" && t.Status() == domain.StatusInProgress {
			updated, _ = updated.SetStatus(domain.StatusWaiting, now)
		} else if input.SenderType == "user" && t.Status() == domain.StatusWaiting {
			updated, _ = updated.SetStatus(domain.StatusInProgress, now)
		}
		if err := s.deps.Repos.Tickets.Update(ctx, exec, updated); err != nil {
			return err
		}
		// Publish ticket.replied event
		ec := s.eventContext(ctx, port.ActorContext{Type: input.SenderType, ID: input.SenderID})
		envelope, err := events.TicketRepliedEnvelope(port.TicketRepliedPayload{
			TicketID: input.TicketID, MessageID: msgID,
			SenderType: input.SenderType, SenderID: input.SenderID,
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}
		msg = &newMsg
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketMessageDTOPtr(*msg), nil
}

// ===== ListMessages / EditMessage =====

func (s *Service) ListMessages(ctx context.Context, ticketID string, page port.PageQuery) (port.Page[port.TicketMessageDTO], error) {
	result, err := s.deps.Repos.Messages.ListByTicket(ctx, s.pool, ticketID, page)
	if err != nil {
		return port.Page[port.TicketMessageDTO]{}, err
	}
	dtos := make([]port.TicketMessageDTO, 0, len(result.Items))
	for _, m := range result.Items {
		dtos = append(dtos, port.ToTicketMessageDTO(m))
	}
	return port.Page[port.TicketMessageDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

func (s *Service) EditMessage(ctx context.Context, messageID, newBody string) (*port.TicketMessageDTO, error) {
	var updated *domain.TicketMessage
	err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		m, err := s.deps.Repos.Messages.GetByID(ctx, exec, messageID)
		if err != nil {
			return err
		}
		edited, err := m.Edit(newBody, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Messages.Update(ctx, exec, edited); err != nil {
			return err
		}
		updated = &edited
		return nil
	})
	if err != nil {
		return nil, err
	}
	return port.ToTicketMessageDTOPtr(*updated), nil
}

// suppress unused import
var _ = fmt.Sprintf
