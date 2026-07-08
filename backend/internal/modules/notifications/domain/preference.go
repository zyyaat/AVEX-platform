// Package domain preference: UserNotificationPreferences value object.
//
// Each user/driver/merchant has a set of preferences that control which
// notification types they receive via which channels.
//
// Default preferences:
//   - Orders notifications: push + sms (high priority)
//   - Dispatch notifications: push only (drivers)
//   - Wallet notifications: push + email
//   - Promotions: push only
//   - System: push + sms
//   - Marketing: disabled by default (opt-in)
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// UserNotificationPreferences tracks per-recipient notification preferences.
type UserNotificationPreferences struct {
	id           string
	recipientType string // user | driver | merchant
	recipientID  string
	// Per-category, per-channel enable flags.
	// Key format: "orders:push", "dispatch:sms", etc.
	prefs        map[string]bool
	phoneNumber  string // for SMS
	email        string // for email
	deviceTokens []string // FCM/APNs tokens (for push)
	updatedAt    time.Time
	createdAt    time.Time
}

// preferenceKey constructs the map key for a (category, channel) pair.
func preferenceKey(cat Category, ch Channel) string {
	return string(cat) + ":" + string(ch)
}

// NewUserNotificationPreferences creates a new preferences object with defaults.
func NewUserNotificationPreferences(
	id, recipientType, recipientID string,
	phoneNumber, email string,
	now time.Time,
) (UserNotificationPreferences, error) {
	if id == "" {
		return UserNotificationPreferences{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if recipientID == "" {
		return UserNotificationPreferences{}, ErrEmptyRecipientID
	}
	if recipientType != "user" && recipientType != "driver" && recipientType != "merchant" {
		return UserNotificationPreferences{}, fmt.Errorf("%w: recipient type %q", ErrInvalidInput, recipientType)
	}

	prefs := make(map[string]bool)
	// Defaults: enable orders + system + dispatch on push+sms, wallet on push+email,
	// promotions on push, marketing disabled.
	for _, ch := range []Channel{ChannelPush, ChannelSMS} {
		prefs[preferenceKey(CategoryOrders, ch)] = true
		prefs[preferenceKey(CategorySystem, ch)] = true
		if recipientType == "driver" {
			prefs[preferenceKey(CategoryDispatch, ch)] = true
		}
	}
	prefs[preferenceKey(CategoryWallet, ChannelPush)] = true
	prefs[preferenceKey(CategoryWallet, ChannelEmail)] = true
	prefs[preferenceKey(CategoryPromotions, ChannelPush)] = true
	// Marketing: disabled by default
	prefs[preferenceKey(CategoryMarketing, ChannelPush)] = false
	prefs[preferenceKey(CategoryMarketing, ChannelEmail)] = false

	return UserNotificationPreferences{
		id:            id,
		recipientType: recipientType,
		recipientID:   recipientID,
		prefs:         prefs,
		phoneNumber:   phoneNumber,
		email:         email,
		updatedAt:     now,
		createdAt:     now,
	}, nil
}

// RehydrateUserNotificationPreferences reconstructs from persistence.
func RehydrateUserNotificationPreferences(
	id, recipientType, recipientID string,
	prefs map[string]bool,
	phoneNumber, email string,
	deviceTokens []string,
	createdAt, updatedAt time.Time,
) UserNotificationPreferences {
	if prefs == nil {
		prefs = make(map[string]bool)
	}
	return UserNotificationPreferences{
		id:            id,
		recipientType: recipientType,
		recipientID:   recipientID,
		prefs:         prefs,
		phoneNumber:   phoneNumber,
		email:         email,
		deviceTokens:  deviceTokens,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

// ===== Accessors =====

func (p UserNotificationPreferences) ID() string            { return p.id }
func (p UserNotificationPreferences) RecipientType() string { return p.recipientType }
func (p UserNotificationPreferences) RecipientID() string   { return p.recipientID }
func (p UserNotificationPreferences) PhoneNumber() string   { return p.phoneNumber }
func (p UserNotificationPreferences) Email() string         { return p.email }
func (p UserNotificationPreferences) DeviceTokens() []string { return p.deviceTokens }
func (p UserNotificationPreferences) CreatedAt() time.Time  { return p.createdAt }
func (p UserNotificationPreferences) UpdatedAt() time.Time  { return p.updatedAt }

// IsEnabled reports whether the given (category, channel) is enabled.
// Returns false for unknown keys.
func (p UserNotificationPreferences) IsEnabled(cat Category, ch Channel) bool {
	return p.prefs[preferenceKey(cat, ch)]
}

// SetEnabled enables/disables a (category, channel) pair.
func (p UserNotificationPreferences) SetEnabled(cat Category, ch Channel, enabled bool) UserNotificationPreferences {
	p.prefs[preferenceKey(cat, ch)] = enabled
	p.updatedAt = time.Now().UTC()
	return p
}

// IsChannelEnabledForType reports whether the channel is enabled for a notification type.
// This combines the category lookup + channel check.
func (p UserNotificationPreferences) IsChannelEnabledForType(t NotificationType, ch Channel) bool {
	return p.IsEnabled(CategoryFor(t), ch)
}

// CanSendPush reports whether the user can receive push notifications
// (i.e. has at least one device token + push enabled for the type).
func (p UserNotificationPreferences) CanSendPush(t NotificationType) bool {
	if len(p.deviceTokens) == 0 {
		return false
	}
	return p.IsChannelEnabledForType(t, ChannelPush)
}

// CanSendSMS reports whether the user can receive SMS.
func (p UserNotificationPreferences) CanSendSMS(t NotificationType) bool {
	if p.phoneNumber == "" {
		return false
	}
	return p.IsChannelEnabledForType(t, ChannelSMS)
}

// CanSendEmail reports whether the user can receive email.
func (p UserNotificationPreferences) CanSendEmail(t NotificationType) bool {
	if p.email == "" {
		return false
	}
	return p.IsChannelEnabledForType(t, ChannelEmail)
}

// AddDeviceToken adds a push device token.
func (p UserNotificationPreferences) AddDeviceToken(token string) UserNotificationPreferences {
	for _, t := range p.deviceTokens {
		if t == token {
			return p // already exists
		}
	}
	p.deviceTokens = append(p.deviceTokens, token)
	p.updatedAt = time.Now().UTC()
	return p
}

// RemoveDeviceToken removes a push device token.
func (p UserNotificationPreferences) RemoveDeviceToken(token string) UserNotificationPreferences {
	for i, t := range p.deviceTokens {
		if t == token {
			p.deviceTokens = append(p.deviceTokens[:i], p.deviceTokens[i+1:]...)
			p.updatedAt = time.Now().UTC()
			break
		}
	}
	return p
}

// SetPhoneNumber updates the phone number.
func (p UserNotificationPreferences) SetPhoneNumber(phone string) UserNotificationPreferences {
	p.phoneNumber = phone
	p.updatedAt = time.Now().UTC()
	return p
}

// SetEmail updates the email.
func (p UserNotificationPreferences) SetEmail(email string) UserNotificationPreferences {
	p.email = email
	p.updatedAt = time.Now().UTC()
	return p
}

// AllPrefs returns a copy of the preferences map (for persistence).
func (p UserNotificationPreferences) AllPrefs() map[string]bool {
	cp := make(map[string]bool, len(p.prefs))
	for k, v := range p.prefs {
		cp[k] = v
	}
	return cp
}
