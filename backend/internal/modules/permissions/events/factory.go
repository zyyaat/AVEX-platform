// Package events factory: convenience constructors.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/permissions/port"
)

func BuildEnvelope(eventType string, eventVersion, schemaVersion int, payload any, ec port.EventContext) (port.EventEnvelope, error) {
	b, err := json.Marshal(payload)
	if err != nil { return port.EventEnvelope{}, err }
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, b, ec), nil
}

func RoleCreatedEnvelope(payload port.RoleCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventRoleCreated, port.RoleCreatedEventVersion, port.RoleCreatedSchemaVersion, payload, ec)
}
func RoleDeletedEnvelope(payload port.RoleDeletedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventRoleDeleted, port.RoleDeletedEventVersion, port.RoleDeletedSchemaVersion, payload, ec)
}
func PermissionGrantedEnvelope(payload port.PermissionGrantedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventPermissionGranted, port.PermissionGrantedEventVersion, port.PermissionGrantedSchemaVersion, payload, ec)
}
func PermissionRevokedEnvelope(payload port.PermissionRevokedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventPermissionRevoked, port.PermissionRevokedEventVersion, port.PermissionRevokedSchemaVersion, payload, ec)
}
func RoleAssignedEnvelope(payload port.RoleAssignedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventRoleAssigned, port.RoleAssignedEventVersion, port.RoleAssignedSchemaVersion, payload, ec)
}
func RoleUnassignedEnvelope(payload port.RoleUnassignedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventRoleUnassigned, port.RoleUnassignedEventVersion, port.RoleUnassignedSchemaVersion, payload, ec)
}
