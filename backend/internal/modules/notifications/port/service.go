// Package port service: ServicePort + DTOs + event types.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/notifications/domain"
)

// ===== Event Types =====

const (
	EventNotificationSent      = "notifications.notification.sent"
	EventNotificationFailed    = "notifications.notification.failed"
	EventNotificationDelivered = "notifications.notification.delivered"
)

const (
	NotificationSentEventVersion      = 1
	NotificationSentSchemaVersion     = 1
	NotificationFailedEventVersion    = 1
	NotificationFailedSchemaVersion   = 1
	NotificationDeliveredEventVersion = 1
	NotificationDeliveredSchemaVersion = 1
)

// ===== Event Payloads =====

type NotificationSentPayload struct {
	NotificationID string `json:"notification_id"`
	RecipientID    string `json:"recipient_id"`
	RecipientType  string `json:"recipient_type"`
	Channel        string `json:"channel"`
	Type           string `json:"type"`
}

type NotificationFailedPayload struct {
	NotificationID string `json:"notification_id"`
	RecipientID    string `json:"recipient_id"`
	Channel        string `json:"channel"`
	Error          string `json:"error"`
	RetryCount     int    `json:"retry_count"`
}

type NotificationDeliveredPayload struct {
	NotificationID string `json:"notification_id"`
	Channel        string `json:"channel"`
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

func BuildEnvelope(
	eventID string,
	eventType string,
	eventVersion int,
	schemaVersion int,
	payload []byte,
	ec EventContext,
) EventEnvelope {
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
		Producer:      "notifications",
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

type SendNotificationInput struct {
	RecipientType string // user | driver | merchant
	RecipientID   string
	Type          string // notification type
	Channels      []string // push | sms | email (will be filtered by preferences)
	Title         string
	TitleAr       string
	Body          string
	BodyAr        string
	Data          map[string]any
	Priority      string // normal | high
}

type NotificationDTO struct {
	ID            string     `json:"id"`
	RecipientType string     `json:"recipient_type"`
	RecipientID   string     `json:"recipient_id"`
	Type          string     `json:"type"`
	Channel       string     `json:"channel"`
	Title         string     `json:"title"`
	TitleAr       string     `json:"title_ar,omitempty"`
	Body          string     `json:"body"`
	BodyAr        string     `json:"body_ar,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	RetryCount    int        `json:"retry_count"`
	LastError     string     `json:"last_error,omitempty"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type PreferenceDTO struct {
	ID            string         `json:"id"`
	RecipientType string         `json:"recipient_type"`
	RecipientID   string         `json:"recipient_id"`
	PhoneNumber   string         `json:"phone_number,omitempty"`
	Email         string         `json:"email,omitempty"`
	DeviceTokens  []string       `json:"device_tokens,omitempty"`
	Prefs         map[string]bool `json:"prefs"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type UpdatePreferenceInput struct {
	RecipientType string
	RecipientID   string
	PhoneNumber   string
	Email         string
	DeviceToken   string // optional — add this device token
	RemoveToken   string // optional — remove this device token
	// Per-category, per-channel enable flags.
	// Key format: "orders:push", "dispatch:sms", etc.
	// Only the keys present in this map will be updated.
	Prefs map[string]bool
}

// ===== ServicePort =====

type ServicePort interface {
	// SendNotification creates + sends a notification via all enabled channels.
	// It checks the recipient's preferences and only sends via channels they've enabled.
	// Returns the created notification IDs (one per channel).
	SendNotification(ctx context.Context, input SendNotificationInput) ([]NotificationDTO, error)

	// GetNotification retrieves a notification by ID.
	GetNotification(ctx context.Context, id string) (*NotificationDTO, error)

	// ListNotificationsByRecipient retrieves notifications for a recipient.
	ListNotificationsByRecipient(ctx context.Context, recipientType, recipientID string, page PageQuery) (Page[NotificationDTO], error)

	// RetryFailed re-queues failed notifications for retry.
	RetryFailed(ctx context.Context, notificationID string) (*NotificationDTO, error)

	// ProcessPending sends all pending notifications (called by a background worker).
	// Returns the number of notifications processed.
	ProcessPending(ctx context.Context, limit int) (int, error)

	// ===== Preferences =====
	GetPreferences(ctx context.Context, recipientType, recipientID string) (*PreferenceDTO, error)
	UpdatePreferences(ctx context.Context, input UpdatePreferenceInput) (*PreferenceDTO, error)
}

// ===== Domain → DTO Mappers =====

func ToNotificationDTO(n domain.Notification) NotificationDTO {
	return NotificationDTO{
		ID:            n.ID(),
		RecipientType: n.RecipientType(),
		RecipientID:   n.RecipientID(),
		Type:          string(n.Type()),
		Channel:       string(n.Channel()),
		Title:         n.Title(),
		TitleAr:       n.TitleAr(),
		Body:          n.Body(),
		BodyAr:        n.BodyAr(),
		Data:          n.Data(),
		Status:        string(n.Status()),
		Priority:      n.Priority(),
		RetryCount:    n.RetryCount(),
		LastError:     n.LastError(),
		ScheduledAt:   n.ScheduledAt(),
		SentAt:        n.SentAt(),
		DeliveredAt:   n.DeliveredAt(),
		CreatedAt:     n.CreatedAt(),
	}
}

func ToPreferenceDTO(p domain.UserNotificationPreferences) PreferenceDTO {
	return PreferenceDTO{
		ID:            p.ID(),
		RecipientType: p.RecipientType(),
		RecipientID:   p.RecipientID(),
		PhoneNumber:   p.PhoneNumber(),
		Email:         p.Email(),
		DeviceTokens:  p.DeviceTokens(),
		Prefs:         p.AllPrefs(),
		CreatedAt:     p.CreatedAt(),
		UpdatedAt:     p.UpdatedAt(),
	}
}

func ToNotificationDTOPtr(n domain.Notification) *NotificationDTO {
	dto := ToNotificationDTO(n)
	return &dto
}

func ToPreferenceDTOPtr(p domain.UserNotificationPreferences) *PreferenceDTO {
	dto := ToPreferenceDTO(p)
	return &dto
}
