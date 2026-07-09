// Package port service: ServicePort + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/audit/domain"
)

// ===== Events =====
const (
	EventAuditEntryCreated = "audit.entry.created"
)
const (
	AuditEntryCreatedEventVersion = 1
	AuditEntryCreatedSchemaVersion = 1
)

type AuditEntryCreatedPayload struct {
	EntryID      string `json:"entry_id"`
	ActorType    string `json:"actor_type"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Severity     string `json:"severity"`
}

type EventMetadata struct{ CorrelationID, TraceID string; OccurredAt time.Time }
type EventContext struct{ Actor ActorContext; Metadata EventMetadata }

func BuildEnvelope(eventID, eventType string, eventVersion, schemaVersion int, payload []byte, ec EventContext) EventEnvelope {
	occurredAt := ec.Metadata.OccurredAt
	if occurredAt.IsZero() { occurredAt = time.Now().UTC() }
	return EventEnvelope{
		EventID: eventID, EventType: eventType, EventVersion: eventVersion, SchemaVersion: schemaVersion,
		OccurredAt: occurredAt, Producer: "audit",
		CorrelationID: ec.Metadata.CorrelationID, TraceID: ec.Metadata.TraceID,
		ActorType: ec.Actor.Type, ActorID: ec.Actor.ID, ActorIP: ec.Actor.IP, ActorUA: ec.Actor.UserAgent,
		Payload: payload,
	}
}

// ===== DTOs =====
type LogActionInput struct {
	ActorType    string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	Severity     string
	Description  string
	Metadata     map[string]any
	IPAddress    string
	UserAgent    string
	CorrelationID string
	TraceID      string
}

type AuditEntryDTO struct {
	ID            string         `json:"id"`
	ActorType     string         `json:"actor_type"`
	ActorID       string         `json:"actor_id"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id"`
	Severity      string         `json:"severity"`
	Description   string         `json:"description,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
	IPAddress     string         `json:"ip_address,omitempty"`
	UserAgent     string         `json:"user_agent,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type AuditStatsDTO struct {
	TotalEntries int64            `json:"total_entries"`
	ByAction     map[string]int64 `json:"by_action"`
	FromTime     time.Time        `json:"from_time"`
	ToTime       time.Time        `json:"to_time"`
}

// ===== ServicePort =====
type ServicePort interface {
	// Log records a new audit entry (append-only).
	Log(ctx context.Context, input LogActionInput) (*AuditEntryDTO, error)

	// GetEntry retrieves a single audit entry by ID.
	GetEntry(ctx context.Context, id string) (*AuditEntryDTO, error)

	// Query methods
	ListByActor(ctx context.Context, actorType, actorID string, page PageQuery) (Page[AuditEntryDTO], error)
	ListByResource(ctx context.Context, resourceType, resourceID string, page PageQuery) (Page[AuditEntryDTO], error)
	ListByAction(ctx context.Context, action string, page PageQuery) (Page[AuditEntryDTO], error)
	ListBySeverity(ctx context.Context, severity string, page PageQuery) (Page[AuditEntryDTO], error)
	ListByTimeRange(ctx context.Context, from, to time.Time, page PageQuery) (Page[AuditEntryDTO], error)
	ListAll(ctx context.Context, page PageQuery) (Page[AuditEntryDTO], error)

	// Stats
	GetStats(ctx context.Context, from, to time.Time) (*AuditStatsDTO, error)
}

// ===== Mappers =====
func ToAuditEntryDTO(e domain.AuditEntry) AuditEntryDTO {
	return AuditEntryDTO{
		ID: e.ID(), ActorType: string(e.ActorType()), ActorID: e.ActorID(),
		Action: e.Action(), ResourceType: e.ResourceType(), ResourceID: e.ResourceID(),
		Severity: string(e.Severity()), Description: e.Description(), Metadata: e.Metadata(),
		IPAddress: e.IPAddress(), UserAgent: e.UserAgent(),
		CorrelationID: e.CorrelationID(), TraceID: e.TraceID(), CreatedAt: e.CreatedAt(),
	}
}

func ToAuditEntryDTOPtr(e domain.AuditEntry) *AuditEntryDTO {
	dto := ToAuditEntryDTO(e)
	return &dto
}
