// Package postgres preference_repository: PreferenceRepository implementation.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/notifications/domain"
	"avex-backend/internal/modules/notifications/port"
)

type PreferenceRepository struct{}

var _ port.PreferenceRepository = (*PreferenceRepository)(nil)

func (r *PreferenceRepository) GetByRecipient(ctx context.Context, exec port.Executor, recipientType, recipientID string) (*domain.UserNotificationPreferences, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `
		SELECT id, recipient_type, recipient_id, phone_number, email, device_tokens, prefs, created_at, updated_at
		FROM notifications.preferences
		WHERE recipient_type = $1 AND recipient_id = $2
	`, recipientType, recipientID)
	p, err := scanPreference(row)
	if err != nil {
		return nil, mapPrefReadError(err)
	}
	return &p, nil
}

func (r *PreferenceRepository) Upsert(ctx context.Context, exec port.Executor, prefs domain.UserNotificationPreferences) error {
	dbtx := toDBTX(exec)
	prefsJSON, err := json.Marshal(prefs.AllPrefs())
	if err != nil {
		return fmt.Errorf("marshal prefs: %w", err)
	}
	_, err = dbtx.Exec(ctx, `
		INSERT INTO notifications.preferences (
			id, recipient_type, recipient_id, phone_number, email, device_tokens, prefs, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (recipient_type, recipient_id) DO UPDATE SET
			phone_number = EXCLUDED.phone_number,
			email = EXCLUDED.email,
			device_tokens = EXCLUDED.device_tokens,
			prefs = EXCLUDED.prefs,
			updated_at = EXCLUDED.updated_at
	`,
		prefs.ID(), prefs.RecipientType(), prefs.RecipientID(),
		nilIfEmptyStr(prefs.PhoneNumber()), nilIfEmptyStr(prefs.Email()),
		prefs.DeviceTokens(), prefsJSON, prefs.CreatedAt(), prefs.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("upsert prefs: %w", err)
	}
	return nil
}

func scanPreference(s scanner) (domain.UserNotificationPreferences, error) {
	var (
		id, recipientType, recipientID                   string
		phoneNumber, email                               *string
		deviceTokens                                     []string
		prefsRaw                                         []byte
		createdAt, updatedAt                             time.Time
	)
	if err := s.Scan(&id, &recipientType, &recipientID, &phoneNumber, &email, &deviceTokens, &prefsRaw, &createdAt, &updatedAt); err != nil {
		return domain.UserNotificationPreferences{}, err
	}

	var phoneStr, emailStr string
	if phoneNumber != nil {
		phoneStr = *phoneNumber
	}
	if email != nil {
		emailStr = *email
	}

	var prefsMap map[string]bool
	if len(prefsRaw) > 0 {
		_ = json.Unmarshal(prefsRaw, &prefsMap)
	}

	return domain.RehydrateUserNotificationPreferences(
		id, recipientType, recipientID,
		prefsMap, phoneStr, emailStr, deviceTokens,
		createdAt, updatedAt,
	), nil
}

func mapPrefReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPreferenceNotFound
	}
	return fmt.Errorf("preference read: %w", err)
}
