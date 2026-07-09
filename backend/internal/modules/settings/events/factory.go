// Package events factory: convenience constructors.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/settings/port"
)

func BuildEnvelope(eventType string, eventVersion, schemaVersion int, payload any, ec port.EventContext) (port.EventEnvelope, error) {
	b, err := json.Marshal(payload)
	if err != nil { return port.EventEnvelope{}, err }
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, b, ec), nil
}

func SettingCreatedEnvelope(payload port.SettingCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventSettingCreated, port.SettingCreatedEventVersion, port.SettingCreatedSchemaVersion, payload, ec)
}
func SettingUpdatedEnvelope(payload port.SettingUpdatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventSettingUpdated, port.SettingUpdatedEventVersion, port.SettingUpdatedSchemaVersion, payload, ec)
}
func SettingDeletedEnvelope(payload port.SettingDeletedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventSettingDeleted, port.SettingDeletedEventVersion, port.SettingDeletedSchemaVersion, payload, ec)
}
func FeatureFlagToggledEnvelope(payload port.FeatureFlagToggledPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventFeatureFlagToggled, port.FeatureFlagToggledEventVersion, port.FeatureFlagToggledSchemaVersion, payload, ec)
}
func FeatureFlagCreatedEnvelope(payload port.FeatureFlagCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventFeatureFlagCreated, port.FeatureFlagCreatedEventVersion, port.FeatureFlagCreatedSchemaVersion, payload, ec)
}
func FeatureFlagDeletedEnvelope(payload port.FeatureFlagDeletedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventFeatureFlagDeleted, port.FeatureFlagDeletedEventVersion, port.FeatureFlagDeletedSchemaVersion, payload, ec)
}
