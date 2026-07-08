// Package port events: event type constants and payloads for the dispatch module.
package port

import "time"

// ===== Event Type Constants =====

const (
	EventDriverRegistered    = "dispatch.driver.registered"
	EventDriverWentOnline    = "dispatch.driver.online"
	EventDriverWentOffline   = "dispatch.driver.offline"
	EventDriverSuspended     = "dispatch.driver.suspended"
	EventDriverLocationUpdated = "dispatch.driver.location_updated"
	EventOfferCreated        = "dispatch.offer.created"
	EventOfferAccepted       = "dispatch.offer.accepted"
	EventOfferRejected       = "dispatch.offer.rejected"
	EventOfferExpired        = "dispatch.offer.expired"
	EventOfferCancelled      = "dispatch.offer.cancelled"
	EventDispatchFailed      = "dispatch.dispatch_failed"
)

// ===== Event Versions =====

const (
	DriverRegisteredEventVersion    = 1
	DriverRegisteredSchemaVersion   = 1
	DriverWentOnlineEventVersion    = 1
	DriverWentOnlineSchemaVersion   = 1
	DriverWentOfflineEventVersion   = 1
	DriverWentOfflineSchemaVersion  = 1
	DriverSuspendedEventVersion     = 1
	DriverSuspendedSchemaVersion    = 1
	DriverLocationUpdatedEventVersion  = 1
	DriverLocationUpdatedSchemaVersion = 1
	OfferCreatedEventVersion        = 1
	OfferCreatedSchemaVersion       = 1
	OfferAcceptedEventVersion       = 1
	OfferAcceptedSchemaVersion      = 1
	OfferRejectedEventVersion       = 1
	OfferRejectedSchemaVersion      = 1
	OfferExpiredEventVersion        = 1
	OfferExpiredSchemaVersion       = 1
	OfferCancelledEventVersion      = 1
	OfferCancelledSchemaVersion     = 1
	DispatchFailedEventVersion      = 1
	DispatchFailedSchemaVersion     = 1
)

// ===== Event Payloads =====

type DriverRegisteredPayload struct {
	DriverID    string `json:"driver_id"`
	UserID      string `json:"user_id"`
	VehicleType string `json:"vehicle_type"`
}

type DriverWentOnlinePayload struct {
	DriverID string `json:"driver_id"`
	ZoneIDs  []string `json:"zone_ids,omitempty"`
}

type DriverWentOfflinePayload struct {
	DriverID string `json:"driver_id"`
}

type DriverSuspendedPayload struct {
	DriverID string `json:"driver_id"`
	Reason   string `json:"reason"`
}

type DriverLocationUpdatedPayload struct {
	DriverID   string  `json:"driver_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Bearing    float64 `json:"bearing"`
	Speed      float64 `json:"speed"`
	CapturedAt string  `json:"captured_at"` // RFC3339
}

type OfferCreatedPayload struct {
	OfferID       string `json:"offer_id"`
	OrderID       string `json:"order_id"`
	DriverID      string `json:"driver_id"`
	AttemptNumber int    `json:"attempt_number"`
	ExpiresAt     string `json:"expires_at"` // RFC3339
	EstDistanceM  *int   `json:"est_distance_m,omitempty"`
	EstDurationS  *int   `json:"est_duration_s,omitempty"`
	EstFareCents  *int64 `json:"est_fare_cents,omitempty"`
	Currency      string `json:"currency,omitempty"`
}

type OfferAcceptedPayload struct {
	OfferID  string `json:"offer_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
}

type OfferRejectedPayload struct {
	OfferID  string `json:"offer_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
	Reason   string `json:"reason,omitempty"`
}

type OfferExpiredPayload struct {
	OfferID  string `json:"offer_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
}

type OfferCancelledPayload struct {
	OfferID  string `json:"offer_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
}

type DispatchFailedPayload struct {
	OrderID     string `json:"order_id"`
	ZoneID      string `json:"zone_id,omitempty"`
	Reason      string `json:"reason"`
	AttemptCount int   `json:"attempt_count"`
}

// ===== Event Context =====

type EventMetadata struct {
	CorrelationID string
	TraceID       string
	OccurredAt    time.Time
}

type EventContext struct {
	Actor    ActorContext
	Metadata EventMetadata
}

// BuildEnvelope constructs an EventEnvelope.
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
		Producer:      "dispatch",
		CorrelationID: ec.Metadata.CorrelationID,
		TraceID:       ec.Metadata.TraceID,
		ActorType:     ec.Actor.Type,
		ActorID:       ec.Actor.ID,
		ActorIP:       ec.Actor.IP,
		ActorUA:       ec.Actor.UserAgent,
		Payload:       payload,
	}
}
