// Package postgres notification_repository: NotificationRepository implementation.
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

type NotificationRepository struct{}

var _ port.NotificationRepository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Create(ctx context.Context, exec port.Executor, n domain.Notification) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO notifications.notifications (
			id, recipient_type, recipient_id, notif_type, channel,
			title, title_ar, body, body_ar, data,
			status, priority, retry_count, max_retries, last_error,
			scheduled_at, sent_at, delivered_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20
		)
	`,
		n.ID(), n.RecipientType(), n.RecipientID(), string(n.Type()), string(n.Channel()),
		n.Title(), nilIfEmptyStr(n.TitleAr()), n.Body(), nilIfEmptyStr(n.BodyAr()), n.DataJSON(),
		string(n.Status()), n.Priority(), n.RetryCount(), n.MaxRetries(), nilIfEmptyStr(n.LastError()),
		n.ScheduledAt(), n.SentAt(), n.DeliveredAt(), n.CreatedAt(), n.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Notification, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+notifColumns+` FROM notifications.notifications WHERE id = $1`, id)
	n, err := scanNotification(row)
	if err != nil {
		return nil, mapNotifReadError(err)
	}
	return &n, nil
}

func (r *NotificationRepository) Update(ctx context.Context, exec port.Executor, n domain.Notification) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE notifications.notifications SET
			status = $2,
			retry_count = $3,
			last_error = $4,
			sent_at = $5,
			delivered_at = $6,
			updated_at = $7
		WHERE id = $1
	`,
		n.ID(), string(n.Status()), n.RetryCount(), nilIfEmptyStr(n.LastError()),
		n.SentAt(), n.DeliveredAt(), n.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("update notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) ListByRecipient(ctx context.Context, exec port.Executor, recipientType, recipientID string, page port.PageQuery) (port.Page[domain.Notification], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM notifications.notifications WHERE recipient_type = $1 AND recipient_id = $2`, recipientType, recipientID).Scan(&total)
	if err != nil {
		return port.Page[domain.Notification]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT `+notifColumns+`
		FROM notifications.notifications
		WHERE recipient_type = $1 AND recipient_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, recipientType, recipientID, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.Notification]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return port.Page[domain.Notification]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.Notification]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.Notification]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (r *NotificationRepository) ListPending(ctx context.Context, exec port.Executor, limit int) ([]domain.Notification, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+notifColumns+`
		FROM notifications.notifications
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()

	var items []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

const notifColumns = `id, recipient_type, recipient_id, notif_type, channel, title, title_ar, body, body_ar, data, status, priority, retry_count, max_retries, last_error, scheduled_at, sent_at, delivered_at, created_at, updated_at`

func scanNotification(s scanner) (domain.Notification, error) {
	var (
		id, recipientType, recipientID, notifType, channel string
		title, body                                        string
		titleAr, bodyAr                                    *string
		dataRaw                                            []byte
		status, priority                                   string
		retryCount, maxRetries                             int
		lastError                                          *string
		scheduledAt                                        time.Time
		sentAt, deliveredAt                                *time.Time
		createdAt, updatedAt                               time.Time
	)
	if err := s.Scan(
		&id, &recipientType, &recipientID, &notifType, &channel,
		&title, &titleAr, &body, &bodyAr, &dataRaw,
		&status, &priority, &retryCount, &maxRetries, &lastError,
		&scheduledAt, &sentAt, &deliveredAt, &createdAt, &updatedAt,
	); err != nil {
		return domain.Notification{}, err
	}

	var titleArStr, bodyArStr, lastErrStr string
	if titleAr != nil {
		titleArStr = *titleAr
	}
	if bodyAr != nil {
		bodyArStr = *bodyAr
	}
	if lastError != nil {
		lastErrStr = *lastError
	}

	var dataMap map[string]any
	if len(dataRaw) > 0 {
		_ = json.Unmarshal(dataRaw, &dataMap)
	}

	return domain.RehydrateNotification(
		id, recipientType, recipientID,
		domain.NotificationType(notifType),
		domain.Channel(channel),
		title, titleArStr, body, bodyArStr, dataMap,
		domain.Status(status), priority, retryCount, maxRetries, lastErrStr,
		scheduledAt, sentAt, deliveredAt, createdAt, updatedAt,
	), nil
}

func mapNotifReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotificationNotFound
	}
	return fmt.Errorf("notification read: %w", err)
}
