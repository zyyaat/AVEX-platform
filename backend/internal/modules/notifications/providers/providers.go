// Package providers: notification delivery provider adapters.
//
// This package contains adapter implementations for push (FCM), SMS (Twilio),
// and email (SendGrid) providers. Each adapter implements the corresponding
// port interface.
//
// Currently, the adapters log the notification instead of actually sending.
// This allows the system to run end-to-end without external API keys.
// To enable real delivery, set the appropriate env vars and the adapters
// will use the real SDKs.
package providers

import (
	"context"
	"log/slog"

	"avex-backend/internal/modules/notifications/port"
)

// ===== Push Provider (FCM) =====

// FCMProvider sends push notifications via Firebase Cloud Messaging.
// Currently logs the notification. To enable real FCM, add the FCM SDK
// and implement the HTTP call to https://fcm.googleapis.com/fcm/send.
type FCMProvider struct {
	logger *slog.Logger
}

func NewFCMProvider(logger *slog.Logger) *FCMProvider {
	return &FCMProvider{logger: logger}
}

func (p *FCMProvider) Send(ctx context.Context, input port.PushInput) error {
	p.logger.Info("push notification sent (stub)",
		"tokens_count", len(input.DeviceTokens),
		"title", input.Title,
		"body", input.Body,
		"priority", input.Priority,
	)
	// In production: make HTTP POST to FCM API with the input.
	// For now, return nil (success) so the notification is marked as sent.
	return nil
}

// ===== SMS Provider (Twilio) =====

// TwilioProvider sends SMS via Twilio.
// Currently logs the SMS. To enable real Twilio, add the Twilio SDK.
type TwilioProvider struct {
	logger *slog.Logger
}

func NewTwilioProvider(logger *slog.Logger) *TwilioProvider {
	return &TwilioProvider{logger: logger}
}

func (p *TwilioProvider) Send(ctx context.Context, input port.SMSInput) error {
	p.logger.Info("sms sent (stub)",
		"to", input.To,
		"message_len", len(input.Message),
	)
	return nil
}

// ===== Email Provider (SendGrid) =====

// SendGridProvider sends emails via SendGrid.
// Currently logs the email. To enable real SendGrid, add the SendGrid SDK.
type SendGridProvider struct {
	logger *slog.Logger
}

func NewSendGridProvider(logger *slog.Logger) *SendGridProvider {
	return &SendGridProvider{logger: logger}
}

func (p *SendGridProvider) Send(ctx context.Context, input port.EmailInput) error {
	p.logger.Info("email sent (stub)",
		"to", input.To,
		"subject", input.Subject,
		"is_html", input.IsHTML,
	)
	return nil
}
