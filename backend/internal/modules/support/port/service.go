// Package port service: ServicePort + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/support/domain"
)

// ===== Event Types =====

const (
	EventTicketCreated       = "support.ticket.created"
	EventTicketAssigned      = "support.ticket.assigned"
	EventTicketReplied       = "support.ticket.replied"
	EventTicketStatusChanged = "support.ticket.status_changed"
	EventTicketClosed        = "support.ticket.closed"
)

const (
	TicketCreatedEventVersion      = 1
	TicketCreatedSchemaVersion     = 1
	TicketAssignedEventVersion     = 1
	TicketAssignedSchemaVersion    = 1
	TicketRepliedEventVersion      = 1
	TicketRepliedSchemaVersion     = 1
	TicketStatusChangedEventVersion = 1
	TicketStatusChangedSchemaVersion = 1
	TicketClosedEventVersion       = 1
	TicketClosedSchemaVersion      = 1
)

// ===== Event Payloads =====

type TicketCreatedPayload struct {
	TicketID   string `json:"ticket_id"`
	TicketNo   string `json:"ticket_no"`
	UserID     string `json:"user_id"`
	Subject    string `json:"subject"`
	Category   string `json:"category"`
	Priority   string `json:"priority"`
	OrderID    string `json:"order_id,omitempty"`
}

type TicketAssignedPayload struct {
	TicketID string `json:"ticket_id"`
	AgentID  string `json:"agent_id"`
}

type TicketRepliedPayload struct {
	TicketID    string `json:"ticket_id"`
	MessageID   string `json:"message_id"`
	SenderType  string `json:"sender_type"`
	SenderID    string `json:"sender_id"`
}

type TicketStatusChangedPayload struct {
	TicketID  string `json:"ticket_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type TicketClosedPayload struct {
	TicketID  string `json:"ticket_id"`
	ClosedBy  string `json:"closed_by"`
	Reason    string `json:"reason"`
}

type EventMetadata struct {
	CorrelationID string
	TraceID       string
	OccurredAt    time.Time
}

type EventContext struct {
	Actor    ActorContext
	Metadata EventMetadata
}

func BuildEnvelope(eventID, eventType string, eventVersion, schemaVersion int, payload []byte, ec EventContext) EventEnvelope {
	occurredAt := ec.Metadata.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return EventEnvelope{
		EventID:       eventID,
		EventType:     eventType,
		EventVersion:  eventVersion,
		SchemaVersion: schemaVersion,
		OccurredAt:    occurredAt,
		Producer:      "support",
		CorrelationID: ec.Metadata.CorrelationID,
		TraceID:       ec.Metadata.TraceID,
		ActorType:     ec.Actor.Type,
		ActorID:       ec.Actor.ID,
		ActorIP:       ec.Actor.IP,
		ActorUA:       ec.Actor.UserAgent,
		Payload:       payload,
	}
}

// ===== DTOs =====

type CreateTicketInput struct {
	UserID        string
	OrderID       string
	DriverID      string
	RestaurantID  string
	Subject       string
	Description   string
	Category      string
	Priority      string
	CreatedBy     string
}

type ReplyTicketInput struct {
	TicketID   string
	SenderType string // user | agent | internal
	SenderID   string
	Body       string
}

type AssignTicketInput struct {
	TicketID string
	AgentID  string
}

type CloseTicketInput struct {
	TicketID string
	ClosedBy string
	Reason   string
}

type TicketDTO struct {
	ID              string     `json:"id"`
	TicketNo        string     `json:"ticket_no"`
	UserID          string     `json:"user_id"`
	OrderID         string     `json:"order_id,omitempty"`
	DriverID        string     `json:"driver_id,omitempty"`
	RestaurantID    string     `json:"restaurant_id,omitempty"`
	Subject         string     `json:"subject"`
	Description     string     `json:"description"`
	Category        string     `json:"category"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	AssignedTo      string     `json:"assigned_to,omitempty"`
	CreatedBy       string     `json:"created_by"`
	ClosedBy        string     `json:"closed_by,omitempty"`
	ClosedReason    string     `json:"closed_reason,omitempty"`
	MessageCount    int        `json:"message_count"`
	FirstResponseAt *time.Time `json:"first_response_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type TicketMessageDTO struct {
	ID         string     `json:"id"`
	TicketID   string     `json:"ticket_id"`
	SenderType string     `json:"sender_type"`
	SenderID   string     `json:"sender_id"`
	Body       string     `json:"body"`
	EditedAt   *time.Time `json:"edited_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type TicketAttachmentDTO struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	FileName  string    `json:"file_name"`
	FileType  string    `json:"file_type"`
	FileURL   string    `json:"file_url"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// ===== ServicePort =====

type ServicePort interface {
	// ===== Ticket CRUD =====
	CreateTicket(ctx context.Context, input CreateTicketInput) (*TicketDTO, error)
	GetTicket(ctx context.Context, id string) (*TicketDTO, error)
	GetTicketByNumber(ctx context.Context, ticketNo string) (*TicketDTO, error)
	ListMyTickets(ctx context.Context, userID string, page PageQuery) (Page[TicketDTO], error)
	ListAgentTickets(ctx context.Context, agentID string, page PageQuery) (Page[TicketDTO], error)
	ListTicketsByStatus(ctx context.Context, status string, page PageQuery) (Page[TicketDTO], error)
	ListAllTickets(ctx context.Context, page PageQuery) (Page[TicketDTO], error)
	ListUnassignedTickets(ctx context.Context, page PageQuery) (Page[TicketDTO], error)

	// ===== Ticket Lifecycle =====
	AssignTicket(ctx context.Context, input AssignTicketInput) (*TicketDTO, error)
	UnassignTicket(ctx context.Context, ticketID string) (*TicketDTO, error)
	SetTicketStatus(ctx context.Context, ticketID, status string) (*TicketDTO, error)
	SetTicketPriority(ctx context.Context, ticketID, priority string) (*TicketDTO, error)
	CloseTicket(ctx context.Context, input CloseTicketInput) (*TicketDTO, error)
	ReopenTicket(ctx context.Context, ticketID string) (*TicketDTO, error)

	// ===== Messages =====
	ReplyToTicket(ctx context.Context, input ReplyTicketInput) (*TicketMessageDTO, error)
	ListMessages(ctx context.Context, ticketID string, page PageQuery) (Page[TicketMessageDTO], error)
	EditMessage(ctx context.Context, messageID, newBody string) (*TicketMessageDTO, error)
}

// ===== Mappers =====

func ToTicketDTO(t domain.Ticket) TicketDTO {
	return TicketDTO{
		ID:              t.ID(),
		TicketNo:        t.TicketNo(),
		UserID:          t.UserID(),
		OrderID:         t.OrderID(),
		DriverID:        t.DriverID(),
		RestaurantID:    t.RestaurantID(),
		Subject:         t.Subject(),
		Description:     t.Description(),
		Category:        string(t.Category()),
		Priority:        string(t.Priority()),
		Status:          string(t.Status()),
		AssignedTo:      t.AssignedTo(),
		CreatedBy:       t.CreatedBy(),
		ClosedBy:        t.ClosedBy(),
		ClosedReason:    t.ClosedReason(),
		MessageCount:    t.MessageCount(),
		FirstResponseAt: t.FirstResponseAt(),
		ResolvedAt:      t.ResolvedAt(),
		ClosedAt:        t.ClosedAt(),
		CreatedAt:       t.CreatedAt(),
		UpdatedAt:       t.UpdatedAt(),
	}
}

func ToTicketMessageDTO(m domain.TicketMessage) TicketMessageDTO {
	return TicketMessageDTO{
		ID:         m.ID(),
		TicketID:   m.TicketID(),
		SenderType: string(m.SenderType()),
		SenderID:   m.SenderID(),
		Body:       m.Body(),
		EditedAt:   m.EditedAt(),
		CreatedAt:  m.CreatedAt(),
	}
}

func ToTicketAttachmentDTO(a domain.TicketAttachment) TicketAttachmentDTO {
	return TicketAttachmentDTO{
		ID:        a.ID(),
		MessageID: a.MessageID(),
		FileName:  a.FileName(),
		FileType:  a.FileType(),
		FileURL:   a.FileURL(),
		FileSize:  a.FileSize(),
		CreatedAt: a.CreatedAt(),
	}
}

func ToTicketDTOPtr(t domain.Ticket) *TicketDTO {
	dto := ToTicketDTO(t)
	return &dto
}

func ToTicketMessageDTOPtr(m domain.TicketMessage) *TicketMessageDTO {
	dto := ToTicketMessageDTO(m)
	return &dto
}
