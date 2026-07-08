// Package domain: Audit module errors + types.
//
// The Audit module records every sensitive operation in the system for
// compliance, security, and debugging purposes. Audit entries are IMMUTABLE
// (append-only) — they can never be modified or deleted.
//
// Each entry captures:
//   - WHO: actor type + ID (user/driver/merchant/agent/admin/system)
//   - WHAT: action + resource type + resource ID
//   - WHEN: timestamp
//   - WHERE: IP address + user agent
//   - WHY: reason (optional)
//   - CONTEXT: metadata (JSON, e.g. old_value/new_value)
//   - SEVERITY: info/warning/critical
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ===== Errors =====

var ErrAuditEntryNotFound = errors.New("audit entry not found")
var ErrCannotModifyAuditEntry = errors.New("audit entries are immutable and cannot be modified or deleted")

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyAction = errors.New("action is required")
var ErrEmptyResourceType = errors.New("resource type is required")
var ErrEmptyActorID = errors.New("actor id is required")
var ErrInvalidSeverity = errors.New("invalid severity")
var ErrInvalidActorType = errors.New("invalid actor type")

// ===== Severity =====

// Severity enumerates the importance level of an audit entry.
type Severity string

const (
	SeverityInfo     Severity = "info"     // normal operations (read, login)
	SeverityWarning  Severity = "warning"  // unusual but non-critical (failed login, rate limit)
	SeverityCritical Severity = "critical" // security-sensitive (role change, wallet debit, data export)
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	}
	return false
}

// ===== ActorType =====

// ActorType enumerates who performed the action.
type ActorType string

const (
	ActorUser     ActorType = "user"
	ActorDriver   ActorType = "driver"
	ActorMerchant ActorType = "merchant"
	ActorAgent    ActorType = "agent"
	ActorAdmin    ActorType = "admin"
	ActorSystem   ActorType = "system"
)

func (a ActorType) IsValid() bool {
	switch a {
	case ActorUser, ActorDriver, ActorMerchant, ActorAgent, ActorAdmin, ActorSystem:
		return true
	}
	return false
}

// ===== AuditEntry =====

// AuditEntry is a single immutable record of a sensitive operation.
type AuditEntry struct {
	id           string
	actorType    ActorType
	actorID      string
	action       string // e.g. "order.create", "wallet.credit", "role.assign"
	resourceType string // e.g. "order", "wallet", "role"
	resourceID   string // the ID of the affected resource
	severity     Severity
	description  string
	metadata     map[string]any // additional context (old_value, new_value, etc.)
	ipAddress    string
	userAgent    string
	correlationID string
	traceID      string
	createdAt    time.Time
}

// NewAuditEntry creates a new AuditEntry with validation.
func NewAuditEntry(
	id string,
	actorType ActorType,
	actorID, action, resourceType, resourceID string,
	severity Severity,
	description string,
	metadata map[string]any,
	ipAddress, userAgent, correlationID, traceID string,
	now time.Time,
) (AuditEntry, error) {
	if id == "" {
		return AuditEntry{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if !actorType.IsValid() {
		return AuditEntry{}, fmt.Errorf("%w: %s", ErrInvalidActorType, actorType)
	}
	if actorID == "" && actorType != ActorSystem {
		return AuditEntry{}, ErrEmptyActorID
	}
	if action == "" {
		return AuditEntry{}, ErrEmptyAction
	}
	if resourceType == "" {
		return AuditEntry{}, ErrEmptyResourceType
	}
	if !severity.IsValid() {
		return AuditEntry{}, fmt.Errorf("%w: %s", ErrInvalidSeverity, severity)
	}
	if severity == "" {
		severity = SeverityInfo
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return AuditEntry{
		id:           id,
		actorType:    actorType,
		actorID:      actorID,
		action:       action,
		resourceType: resourceType,
		resourceID:   resourceID,
		severity:     severity,
		description:  description,
		metadata:     metadata,
		ipAddress:    ipAddress,
		userAgent:    userAgent,
		correlationID: correlationID,
		traceID:      traceID,
		createdAt:    now,
	}, nil
}

// RehydrateAuditEntry reconstructs from persistence (no validation).
func RehydrateAuditEntry(
	id string,
	actorType ActorType,
	actorID, action, resourceType, resourceID string,
	severity Severity,
	description string,
	metadata map[string]any,
	ipAddress, userAgent, correlationID, traceID string,
	createdAt time.Time,
) AuditEntry {
	return AuditEntry{
		id: id, actorType: actorType, actorID: actorID, action: action,
		resourceType: resourceType, resourceID: resourceID, severity: severity,
		description: description, metadata: metadata, ipAddress: ipAddress,
		userAgent: userAgent, correlationID: correlationID, traceID: traceID,
		createdAt: createdAt,
	}
}

// ===== Accessors =====

func (e AuditEntry) ID() string            { return e.id }
func (e AuditEntry) ActorType() ActorType  { return e.actorType }
func (e AuditEntry) ActorID() string       { return e.actorID }
func (e AuditEntry) Action() string        { return e.action }
func (e AuditEntry) ResourceType() string  { return e.resourceType }
func (e AuditEntry) ResourceID() string    { return e.resourceID }
func (e AuditEntry) Severity() Severity    { return e.severity }
func (e AuditEntry) Description() string   { return e.description }
func (e AuditEntry) Metadata() map[string]any { return e.metadata }
func (e AuditEntry) IPAddress() string     { return e.ipAddress }
func (e AuditEntry) UserAgent() string     { return e.userAgent }
func (e AuditEntry) CorrelationID() string { return e.correlationID }
func (e AuditEntry) TraceID() string       { return e.traceID }
func (e AuditEntry) CreatedAt() time.Time  { return e.createdAt }

// MetadataJSON returns the metadata as a JSON byte slice (for persistence).
func (e AuditEntry) MetadataJSON() json.RawMessage {
	if len(e.metadata) == 0 {
		return nil
	}
	b, _ := json.Marshal(e.metadata)
	return b
}

// IsCritical reports whether the entry has critical severity.
func (e AuditEntry) IsCritical() bool { return e.severity == SeverityCritical }

// ===== Action constants (convention: "module.resource.action") =====
//
// These are NOT exhaustive — any string is valid. These are just common
// ones used across modules for consistency.
const (
	ActionOrderCreate      = "orders.order.create"
	ActionOrderCancel      = "orders.order.cancel"
	ActionOrderDeliver     = "orders.order.deliver"
	ActionWalletCredit     = "financial.wallet.credit"
	ActionWalletDebit      = "financial.wallet.debit"
	ActionWalletFreeze     = "financial.wallet.freeze"
	ActionDriverRegister   = "dispatch.driver.register"
	ActionDriverSuspend    = "dispatch.driver.suspend"
	ActionDispatchOffer    = "dispatch.offer.create"
	ActionDispatchAccept   = "dispatch.offer.accept"
	ActionTicketCreate     = "support.ticket.create"
	ActionTicketAssign     = "support.ticket.assign"
	ActionTicketClose      = "support.ticket.close"
	ActionRoleAssign       = "permissions.role.assign"
	ActionRoleUnassign     = "permissions.role.unassign"
	ActionPermissionGrant  = "permissions.permission.grant"
	ActionSettingUpdate    = "settings.setting.update"
	ActionSettingRollback  = "settings.setting.rollback"
	ActionFeatureFlagToggle = "settings.feature_flag.toggle"
	ActionUserLogin        = "identity.user.login"
	ActionUserLogout       = "identity.user.logout"
	ActionUserRegister     = "identity.user.register"
	ActionPasswordChange   = "identity.user.password_change"
)
