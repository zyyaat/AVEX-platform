// Package events factory: convenience constructors for notification EventEnvelopes.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/notifications/port"
)

func BuildEnvelope(eventType string, eventVersion int, schemaVersion int, payload any, ec port.EventContext) (port.EventEnvelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return port.EventEnvelope{}, err
	}
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, b, ec), nil
}

func NotificationSentEnvelope(payload port.NotificationSentPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventNotificationSent, port.NotificationSentEventVersion, port.NotificationSentSchemaVersion, payload, ec)
}

func NotificationFailedEnvelope(payload port.NotificationFailedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventNotificationFailed, port.NotificationFailedEventVersion, port.NotificationFailedSchemaVersion, payload, ec)
}

func NotificationDeliveredEnvelope(payload port.NotificationDeliveredPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventNotificationDelivered, port.NotificationDeliveredEventVersion, port.NotificationDeliveredSchemaVersion, payload, ec)
}
