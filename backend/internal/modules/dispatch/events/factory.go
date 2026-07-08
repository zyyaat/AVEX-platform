// Package events factory: convenience constructors for dispatch EventEnvelopes.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/dispatch/port"
)

// BuildEnvelope constructs an EventEnvelope from a typed payload.
func BuildEnvelope(
	eventType string,
	eventVersion int,
	schemaVersion int,
	payload any,
	ec port.EventContext,
) (port.EventEnvelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return port.EventEnvelope{}, err
	}
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, payloadBytes, ec), nil
}

// ===== Per-Event Convenience Functions =====

func DriverRegisteredEnvelope(payload port.DriverRegisteredPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDriverRegistered, port.DriverRegisteredEventVersion, port.DriverRegisteredSchemaVersion, payload, ec)
}

func DriverWentOnlineEnvelope(payload port.DriverWentOnlinePayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDriverWentOnline, port.DriverWentOnlineEventVersion, port.DriverWentOnlineSchemaVersion, payload, ec)
}

func DriverWentOfflineEnvelope(payload port.DriverWentOfflinePayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDriverWentOffline, port.DriverWentOfflineEventVersion, port.DriverWentOfflineSchemaVersion, payload, ec)
}

func DriverSuspendedEnvelope(payload port.DriverSuspendedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDriverSuspended, port.DriverSuspendedEventVersion, port.DriverSuspendedSchemaVersion, payload, ec)
}

func DriverLocationUpdatedEnvelope(payload port.DriverLocationUpdatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDriverLocationUpdated, port.DriverLocationUpdatedEventVersion, port.DriverLocationUpdatedSchemaVersion, payload, ec)
}

func OfferCreatedEnvelope(payload port.OfferCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventOfferCreated, port.OfferCreatedEventVersion, port.OfferCreatedSchemaVersion, payload, ec)
}

func OfferAcceptedEnvelope(payload port.OfferAcceptedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventOfferAccepted, port.OfferAcceptedEventVersion, port.OfferAcceptedSchemaVersion, payload, ec)
}

func OfferRejectedEnvelope(payload port.OfferRejectedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventOfferRejected, port.OfferRejectedEventVersion, port.OfferRejectedSchemaVersion, payload, ec)
}

func OfferExpiredEnvelope(payload port.OfferExpiredPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventOfferExpired, port.OfferExpiredEventVersion, port.OfferExpiredSchemaVersion, payload, ec)
}

func OfferCancelledEnvelope(payload port.OfferCancelledPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventOfferCancelled, port.OfferCancelledEventVersion, port.OfferCancelledSchemaVersion, payload, ec)
}

func DispatchFailedEnvelope(payload port.DispatchFailedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventDispatchFailed, port.DispatchFailedEventVersion, port.DispatchFailedSchemaVersion, payload, ec)
}
