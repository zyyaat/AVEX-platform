// Package domain notification: Notification aggregate root.
//
// A Notification is a message to be delivered to a recipient (user, driver,
// or merchant) via one or more channels (push, sms, email).
//
// Lifecycle:
//   pending → sending → sent → delivered
//                    ↘ failed → (retry) → sending → ...
//   pending → cancelled (e.g. user unsubscribed before send)
//
// Invariants:
//   - Title and Body are required.
//   - RecipientID is required.
//   - At least one channel must be enabled.
//   - retry_count cannot exceed max_retries (default 3).
//
// Imports stdlib only.
package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Notification is the aggregate root for a notification message.
type Notification struct {
	id            string
	recipientType string // user | driver | merchant
	recipientID   string
	notifType     NotificationType
	channel       Channel
	title         string
	titleAr       string
	body          string
	bodyAr        string
	data          map[string]any // additional payload data (e.g. order_id)
	status        Status
	priority      string // normal | high
	retryCount    int
	maxRetries    int
	lastError     string
	scheduledAt   time.Time
	sentAt        *time.Time
	deliveredAt   *time.Time
	createdAt     time.Time
	updatedAt     time.Time
}

// NewNotification creates a new Notification with validation.
// New notifications start in "pending" status.
func NewNotification(
	id, recipientType, recipientID string,
	notifType NotificationType,
	channel Channel,
	title, titleAr, body, bodyAr string,
	data map[string]any,
	priority string,
	maxRetries int,
	now time.Time,
) (Notification, error) {
	if id == "" {
		return Notification{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if recipientID == "" {
		return Notification{}, ErrEmptyRecipientID
	}
	if recipientType != "user" && recipientType != "driver" && recipientType != "merchant" {
		return Notification{}, fmt.Errorf("%w: recipient type %q", ErrInvalidInput, recipientType)
	}
	if !notifType.IsValid() {
		return Notification{}, fmt.Errorf("%w: %s", ErrInvalidNotificationType, notifType)
	}
	if !channel.IsValid() {
		return Notification{}, fmt.Errorf("%w: %s", ErrInvalidChannel, channel)
	}
	if title == "" {
		return Notification{}, ErrEmptyTitle
	}
	if body == "" {
		return Notification{}, ErrEmptyBody
	}
	if priority == "" {
		priority = "normal"
	}
	if priority != "normal" && priority != "high" {
		return Notification{}, fmt.Errorf("%w: priority %q", ErrInvalidInput, priority)
	}
	if maxRetries < 0 {
		maxRetries = 3
	}
	if maxRetries == 0 {
		maxRetries = 3
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Notification{
		id:            id,
		recipientType: recipientType,
		recipientID:   recipientID,
		notifType:     notifType,
		channel:       channel,
		title:         title,
		titleAr:       titleAr,
		body:          body,
		bodyAr:        bodyAr,
		data:          data,
		status:        StatusPending,
		priority:      priority,
		retryCount:    0,
		maxRetries:    maxRetries,
		scheduledAt:   now,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// RehydrateNotification reconstructs a Notification from persistence.
func RehydrateNotification(
	id, recipientType, recipientID string,
	notifType NotificationType,
	channel Channel,
	title, titleAr, body, bodyAr string,
	data map[string]any,
	status Status,
	priority string,
	retryCount, maxRetries int,
	lastError string,
	scheduledAt time.Time,
	sentAt, deliveredAt *time.Time,
	createdAt, updatedAt time.Time,
) Notification {
	return Notification{
		id:            id,
		recipientType: recipientType,
		recipientID:   recipientID,
		notifType:     notifType,
		channel:       channel,
		title:         title,
		titleAr:       titleAr,
		body:          body,
		bodyAr:        bodyAr,
		data:          data,
		status:        status,
		priority:      priority,
		retryCount:    retryCount,
		maxRetries:    maxRetries,
		lastError:     lastError,
		scheduledAt:   scheduledAt,
		sentAt:        sentAt,
		deliveredAt:   deliveredAt,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

// ===== Accessors =====

func (n Notification) ID() string                { return n.id }
func (n Notification) RecipientType() string     { return n.recipientType }
func (n Notification) RecipientID() string       { return n.recipientID }
func (n Notification) Type() NotificationType    { return n.notifType }
func (n Notification) Channel() Channel          { return n.channel }
func (n Notification) Title() string             { return n.title }
func (n Notification) TitleAr() string           { return n.titleAr }
func (n Notification) Body() string              { return n.body }
func (n Notification) BodyAr() string            { return n.bodyAr }
func (n Notification) Data() map[string]any      { return n.data }
func (n Notification) Status() Status            { return n.status }
func (n Notification) Priority() string          { return n.priority }
func (n Notification) RetryCount() int           { return n.retryCount }
func (n Notification) MaxRetries() int           { return n.maxRetries }
func (n Notification) LastError() string         { return n.lastError }
func (n Notification) ScheduledAt() time.Time    { return n.scheduledAt }
func (n Notification) SentAt() *time.Time        { return n.sentAt }
func (n Notification) DeliveredAt() *time.Time   { return n.deliveredAt }
func (n Notification) CreatedAt() time.Time      { return n.createdAt }
func (n Notification) UpdatedAt() time.Time      { return n.updatedAt }

// IsPending reports whether the notification is awaiting send.
func (n Notification) IsPending() bool { return n.status == StatusPending }

// IsTerminal reports whether the notification is in a terminal state.
func (n Notification) IsTerminal() bool { return n.status.IsTerminal() }

// CanRetry reports whether the notification can be retried.
func (n Notification) CanRetry() bool {
	return n.status == StatusFailed && n.retryCount < n.maxRetries
}

// DataJSON returns the data field as a JSON byte slice (for the outbox).
func (n Notification) DataJSON() json.RawMessage {
	if len(n.data) == 0 {
		return nil
	}
	b, _ := json.Marshal(n.data)
	return b
}

// ===== Status Transitions =====

// MarkSending transitions the notification from pending → sending.
func (n Notification) MarkSending(now time.Time) (Notification, error) {
	if n.status != StatusPending && n.status != StatusFailed {
		if n.status == StatusSending {
			return n, nil // idempotent
		}
		return n, fmt.Errorf("%w: cannot send from %s", ErrInvalidNotificationStatus, n.status)
	}
	n.status = StatusSending
	n.updatedAt = now
	return n, nil
}

// MarkSent transitions the notification from sending → sent.
func (n Notification) MarkSent(now time.Time) (Notification, error) {
	if n.status != StatusSending {
		if n.status == StatusSent || n.status == StatusDelivered {
			return n, nil // idempotent
		}
		return n, fmt.Errorf("%w: cannot mark sent from %s", ErrInvalidNotificationStatus, n.status)
	}
	n.status = StatusSent
	n.sentAt = &now
	n.updatedAt = now
	return n, nil
}

// MarkDelivered transitions the notification from sent → delivered.
func (n Notification) MarkDelivered(now time.Time) (Notification, error) {
	if n.status != StatusSent {
		if n.status == StatusDelivered {
			return n, nil // idempotent
		}
		return n, fmt.Errorf("%w: cannot mark delivered from %s", ErrInvalidNotificationStatus, n.status)
	}
	n.status = StatusDelivered
	n.deliveredAt = &now
	n.updatedAt = now
	return n, nil
}

// MarkFailed transitions the notification to failed.
// If retry_count < max_retries, the notification can be retried (status → pending).
func (n Notification) MarkFailed(errMsg string, now time.Time) (Notification, error) {
	if n.status.IsTerminal() {
		return n, fmt.Errorf("%w: cannot fail from %s", ErrNotificationCannotRetry, n.status)
	}
	n.status = StatusFailed
	n.lastError = errMsg
	n.retryCount++
	n.updatedAt = now
	return n, nil
}

// Retry transitions the notification from failed → pending.
// Returns an error if max retries has been reached.
func (n Notification) Retry(now time.Time) (Notification, error) {
	if n.status != StatusFailed {
		return n, fmt.Errorf("%w: can only retry failed notifications", ErrNotificationCannotRetry)
	}
	if n.retryCount >= n.maxRetries {
		return n, ErrMaxRetriesReached
	}
	n.status = StatusPending
	n.lastError = ""
	n.scheduledAt = now
	n.updatedAt = now
	return n, nil
}

// Cancel transitions the notification to cancelled.
func (n Notification) Cancel(now time.Time) (Notification, error) {
	if n.status.IsTerminal() {
		return n, fmt.Errorf("%w: cannot cancel from %s", ErrInvalidNotificationStatus, n.status)
	}
	n.status = StatusCancelled
	n.updatedAt = now
	return n, nil
}
