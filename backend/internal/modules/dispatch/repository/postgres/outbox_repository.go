// Package postgres outbox_repository: OutboxRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/dispatch/port"
)

// OutboxRepository implements port.OutboxRepository.
type OutboxRepository struct{}

var _ port.OutboxRepository = (*OutboxRepository)(nil)

// Save persists an event envelope in the outbox.
func (r *OutboxRepository) Save(ctx context.Context, exec port.Executor, envelope port.EventEnvelope) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO dispatch.outbox (
			event_id, event_type, event_version, schema_version,
			payload, occurred_at, producer,
			correlation_id, trace_id,
			actor_type, actor_id, actor_ip, actor_user_agent,
			next_retry_at
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

// GetPending retrieves up to limit unpublished events.
func (r *OutboxRepository) GetPending(ctx context.Context, exec port.Executor, limit int) ([]port.EventEnvelope, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT event_id, event_type, event_version, schema_version,
		       payload, occurred_at, producer,
		       correlation_id, trace_id,
		       actor_type, actor_id, actor_ip, actor_user_agent
		FROM dispatch.outbox
		WHERE published_at IS NULL AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT $1
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

// MarkPublished marks an event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, exec port.Executor, eventID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE dispatch.outbox SET published_at = NOW(), last_error = NULL WHERE event_id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

// ===== Worker Helpers =====

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
		UPDATE dispatch.outbox
		SET retry_count = retry_count + 1,
		    last_error = $2,
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
		       payload, occurred_at, producer,
		       correlation_id, trace_id,
		       actor_type, actor_id, actor_ip, actor_user_agent
		FROM dispatch.outbox
		WHERE published_at IS NULL AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending: %w", err)
	}
	defer rows.Close()

	var entries []OutboxEntryWithID
	for rows.Next() {
		var (
			entryID int64
			env     port.EventEnvelope
		)
		err := rows.Scan(
			&entryID,
			&env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion,
			&env.Payload, &env.OccurredAt, &env.Producer,
			&env.CorrelationID, &env.TraceID,
			&env.ActorType, &env.ActorID, &env.ActorIP, &env.ActorUA,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		entries = append(entries, OutboxEntryWithID{EntryID: entryID, Envelope: env})
	}
	return entries, rows.Err()
}

// OutboxEntryWithID bundles the internal DB ID with the event envelope.
type OutboxEntryWithID struct {
	EntryID  int64
	Envelope port.EventEnvelope
}

func (r *OutboxRepository) MarkPublishedByID(ctx context.Context, pool *pgxpool.Pool, entryID int64) error {
	_, err := pool.Exec(ctx, `UPDATE dispatch.outbox SET published_at = NOW(), last_error = NULL WHERE id = $1`, entryID)
	if err != nil {
		return fmt.Errorf("mark published by id: %w", err)
	}
	return nil
}

// ===== Helpers =====

func scanOutboxEnvelope(s scanner) (port.EventEnvelope, error) {
	var env port.EventEnvelope
	var (
		correlationID, traceID        *string
		actorType, actorID            *string
		actorIP, actorUA              *string
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
