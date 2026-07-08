// Package domain tests: Notification + Preferences.
package domain

import (
	"testing"
	"time"
)

func testNowNotif() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

// ===== Notification Tests =====

func TestNewNotification(t *testing.T) {
	now := testNowNotif()
	tests := []struct {
		name          string
		id            string
		recipientType string
		recipientID   string
		notifType     NotificationType
		channel       Channel
		title         string
		body          string
		priority      string
		wantErr       error
	}{
		{"valid", "n1", "user", "u1", TypeOrderCreated, ChannelPush, "Order Confirmed", "Your order is confirmed", "normal", nil},
		{"valid high priority", "n2", "driver", "d1", TypeDispatchOffer, ChannelPush, "New Offer", "You have a new offer", "high", nil},
		{"empty id", "", "user", "u1", TypeOrderCreated, ChannelPush, "T", "B", "normal", ErrInvalidID},
		{"empty recipient", "n3", "user", "", TypeOrderCreated, ChannelPush, "T", "B", "normal", ErrEmptyRecipientID},
		{"invalid recipient type", "n4", "admin", "a1", TypeOrderCreated, ChannelPush, "T", "B", "normal", ErrInvalidInput},
		{"invalid type", "n5", "user", "u1", NotificationType("bogus"), ChannelPush, "T", "B", "normal", ErrInvalidNotificationType},
		{"invalid channel", "n6", "user", "u1", TypeOrderCreated, Channel("bogus"), "T", "B", "normal", ErrInvalidChannel},
		{"empty title", "n7", "user", "u1", TypeOrderCreated, ChannelPush, "", "B", "normal", ErrEmptyTitle},
		{"empty body", "n8", "user", "u1", TypeOrderCreated, ChannelPush, "T", "", "normal", ErrEmptyBody},
		{"invalid priority", "n9", "user", "u1", TypeOrderCreated, ChannelPush, "T", "B", "urgent", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNotification(tt.id, tt.recipientType, tt.recipientID, tt.notifType, tt.channel, tt.title, "", tt.body, "", nil, tt.priority, 3, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNotificationStatusTransitions(t *testing.T) {
	now := testNowNotif()
	n, _ := NewNotification("n1", "user", "u1", TypeOrderCreated, ChannelPush, "T", "", "B", "", nil, "normal", 3, now)

	// pending → sending
	n, err := n.MarkSending(now)
	if err != nil {
		t.Fatalf("mark sending: %v", err)
	}
	if n.Status() != StatusSending {
		t.Errorf("expected sending, got %s", n.Status())
	}

	// sending → sent
	n, err = n.MarkSent(now)
	if err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	if n.Status() != StatusSent {
		t.Errorf("expected sent, got %s", n.Status())
	}
	if n.SentAt() == nil {
		t.Errorf("expected sent_at set")
	}

	// sent → delivered
	n, err = n.MarkDelivered(now)
	if err != nil {
		t.Fatalf("mark delivered: %v", err)
	}
	if n.Status() != StatusDelivered {
		t.Errorf("expected delivered, got %s", n.Status())
	}
	if n.DeliveredAt() == nil {
		t.Errorf("expected delivered_at set")
	}
}

func TestNotificationFailAndRetry(t *testing.T) {
	now := testNowNotif()
	n, _ := NewNotification("n1", "user", "u1", TypeOrderCreated, ChannelPush, "T", "", "B", "", nil, "normal", 2, now)

	// pending → sending → failed
	n, _ = n.MarkSending(now)
	n, err := n.MarkFailed("FCM timeout", now)
	if err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if n.Status() != StatusFailed {
		t.Errorf("expected failed, got %s", n.Status())
	}
	if n.RetryCount() != 1 {
		t.Errorf("expected retry_count 1, got %d", n.RetryCount())
	}
	if n.LastError() != "FCM timeout" {
		t.Errorf("expected error msg, got %q", n.LastError())
	}

	// retry → pending
	n, err = n.Retry(now)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if n.Status() != StatusPending {
		t.Errorf("expected pending after retry, got %s", n.Status())
	}

	// Fail again (retry_count = 2)
	n, _ = n.MarkSending(now)
	n, _ = n.MarkFailed("still failing", now)
	if n.RetryCount() != 2 {
		t.Errorf("expected retry_count 2, got %d", n.RetryCount())
	}

	// Max retries reached — cannot retry
	_, err = n.Retry(now)
	if !errIs(err, ErrMaxRetriesReached) {
		t.Fatalf("expected ErrMaxRetriesReached, got %v", err)
	}
}

func TestNotificationCancel(t *testing.T) {
	now := testNowNotif()
	n, _ := NewNotification("n1", "user", "u1", TypeOrderCreated, ChannelPush, "T", "", "B", "", nil, "normal", 3, now)

	n, err := n.Cancel(now)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if n.Status() != StatusCancelled {
		t.Errorf("expected cancelled, got %s", n.Status())
	}

	// Cannot cancel a terminal state
	_, err = n.Cancel(now)
	if !errIs(err, ErrInvalidNotificationStatus) {
		t.Fatalf("expected ErrInvalidNotificationStatus, got %v", err)
	}
}

// ===== Preferences Tests =====

func TestNewUserNotificationPreferences(t *testing.T) {
	now := testNowNotif()
	tests := []struct {
		name          string
		id            string
		recipientType string
		recipientID   string
		wantErr       error
	}{
		{"valid user", "p1", "user", "u1", nil},
		{"valid driver", "p2", "driver", "d1", nil},
		{"valid merchant", "p3", "merchant", "m1", nil},
		{"empty id", "", "user", "u1", ErrInvalidID},
		{"empty recipient", "p4", "user", "", ErrEmptyRecipientID},
		{"invalid type", "p5", "admin", "a1", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUserNotificationPreferences(tt.id, tt.recipientType, tt.recipientID, "+201234567890", "test@example.com", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPreferencesDefaults(t *testing.T) {
	now := testNowNotif()
	prefs, _ := NewUserNotificationPreferences("p1", "user", "u1", "+201234567890", "test@example.com", now)

	// Orders: push + sms should be enabled by default
	if !prefs.IsEnabled(CategoryOrders, ChannelPush) {
		t.Error("expected orders:push enabled by default")
	}
	if !prefs.IsEnabled(CategoryOrders, ChannelSMS) {
		t.Error("expected orders:sms enabled by default")
	}

	// Wallet: push + email enabled, sms disabled
	if !prefs.IsEnabled(CategoryWallet, ChannelPush) {
		t.Error("expected wallet:push enabled")
	}
	if !prefs.IsEnabled(CategoryWallet, ChannelEmail) {
		t.Error("expected wallet:email enabled")
	}
	if prefs.IsEnabled(CategoryWallet, ChannelSMS) {
		t.Error("expected wallet:sms disabled")
	}

	// Marketing: disabled by default
	if prefs.IsEnabled(CategoryMarketing, ChannelPush) {
		t.Error("expected marketing:push disabled by default")
	}
	if prefs.IsEnabled(CategoryMarketing, ChannelEmail) {
		t.Error("expected marketing:email disabled by default")
	}
}

func TestPreferencesSetEnabled(t *testing.T) {
	now := testNowNotif()
	prefs, _ := NewUserNotificationPreferences("p1", "user", "u1", "+201234567890", "test@example.com", now)

	// Disable orders:sms
	prefs = prefs.SetEnabled(CategoryOrders, ChannelSMS, false)
	if prefs.IsEnabled(CategoryOrders, ChannelSMS) {
		t.Error("expected orders:sms disabled after SetEnabled(false)")
	}

	// Enable marketing:push
	prefs = prefs.SetEnabled(CategoryMarketing, ChannelPush, true)
	if !prefs.IsEnabled(CategoryMarketing, ChannelPush) {
		t.Error("expected marketing:push enabled after SetEnabled(true)")
	}
}

func TestPreferencesCanSend(t *testing.T) {
	now := testNowNotif()

	// User with phone + email but no device tokens
	prefs, _ := NewUserNotificationPreferences("p1", "user", "u1", "+201234567890", "test@example.com", now)

	// Cannot send push (no device tokens)
	if prefs.CanSendPush(TypeOrderCreated) {
		t.Error("expected CanSendPush=false (no device tokens)")
	}

	// Can send SMS
	if !prefs.CanSendSMS(TypeOrderCreated) {
		t.Error("expected CanSendSMS=true")
	}

	// Can send email for wallet
	if !prefs.CanSendEmail(TypeWalletCredited) {
		t.Error("expected CanSendEmail=true for wallet")
	}

	// Add device token
	prefs = prefs.AddDeviceToken("token-123")
	if !prefs.CanSendPush(TypeOrderCreated) {
		t.Error("expected CanSendPush=true after adding device token")
	}

	// Remove device token
	prefs = prefs.RemoveDeviceToken("token-123")
	if prefs.CanSendPush(TypeOrderCreated) {
		t.Error("expected CanSendPush=false after removing device token")
	}
}

func TestPreferencesMarketingDisabledForUser(t *testing.T) {
	// Marketing is disabled by default — verify marketing notifications can't be sent.
	now := testNowNotif()
	prefs, _ := NewUserNotificationPreferences("p1", "user", "u1", "+201234567890", "test@example.com", now)
	prefs = prefs.AddDeviceToken("token-1")

	if prefs.CanSendPush(TypeMarketing) {
		t.Error("expected CanSendPush=false for marketing (disabled by default)")
	}
}

func TestPreferencesDriverDispatchEnabled(t *testing.T) {
	now := testNowNotif()
	prefs, _ := NewUserNotificationPreferences("p1", "driver", "d1", "+201234567890", "driver@example.com", now)
	prefs = prefs.AddDeviceToken("token-driver")

	// Drivers get dispatch:push enabled by default
	if !prefs.CanSendPush(TypeDispatchOffer) {
		t.Error("expected driver CanSendPush=true for dispatch offers")
	}
}

// errIs helper
func errIs(err, target error) bool {
	if err == target {
		return true
	}
	for {
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == target {
			return true
		}
		if err == nil {
			return false
		}
	}
}
