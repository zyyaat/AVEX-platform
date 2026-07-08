// Package events factory: convenience constructors for support EventEnvelopes.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/support/port"
)

func BuildEnvelope(eventType string, eventVersion int, schemaVersion int, payload any, ec port.EventContext) (port.EventEnvelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return port.EventEnvelope{}, err
	}
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, b, ec), nil
}

func TicketCreatedEnvelope(payload port.TicketCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTicketCreated, port.TicketCreatedEventVersion, port.TicketCreatedSchemaVersion, payload, ec)
}

func TicketAssignedEnvelope(payload port.TicketAssignedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTicketAssigned, port.TicketAssignedEventVersion, port.TicketAssignedSchemaVersion, payload, ec)
}

func TicketRepliedEnvelope(payload port.TicketRepliedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTicketReplied, port.TicketRepliedEventVersion, port.TicketRepliedSchemaVersion, payload, ec)
}

func TicketStatusChangedEnvelope(payload port.TicketStatusChangedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTicketStatusChanged, port.TicketStatusChangedEventVersion, port.TicketStatusChangedSchemaVersion, payload, ec)
}

func TicketClosedEnvelope(payload port.TicketClosedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTicketClosed, port.TicketClosedEventVersion, port.TicketClosedSchemaVersion, payload, ec)
}
