// Package domain contains pure domain entities for the notifications module.
//
// This file: typed domain errors + notification types + channels + status.
//
// Imports stdlib only.
package domain

import (
	"errors"
	"fmt"
)

// ===== Notification Errors =====

var ErrNotificationNotFound = errors.New("notification not found")
var ErrNotificationAlreadyExists = errors.New("notification already exists")
var ErrNotificationAlreadySent = errors.New("notification already sent")
var ErrNotificationAlreadyFailed = errors.New("notification already failed")
var ErrNotificationCannotRetry = errors.New("notification cannot be retried")
var ErrInvalidNotificationStatus = errors.New("invalid notification status")
var ErrInvalidChannel = errors.New("invalid channel")
var ErrInvalidNotificationType = errors.New("invalid notification type")
var ErrNoChannelEnabled = errors.New("no notification channel enabled for this user")
var ErrMaxRetriesReached = errors.New("max retries reached")

// ===== Preference Errors =====

var ErrPreferenceNotFound = errors.New("preference not found")
var ErrPreferenceAlreadyExists = errors.New("preference already exists")

// ===== Device Errors =====

var ErrDeviceNotFound = errors.New("device not found")
var ErrDeviceAlreadyExists = errors.New("device already exists")
var ErrInvalidPlatform = errors.New("invalid platform")

// ===== Validation Errors =====

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyRecipientID = errors.New("recipient id is required")
var ErrEmptyTitle = errors.New("title is required")
var ErrEmptyBody = errors.New("body is required")

// Channel enumerates notification delivery channels.
type Channel string

const (
	ChannelPush  Channel = "push"
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
)

// IsValid reports whether the channel is recognized.
func (c Channel) IsValid() bool {
	switch c {
	case ChannelPush, ChannelSMS, ChannelEmail:
		return true
	}
	return false
}

// NotificationType enumerates the kinds of notifications.
type NotificationType string

const (
	TypeOrderCreated       NotificationType = "order_created"
	TypeOrderConfirmed     NotificationType = "order_confirmed"
	TypeOrderPreparing     NotificationType = "order_preparing"
	TypeOrderReady         NotificationType = "order_ready"
	TypeOrderAssigned      NotificationType = "order_assigned"
	TypeOrderPickedUp      NotificationType = "order_picked_up"
	TypeOrderDelivered     NotificationType = "order_delivered"
	TypeOrderCancelled     NotificationType = "order_cancelled"
	TypeDispatchOffer      NotificationType = "dispatch_offer"
	TypeDispatchAssigned   NotificationType = "dispatch_assigned"
	TypeWalletCredited     NotificationType = "wallet_credited"
	TypeWalletDebited      NotificationType = "wallet_debited"
	TypePromotionRedeemed  NotificationType = "promotion_redeemed"
	TypeSystemAnnouncement NotificationType = "system_announcement"
	TypeMarketing          NotificationType = "marketing"
)

// IsValid reports whether the notification type is recognized.
func (t NotificationType) IsValid() bool {
	switch t {
	case TypeOrderCreated, TypeOrderConfirmed, TypeOrderPreparing, TypeOrderReady,
		TypeOrderAssigned, TypeOrderPickedUp, TypeOrderDelivered, TypeOrderCancelled,
		TypeDispatchOffer, TypeDispatchAssigned,
		TypeWalletCredited, TypeWalletDebited, TypePromotionRedeemed,
		TypeSystemAnnouncement, TypeMarketing:
		return true
	}
	return false
}

// Category groups notification types for preference management.
type Category string

const (
	CategoryOrders       Category = "orders"
	CategoryDispatch     Category = "dispatch"
	CategoryWallet       Category = "wallet"
	CategoryPromotions   Category = "promotions"
	CategorySystem       Category = "system"
	CategoryMarketing    Category = "marketing"
)

// CategoryFor returns the category for a notification type.
func CategoryFor(t NotificationType) Category {
	switch t {
	case TypeOrderCreated, TypeOrderConfirmed, TypeOrderPreparing, TypeOrderReady,
		TypeOrderAssigned, TypeOrderPickedUp, TypeOrderDelivered, TypeOrderCancelled:
		return CategoryOrders
	case TypeDispatchOffer, TypeDispatchAssigned:
		return CategoryDispatch
	case TypeWalletCredited, TypeWalletDebited:
		return CategoryWallet
	case TypePromotionRedeemed:
		return CategoryPromotions
	case TypeSystemAnnouncement:
		return CategorySystem
	case TypeMarketing:
		return CategoryMarketing
	}
	return CategorySystem
}

// Status enumerates notification lifecycle states.
type Status string

const (
	StatusPending   Status = "pending"
	StatusSending   Status = "sending"
	StatusSent      Status = "sent"
	StatusDelivered Status = "delivered"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// IsValid reports whether the status is recognized.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusSending, StatusSent, StatusDelivered, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

// IsTerminal reports whether the status is terminal.
func (s Status) IsTerminal() bool {
	return s == StatusDelivered || s == StatusFailed || s == StatusCancelled
}

// ===== Composite Error =====

type ValidationError struct {
	Field   string
	Wrapped error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %v", e.Field, e.Wrapped)
	}
	return e.Wrapped.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Wrapped
}

func NewValidationError(field string, err error) *ValidationError {
	return &ValidationError{Field: field, Wrapped: err}
}
