// Package jobs: bus subscriber that auto-creates audit entries from events.
//
// The subscriber listens to ALL events from ALL modules and creates audit
// entries for security-sensitive actions. Non-sensitive events (e.g. location
// updates) are skipped to avoid noise.
package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"avex-backend/internal/modules/audit/domain"
	"avex-backend/internal/modules/audit/port"
	"avex-backend/internal/platform/bus"
	"avex-backend/internal/platform/inbox"
)

type Subscriber struct {
	svc    port.ServicePort
	bus    bus.Subscriber
	inbox  inbox.Inbox
	logger *slog.Logger
}

func NewSubscriber(svc port.ServicePort, bus bus.Subscriber, inbox inbox.Inbox, logger *slog.Logger) *Subscriber {
	return &Subscriber{svc: svc, bus: bus, inbox: inbox, logger: logger}
}

// eventRule defines which events to audit and how to map them.
type eventRule struct {
	eventType  string
	handlerName string
	action     string
	resourceType string
	severity   domain.Severity
}

// eventsToAudit lists all events we want to record in the audit log.
// We skip high-volume, non-sensitive events (driver.location_updated, etc.).
var eventsToAudit = []eventRule{
	// Orders
	{"orders.order.created", "audit.on_order_created", domain.ActionOrderCreate, "order", domain.SeverityInfo},
	{"orders.order.cancelled", "audit.on_order_cancelled", domain.ActionOrderCancel, "order", domain.SeverityWarning},
	{"orders.order.delivered", "audit.on_order_delivered", domain.ActionOrderDeliver, "order", domain.SeverityInfo},
	// Financial
	{"financial.wallet.credited", "audit.on_wallet_credited", domain.ActionWalletCredit, "wallet", domain.SeverityCritical},
	{"financial.wallet.debited", "audit.on_wallet_debited", domain.ActionWalletDebit, "wallet", domain.SeverityCritical},
	{"financial.wallet.frozen", "audit.on_wallet_frozen", domain.ActionWalletFreeze, "wallet", domain.SeverityCritical},
	{"financial.promotion.redeemed", "audit.on_promo_redeemed", "financial.promotion.redeem", "promotion", domain.SeverityInfo},
	// Dispatch
	{"dispatch.driver.registered", "audit.on_driver_registered", domain.ActionDriverRegister, "driver", domain.SeverityInfo},
	{"dispatch.driver.suspended", "audit.on_driver_suspended", domain.ActionDriverSuspend, "driver", domain.SeverityWarning},
	{"dispatch.offer.created", "audit.on_dispatch_offer", domain.ActionDispatchOffer, "offer", domain.SeverityInfo},
	{"dispatch.offer.accepted", "audit.on_dispatch_accepted", domain.ActionDispatchAccept, "offer", domain.SeverityInfo},
	// Support
	{"support.ticket.created", "audit.on_ticket_created", domain.ActionTicketCreate, "ticket", domain.SeverityInfo},
	{"support.ticket.assigned", "audit.on_ticket_assigned", domain.ActionTicketAssign, "ticket", domain.SeverityInfo},
	{"support.ticket.closed", "audit.on_ticket_closed", domain.ActionTicketClose, "ticket", domain.SeverityInfo},
	// Permissions
	{"permissions.role.assigned", "audit.on_role_assigned", domain.ActionRoleAssign, "role", domain.SeverityCritical},
	{"permissions.role.unassigned", "audit.on_role_unassigned", domain.ActionRoleUnassign, "role", domain.SeverityCritical},
	{"permissions.permission.granted", "audit.on_perm_granted", domain.ActionPermissionGrant, "permission", domain.SeverityCritical},
	// Settings
	{"settings.setting.updated", "audit.on_setting_updated", domain.ActionSettingUpdate, "setting", domain.SeverityWarning},
	{"settings.setting.deleted", "audit.on_setting_deleted", "settings.setting.delete", "setting", domain.SeverityWarning},
	{"settings.setting.created", "audit.on_setting_created", "settings.setting.create", "setting", domain.SeverityInfo},
	{"settings.feature_flag.toggled", "audit.on_flag_toggled", domain.ActionFeatureFlagToggle, "feature_flag", domain.SeverityWarning},
	// Identity
	{"identity.user.registered", "audit.on_user_registered", domain.ActionUserRegister, "user", domain.SeverityInfo},
	{"identity.user.login_succeeded", "audit.on_user_login", domain.ActionUserLogin, "session", domain.SeverityInfo},
	{"identity.user.login_failed", "audit.on_user_login_failed", "identity.user.login_failed", "session", domain.SeverityWarning},
	{"identity.user.password_changed", "audit.on_password_changed", domain.ActionPasswordChange, "user", domain.SeverityCritical},
}

func (s *Subscriber) Start(ctx context.Context) error {
	for _, rule := range eventsToAudit {
		rule := rule // capture
		handler := inbox.Dedup(s.inbox, rule.handlerName, func(ctx context.Context, envelope bus.EventEnvelope) error {
			return s.handleEvent(ctx, envelope, rule)
		}, s.logger)
		if err := s.bus.Subscribe(ctx, rule.eventType, handler); err != nil {
			return err
		}
		s.logger.Info("audit subscriber registered", "event_type", rule.eventType, "action", rule.action)
	}
	s.logger.Info("audit subscriber started", "events_count", len(eventsToAudit))
	return nil
}

func (s *Subscriber) handleEvent(ctx context.Context, envelope bus.EventEnvelope, rule eventRule) error {
	// Extract common fields from the payload.
	var payload map[string]any
	_ = json.Unmarshal(envelope.Payload, &payload)

	// Determine actor + resource ID from common payload fields.
	actorType, actorID := extractActor(envelope, payload)
	resourceID := extractResourceID(payload)

	// Build description from the event type.
	description := rule.eventType

	// Log the audit entry.
	_, err := s.svc.Log(ctx, port.LogActionInput{
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       rule.action,
		ResourceType: rule.resourceType,
		ResourceID:   resourceID,
		Severity:     string(rule.severity),
		Description:  description,
		Metadata:     payload,
		IPAddress:    envelope.Actor.IP,
		UserAgent:    envelope.Actor.UserAgent,
		CorrelationID: envelope.CorrelationID,
		TraceID:      envelope.TraceID,
	})
	if err != nil {
		s.logger.Warn("audit log failed", "event_type", rule.eventType, "error", err)
	}
	return nil
}

// extractActor determines the actor type + ID from the event envelope + payload.
func extractActor(envelope bus.EventEnvelope, payload map[string]any) (string, string) {
	// Use envelope actor if available
	if envelope.Actor.Type != "" && envelope.Actor.ID != "" {
		return envelope.Actor.Type, envelope.Actor.ID
	}
	// Fall back to common payload fields
	if uid, ok := payload["user_id"].(string); ok && uid != "" {
		return "user", uid
	}
	if did, ok := payload["driver_id"].(string); ok && did != "" {
		return "driver", did
	}
	if mid, ok := payload["merchant_id"].(string); ok && mid != "" {
		return "merchant", mid
	}
	if oid, ok := payload["owner_id"].(string); ok && oid != "" {
		if ot, ok := payload["owner_type"].(string); ok && ot != "" {
			return ot, oid
		}
	}
	return "system", ""
}

// extractResourceID extracts the resource ID from common payload fields.
func extractResourceID(payload map[string]any) string {
	for _, key := range []string{"order_id", "wallet_id", "driver_id", "ticket_id", "role_id", "setting_id", "flag_id", "offer_id", "promotion_id", "permission_id"} {
		if id, ok := payload[key].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// suppress unused import
var _ = strings.TrimSpace
