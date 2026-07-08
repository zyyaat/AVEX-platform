// Package postgres message_repository: MessageRepository + AttachmentRepository + OutboxRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/support/domain"
	"avex-backend/internal/modules/support/port"
)

// ===== MessageRepository =====

type MessageRepository struct{}

var _ port.MessageRepository = (*MessageRepository)(nil)

func (r *MessageRepository) Create(ctx context.Context, exec port.Executor, m domain.TicketMessage) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO support.ticket_messages (id, ticket_id, sender_type, sender_id, body, edited_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID(), m.TicketID(), string(m.SenderType()), m.SenderID(), m.Body(), m.EditedAt(), m.CreatedAt())
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func (r *MessageRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.TicketMessage, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, ticket_id, sender_type, sender_id, body, edited_at, created_at FROM support.ticket_messages WHERE id = $1`, id)
	m, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("message read: %w", err)
	}
	return &m, nil
}

func (r *MessageRepository) Update(ctx context.Context, exec port.Executor, m domain.TicketMessage) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `UPDATE support.ticket_messages SET body = $2, edited_at = $3 WHERE id = $1`,
		m.ID(), m.Body(), m.EditedAt())
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

func (r *MessageRepository) ListByTicket(ctx context.Context, exec port.Executor, ticketID string, page port.PageQuery) (port.Page[domain.TicketMessage], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	var total int64
	if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM support.ticket_messages WHERE ticket_id = $1`, ticketID).Scan(&total); err != nil {
		return port.Page[domain.TicketMessage]{}, fmt.Errorf("count: %w", err)
	}

	rows, err := dbtx.Query(ctx, `
		SELECT id, ticket_id, sender_type, sender_id, body, edited_at, created_at
		FROM support.ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, ticketID, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.TicketMessage]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.TicketMessage
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return port.Page[domain.TicketMessage]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.TicketMessage]{}, fmt.Errorf("rows: %w", err)
	}
	return port.Page[domain.TicketMessage]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func scanMessage(s scanner) (domain.TicketMessage, error) {
	var (
		id, ticketID, senderType, senderID, body string
		editedAt                                 *time.Time
		createdAt                                time.Time
	)
	if err := s.Scan(&id, &ticketID, &senderType, &senderID, &body, &editedAt, &createdAt); err != nil {
		return domain.TicketMessage{}, err
	}
	return domain.RehydrateTicketMessage(id, ticketID, domain.MessageType(senderType), senderID, body, editedAt, createdAt), nil
}

// ===== AttachmentRepository =====

type AttachmentRepository struct{}

var _ port.AttachmentRepository = (*AttachmentRepository)(nil)

func (r *AttachmentRepository) Create(ctx context.Context, exec port.Executor, a domain.TicketAttachment) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO support.ticket_attachments (id, message_id, file_name, file_type, file_url, file_size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, a.ID(), a.MessageID(), a.FileName(), a.FileType(), a.FileURL(), a.FileSize(), a.CreatedAt())
	if err != nil {
		return fmt.Errorf("create attachment: %w", err)
	}
	return nil
}

func (r *AttachmentRepository) ListByMessage(ctx context.Context, exec port.Executor, messageID string) ([]domain.TicketAttachment, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT id, message_id, file_name, file_type, file_url, file_size, created_at
		FROM support.ticket_attachments WHERE message_id = $1
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("list by message: %w", err)
	}
	defer rows.Close()
	return scanAttachments(rows)
}

func (r *AttachmentRepository) ListByTicket(ctx context.Context, exec port.Executor, ticketID string) ([]domain.TicketAttachment, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT a.id, a.message_id, a.file_name, a.file_type, a.file_url, a.file_size, a.created_at
		FROM support.ticket_attachments a
		JOIN support.ticket_messages m ON m.id = a.message_id
		WHERE m.ticket_id = $1
	`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list by ticket: %w", err)
	}
	defer rows.Close()
	return scanAttachments(rows)
}

func scanAttachments(rows pgx.Rows) ([]domain.TicketAttachment, error) {
	var items []domain.TicketAttachment
	for rows.Next() {
		var (
			id, messageID, fileName, fileType, fileURL string
			fileSize                                   int64
			createdAt                                  time.Time
		)
		if err := rows.Scan(&id, &messageID, &fileName, &fileType, &fileURL, &fileSize, &createdAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, domain.RehydrateTicketAttachment(id, messageID, fileName, fileType, fileURL, fileSize, createdAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ===== OutboxRepository =====

type OutboxRepository struct{}

var _ port.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Save(ctx context.Context, exec port.Executor, envelope port.EventEnvelope) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO support.outbox (
			event_id, event_type, event_version, schema_version,
			payload, occurred_at, producer,
			correlation_id, trace_id,
			actor_type, actor_id, actor_ip, actor_user_agent, next_retry_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
	`,
		envelope.EventID, envelope.EventType, envelope.EventVersion, envelope.SchemaVersion,
		envelope.Payload, envelope.OccurredAt, envelope.Producer,
		nilIfEmptyStr(envelope.CorrelationID), nilIfEmptyStr(envelope.TraceID),
		nilIfEmptyStr(envelope.ActorType), nilIfEmptyStr(envelope.ActorID),
		nilIfEmptyStr(envelope.ActorIP), nilIfEmptyStr(envelope.ActorUA),
	)
	if err != nil {
		return fmt.Errorf("save outbox: %w", err)
	}
	return nil
}

func (r *OutboxRepository) GetPending(ctx context.Context, exec port.Executor, limit int) ([]port.EventEnvelope, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT event_id, event_type, event_version, schema_version,
		       payload, occurred_at, producer, correlation_id, trace_id,
		       actor_type, actor_id, actor_ip, actor_user_agent
		FROM support.outbox
		WHERE published_at IS NULL AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending: %w", err)
	}
	defer rows.Close()

	var envelopes []port.EventEnvelope
	for rows.Next() {
		env, err := scanOutboxEnvelope(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		envelopes = append(envelopes, env)
	}
	return envelopes, rows.Err()
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, exec port.Executor, eventID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE support.outbox SET published_at = NOW(), last_error = NULL WHERE event_id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

// Worker helpers
func (r *OutboxRepository) MarkFailed(ctx context.Context, pool *pgxpool.Pool, eventID string, err error) error {
	errMsg := ""
	if err != nil {
		if len(err.Error()) > 2000 {
			errMsg = err.Error()[:2000]
		} else {
			errMsg = err.Error()
		}
	}
	_, execErr := pool.Exec(ctx, `
		UPDATE support.outbox
		SET retry_count = retry_count + 1, last_error = $2,
		    next_retry_at = NOW() + make_interval(secs => LEAST(1 * POWER(2, retry_count), 3600))
		WHERE event_id = $1
	`, eventID, errMsg)
	if execErr != nil {
		return fmt.Errorf("mark failed: %w", execErr)
	}
	return nil
}

func (r *OutboxRepository) FetchPendingWithIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxEntryWithID, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, event_id, event_type, event_version, schema_version,
		       payload, occurred_at, producer, correlation_id, trace_id,
		       actor_type, actor_id, actor_ip, actor_user_agent
		FROM support.outbox
		WHERE published_at IS NULL AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending: %w", err)
	}
	defer rows.Close()

	var entries []OutboxEntryWithID
	for rows.Next() {
		var entryID int64
		var env port.EventEnvelope
		if err := rows.Scan(
			&entryID,
			&env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion,
			&env.Payload, &env.OccurredAt, &env.Producer,
			&env.CorrelationID, &env.TraceID,
			&env.ActorType, &env.ActorID, &env.ActorIP, &env.ActorUA,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		entries = append(entries, OutboxEntryWithID{EntryID: entryID, Envelope: env})
	}
	return entries, rows.Err()
}

type OutboxEntryWithID struct {
	EntryID  int64
	Envelope port.EventEnvelope
}

func (r *OutboxRepository) MarkPublishedByID(ctx context.Context, pool *pgxpool.Pool, entryID int64) error {
	_, err := pool.Exec(ctx, `UPDATE support.outbox SET published_at = NOW(), last_error = NULL WHERE id = $1`, entryID)
	if err != nil {
		return fmt.Errorf("mark published by id: %w", err)
	}
	return nil
}

func scanOutboxEnvelope(s scanner) (port.EventEnvelope, error) {
	var env port.EventEnvelope
	var (
		correlationID, traceID       *string
		actorType, actorID           *string
		actorIP, actorUA             *string
	)
	if err := s.Scan(
		&env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion,
		&env.Payload, &env.OccurredAt, &env.Producer,
		&correlationID, &traceID,
		&actorType, &actorID, &actorIP, &actorUA,
	); err != nil {
		return port.EventEnvelope{}, err
	}
	if correlationID != nil {
		env.CorrelationID = *correlationID
	}
	if traceID != nil {
		env.TraceID = *traceID
	}
	if actorType != nil {
		env.ActorType = *actorType
	}
	if actorID != nil {
		env.ActorID = *actorID
	}
	if actorIP != nil {
		env.ActorIP = *actorIP
	}
	if actorUA != nil {
		env.ActorUA = *actorUA
	}
	return env, nil
}

var _ = time.Now
var _ = errors.Is
var _ = pgx.ErrNoRows
