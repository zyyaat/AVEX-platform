// Package service: notifications service implementation.
package service

import (
	"context"
	"fmt"

	"avex-backend/internal/modules/notifications/domain"
	"avex-backend/internal/modules/notifications/events"
	"avex-backend/internal/modules/notifications/port"
)

type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service {
	return &Service{deps: deps, pool: pool}
}

func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
	return port.EventContext{
		Actor: actor,
		Metadata: port.EventMetadata{
			OccurredAt: s.deps.Clock.Now(),
		},
	}
}

// ===== SendNotification =====
//
// Creates + sends a notification via all channels the recipient has enabled.
// For each enabled channel, a separate Notification record is created.
// The actual send happens synchronously (could be moved to a background worker).

func (s *Service) SendNotification(ctx context.Context, input port.SendNotificationInput) ([]port.NotificationDTO, error) {
	if input.RecipientID == "" {
		return nil, domain.ErrEmptyRecipientID
	}
	notifType := domain.NotificationType(input.Type)
	if !notifType.IsValid() {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidNotificationType, input.Type)
	}

	// Load preferences (or create defaults if not found).
	prefs, err := s.deps.Repos.Preferences.GetByRecipient(ctx, s.pool, input.RecipientType, input.RecipientID)
	if err != nil {
		if err == domain.ErrPreferenceNotFound {
			// Create default preferences
			now := s.deps.Clock.Now()
			newPrefs, _ := domain.NewUserNotificationPreferences(
				s.deps.IDGenerator.NewID(), input.RecipientType, input.RecipientID, "", "", now,
			)
			_ = s.deps.Repos.Preferences.Upsert(ctx, s.pool, newPrefs)
			prefs = &newPrefs
		} else {
			return nil, fmt.Errorf("load prefs: %w", err)
		}
	}

	// Determine which channels to send via.
	channels := make([]domain.Channel, 0, 3)
	for _, chStr := range input.Channels {
		ch := domain.Channel(chStr)
		if !ch.IsValid() {
			continue
		}
		switch ch {
		case domain.ChannelPush:
			if prefs.CanSendPush(notifType) {
				channels = append(channels, ch)
			}
		case domain.ChannelSMS:
			if prefs.CanSendSMS(notifType) {
				channels = append(channels, ch)
			}
		case domain.ChannelEmail:
			if prefs.CanSendEmail(notifType) {
				channels = append(channels, ch)
			}
		}
	}

	if len(channels) == 0 {
		return nil, domain.ErrNoChannelEnabled
	}

	// Create + send one notification per channel.
	results := make([]port.NotificationDTO, 0, len(channels))
	for _, ch := range channels {
		now := s.deps.Clock.Now()
		notifID := s.deps.IDGenerator.NewID()
		notif, err := domain.NewNotification(
			notifID, input.RecipientType, input.RecipientID,
			notifType, ch,
			input.Title, input.TitleAr, input.Body, input.BodyAr,
			input.Data, input.Priority, 3, now,
		)
		if err != nil {
			return nil, err
		}

		// Persist
		if err := s.deps.Repos.Notifications.Create(ctx, s.pool, notif); err != nil {
			return nil, err
		}

		// Send
		sent, err := s.sendViaChannel(ctx, notif, prefs)
		if err != nil {
			// Mark as failed
			failed, _ := notif.MarkFailed(err.Error(), s.deps.Clock.Now())
			_ = s.deps.Repos.Notifications.Update(ctx, s.pool, failed)
			s.publishFailedEvent(ctx, failed)
			results = append(results, port.ToNotificationDTO(failed))
			continue
		}

		// Mark as sent
		_ = s.deps.Repos.Notifications.Update(ctx, s.pool, sent)
		s.publishSentEvent(ctx, sent)
		results = append(results, port.ToNotificationDTO(sent))
	}

	return results, nil
}

// sendViaChannel sends the notification via the appropriate provider.
func (s *Service) sendViaChannel(ctx context.Context, n domain.Notification, prefs *domain.UserNotificationPreferences) (domain.Notification, error) {
	now := s.deps.Clock.Now()

	// Mark as sending
	sending, err := n.MarkSending(now)
	if err != nil {
		return n, err
	}
	_ = s.deps.Repos.Notifications.Update(ctx, s.pool, sending)

	switch n.Channel() {
	case domain.ChannelPush:
		if s.deps.PushProvider == nil {
			return sending, fmt.Errorf("push provider not configured")
		}
		err := s.deps.PushProvider.Send(ctx, port.PushInput{
			DeviceTokens: prefs.DeviceTokens(),
			Title:        n.Title(),
			Body:         n.Body(),
			Data:         n.Data(),
			Priority:     n.Priority(),
		})
		if err != nil {
			return sending, err
		}

	case domain.ChannelSMS:
		if s.deps.SMSProvider == nil {
			return sending, fmt.Errorf("sms provider not configured")
		}
		err := s.deps.SMSProvider.Send(ctx, port.SMSInput{
			To:      prefs.PhoneNumber(),
			Message: n.Title() + ": " + n.Body(),
		})
		if err != nil {
			return sending, err
		}

	case domain.ChannelEmail:
		if s.deps.EmailProvider == nil {
			return sending, fmt.Errorf("email provider not configured")
		}
		err := s.deps.EmailProvider.Send(ctx, port.EmailInput{
			To:      prefs.Email(),
			Subject: n.Title(),
			Body:    n.Body(),
			IsHTML:  false,
		})
		if err != nil {
			return sending, err
		}
	}

	// Mark as sent
	sent, _ := sending.MarkSent(s.deps.Clock.Now())
	return sent, nil
}

func (s *Service) publishSentEvent(ctx context.Context, n domain.Notification) {
	ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
	envelope, err := events.NotificationSentEnvelope(port.NotificationSentPayload{
		NotificationID: n.ID(),
		RecipientID:    n.RecipientID(),
		RecipientType:  n.RecipientType(),
		Channel:        string(n.Channel()),
		Type:           string(n.Type()),
	}, ec)
	if err != nil {
		return
	}
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	})
}

func (s *Service) publishFailedEvent(ctx context.Context, n domain.Notification) {
	ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
	envelope, err := events.NotificationFailedEnvelope(port.NotificationFailedPayload{
		NotificationID: n.ID(),
		RecipientID:    n.RecipientID(),
		Channel:        string(n.Channel()),
		Error:          n.LastError(),
		RetryCount:     n.RetryCount(),
	}, ec)
	if err != nil {
		return
	}
	_ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	})
}

// ===== GetNotification / ListNotificationsByRecipient =====

func (s *Service) GetNotification(ctx context.Context, id string) (*port.NotificationDTO, error) {
	n, err := s.deps.Repos.Notifications.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	return port.ToNotificationDTOPtr(*n), nil
}

func (s *Service) ListNotificationsByRecipient(ctx context.Context, recipientType, recipientID string, page port.PageQuery) (port.Page[port.NotificationDTO], error) {
	result, err := s.deps.Repos.Notifications.ListByRecipient(ctx, s.pool, recipientType, recipientID, page)
	if err != nil {
		return port.Page[port.NotificationDTO]{}, err
	}
	dtos := make([]port.NotificationDTO, 0, len(result.Items))
	for _, n := range result.Items {
		dtos = append(dtos, port.ToNotificationDTO(n))
	}
	return port.Page[port.NotificationDTO]{
		Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset,
	}, nil
}

// ===== RetryFailed =====

func (s *Service) RetryFailed(ctx context.Context, notificationID string) (*port.NotificationDTO, error) {
	n, err := s.deps.Repos.Notifications.GetByID(ctx, s.pool, notificationID)
	if err != nil {
		return nil, err
	}
	retried, err := n.Retry(s.deps.Clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.deps.Repos.Notifications.Update(ctx, s.pool, retried); err != nil {
		return nil, err
	}

	// Try to send again
	prefs, err := s.deps.Repos.Preferences.GetByRecipient(ctx, s.pool, retried.RecipientType(), retried.RecipientID())
	if err != nil {
		return port.ToNotificationDTOPtr(retried), nil
	}
	sent, err := s.sendViaChannel(ctx, retried, prefs)
	if err != nil {
		failed, _ := retried.MarkFailed(err.Error(), s.deps.Clock.Now())
		_ = s.deps.Repos.Notifications.Update(ctx, s.pool, failed)
		return port.ToNotificationDTOPtr(failed), nil
	}
	_ = s.deps.Repos.Notifications.Update(ctx, s.pool, sent)
	return port.ToNotificationDTOPtr(sent), nil
}

// ===== ProcessPending =====
//
// Called by a background worker to send pending notifications.

func (s *Service) ProcessPending(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	pending, err := s.deps.Repos.Notifications.ListPending(ctx, s.pool, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, n := range pending {
		prefs, err := s.deps.Repos.Preferences.GetByRecipient(ctx, s.pool, n.RecipientType(), n.RecipientID())
		if err != nil {
			continue
		}
		sent, err := s.sendViaChannel(ctx, n, prefs)
		if err != nil {
			failed, _ := n.MarkFailed(err.Error(), s.deps.Clock.Now())
			_ = s.deps.Repos.Notifications.Update(ctx, s.pool, failed)
		} else {
			_ = s.deps.Repos.Notifications.Update(ctx, s.pool, sent)
		}
		processed++
	}
	return processed, nil
}

// ===== Preferences =====

func (s *Service) GetPreferences(ctx context.Context, recipientType, recipientID string) (*port.PreferenceDTO, error) {
	prefs, err := s.deps.Repos.Preferences.GetByRecipient(ctx, s.pool, recipientType, recipientID)
	if err != nil {
		if err == domain.ErrPreferenceNotFound {
			// Create default prefs
			now := s.deps.Clock.Now()
			newPrefs, _ := domain.NewUserNotificationPreferences(
				s.deps.IDGenerator.NewID(), recipientType, recipientID, "", "", now,
			)
			_ = s.deps.Repos.Preferences.Upsert(ctx, s.pool, newPrefs)
			return port.ToPreferenceDTOPtr(newPrefs), nil
		}
		return nil, err
	}
	return port.ToPreferenceDTOPtr(*prefs), nil
}

func (s *Service) UpdatePreferences(ctx context.Context, input port.UpdatePreferenceInput) (*port.PreferenceDTO, error) {
	// Load existing or create defaults
	prefs, err := s.deps.Repos.Preferences.GetByRecipient(ctx, s.pool, input.RecipientType, input.RecipientID)
	if err != nil {
		if err == domain.ErrPreferenceNotFound {
			now := s.deps.Clock.Now()
			newPrefs, _ := domain.NewUserNotificationPreferences(
				s.deps.IDGenerator.NewID(), input.RecipientType, input.RecipientID, "", "", now,
			)
			prefs = &newPrefs
		} else {
			return nil, err
		}
	}

	updated := *prefs
	if input.PhoneNumber != "" {
		updated = updated.SetPhoneNumber(input.PhoneNumber)
	}
	if input.Email != "" {
		updated = updated.SetEmail(input.Email)
	}
	if input.DeviceToken != "" {
		updated = updated.AddDeviceToken(input.DeviceToken)
	}
	if input.RemoveToken != "" {
		updated = updated.RemoveDeviceToken(input.RemoveToken)
	}
	for key, val := range input.Prefs {
		// Parse "category:channel"
		var cat domain.Category
		var ch domain.Channel
		fmt.Sscanf(key, "%[^:]:%s", &cat, &ch)
		if cat != "" && ch.IsValid() {
			updated = updated.SetEnabled(cat, ch, val)
		}
	}

	if err := s.deps.Repos.Preferences.Upsert(ctx, s.pool, updated); err != nil {
		return nil, err
	}
	return port.ToPreferenceDTOPtr(updated), nil
}
