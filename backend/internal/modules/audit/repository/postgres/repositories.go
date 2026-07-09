// Package postgres implements the audit module's repository interfaces.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/audit/domain"
	"avex-backend/internal/modules/audit/port"
	"avex-backend/internal/platform/database"
)

type Repositories struct {
	audit  *AuditRepository
	outbox *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{audit: &AuditRepository{}, outbox: &OutboxRepository{}}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{Audit: r.audit, Outbox: r.outbox}
}

func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok { panic("postgres: port.Executor does not satisfy database.DBTX") }
	return dbtx
}
type scanner interface{ Scan(dest ...any) error }
func nilIfEmptyStr(s string) any { if s == "" { return nil }; return s }

// ===== AuditRepository =====
type AuditRepository struct{}
var _ port.AuditRepository = (*AuditRepository)(nil)

func (r *AuditRepository) Create(ctx context.Context, exec port.Executor, e domain.AuditEntry) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO audit.entries (
			id, actor_type, actor_id, action, resource_type, resource_id,
			severity, description, metadata, ip_address, user_agent,
			correlation_id, trace_id, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`,
		e.ID(), string(e.ActorType()), nilIfEmptyStr(e.ActorID()), e.Action(), e.ResourceType(), nilIfEmptyStr(e.ResourceID()),
		string(e.Severity()), e.Description(), e.MetadataJSON(), nilIfEmptyStr(e.IPAddress()), nilIfEmptyStr(e.UserAgent()),
		nilIfEmptyStr(e.CorrelationID()), nilIfEmptyStr(e.TraceID()), e.CreatedAt(),
	)
	if err != nil { return fmt.Errorf("create audit entry: %w", err) }
	return nil
}

func (r *AuditRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.AuditEntry, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+auditColumns+` FROM audit.entries WHERE id=$1`, id)
	e, err := scanAuditEntry(row)
	if err != nil { return nil, mapAuditErr(err) }
	return &e, nil
}

func (r *AuditRepository) listWithFilter(ctx context.Context, exec port.Executor, where string, args []any, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM audit.entries WHERE %s`, where)
	var total int64
	if err := dbtx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil { return port.Page[domain.AuditEntry]{}, fmt.Errorf("count: %w", err) }
	listSQL := fmt.Sprintf(`SELECT %s FROM audit.entries WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, auditColumns, where, len(args)+1, len(args)+2)
	args = append(args, page.Limit, page.Offset)
	rows, err := dbtx.Query(ctx, listSQL, args...)
	if err != nil { return port.Page[domain.AuditEntry]{}, fmt.Errorf("list: %w", err) }
	defer rows.Close()
	var items []domain.AuditEntry
	for rows.Next() {
		e, err := scanAuditEntry(rows)
		if err != nil { return port.Page[domain.AuditEntry]{}, fmt.Errorf("scan: %w", err) }
		items = append(items, e)
	}
	if err := rows.Err(); err != nil { return port.Page[domain.AuditEntry]{}, fmt.Errorf("rows: %w", err) }
	return port.Page[domain.AuditEntry]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (r *AuditRepository) ListByActor(ctx context.Context, exec port.Executor, actorType, actorID string, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "actor_type = $1 AND actor_id = $2", []any{actorType, actorID}, page)
}
func (r *AuditRepository) ListByResource(ctx context.Context, exec port.Executor, resourceType, resourceID string, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "resource_type = $1 AND resource_id = $2", []any{resourceType, resourceID}, page)
}
func (r *AuditRepository) ListByAction(ctx context.Context, exec port.Executor, action string, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "action = $1", []any{action}, page)
}
func (r *AuditRepository) ListBySeverity(ctx context.Context, exec port.Executor, severity string, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "severity = $1", []any{severity}, page)
}
func (r *AuditRepository) ListByTimeRange(ctx context.Context, exec port.Executor, from, to time.Time, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "created_at >= $1 AND created_at <= $2", []any{from, to}, page)
}
func (r *AuditRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.AuditEntry], error) {
	return r.listWithFilter(ctx, exec, "1=1", []any{}, page)
}

func (r *AuditRepository) CountByAction(ctx context.Context, exec port.Executor, from, to time.Time) (map[string]int64, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT action, COUNT(*) as cnt FROM audit.entries WHERE created_at >= $1 AND created_at <= $2 GROUP BY action ORDER BY cnt DESC`, from, to)
	if err != nil { return nil, fmt.Errorf("count by action: %w", err) }
	defer rows.Close()
	result := make(map[string]int64)
	for rows.Next() {
		var action string; var cnt int64
		if err := rows.Scan(&action, &cnt); err != nil { return nil, fmt.Errorf("scan: %w", err) }
		result[action] = cnt
	}
	return result, rows.Err()
}

const auditColumns = `id, actor_type, actor_id, action, resource_type, resource_id, severity, description, metadata, ip_address, user_agent, correlation_id, trace_id, created_at`

func scanAuditEntry(s scanner) (domain.AuditEntry, error) {
	var (
		id, actorType, action, resourceType, severity string
		actorID, resourceID, description, ipAddress, userAgent, correlationID, traceID *string
		metadataRaw []byte
		createdAt time.Time
	)
	if err := s.Scan(&id, &actorType, &actorID, &action, &resourceType, &resourceID, &severity, &description, &metadataRaw, &ipAddress, &userAgent, &correlationID, &traceID, &createdAt); err != nil {
		return domain.AuditEntry{}, err
	}
	var actorIDStr, resourceIDStr, descStr, ipStr, uaStr, corStr, traceStr string
	if actorID != nil { actorIDStr = *actorID }
	if resourceID != nil { resourceIDStr = *resourceID }
	if description != nil { descStr = *description }
	if ipAddress != nil { ipStr = *ipAddress }
	if userAgent != nil { uaStr = *userAgent }
	if correlationID != nil { corStr = *correlationID }
	if traceID != nil { traceStr = *traceID }
	var metaMap map[string]any
	if len(metadataRaw) > 0 { _ = json.Unmarshal(metadataRaw, &metaMap) }
	return domain.RehydrateAuditEntry(id, domain.ActorType(actorType), actorIDStr, action, resourceType, resourceIDStr, domain.Severity(severity), descStr, metaMap, ipStr, uaStr, corStr, traceStr, createdAt), nil
}

func mapAuditErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) { return domain.ErrAuditEntryNotFound }
	return err
}

// ===== OutboxRepository =====
type OutboxRepository struct{}
var _ port.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Save(ctx context.Context, exec port.Executor, env port.EventEnvelope) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO audit.outbox (event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent, next_retry_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW())`,
		env.EventID, env.EventType, env.EventVersion, env.SchemaVersion, env.Payload, env.OccurredAt, env.Producer,
		nilIfEmptyStr(env.CorrelationID), nilIfEmptyStr(env.TraceID), nilIfEmptyStr(env.ActorType), nilIfEmptyStr(env.ActorID), nilIfEmptyStr(env.ActorIP), nilIfEmptyStr(env.ActorUA))
	return err
}
func (r *OutboxRepository) GetPending(ctx context.Context, exec port.Executor, limit int) ([]port.EventEnvelope, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM audit.outbox WHERE published_at IS NULL AND next_retry_at <= NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var envs []port.EventEnvelope
	for rows.Next() {
		var env port.EventEnvelope; var corID, traceID, aType, aID, aIP, aUA *string
		if err := rows.Scan(&env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion, &env.Payload, &env.OccurredAt, &env.Producer, &corID, &traceID, &aType, &aID, &aIP, &aUA); err != nil { return nil, err }
		if corID != nil { env.CorrelationID = *corID }; if traceID != nil { env.TraceID = *traceID }
		if aType != nil { env.ActorType = *aType }; if aID != nil { env.ActorID = *aID }
		if aIP != nil { env.ActorIP = *aIP }; if aUA != nil { env.ActorUA = *aUA }
		envs = append(envs, env)
	}
	return envs, rows.Err()
}
func (r *OutboxRepository) MarkPublished(ctx context.Context, exec port.Executor, eventID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE audit.outbox SET published_at=NOW(), last_error=NULL WHERE event_id=$1`, eventID)
	return err
}

// Worker helpers
func (r *OutboxRepository) MarkFailed(ctx context.Context, pool *pgxpool.Pool, eventID string, err error) error {
	errMsg := ""; if err != nil { if len(err.Error()) > 2000 { errMsg = err.Error()[:2000] } else { errMsg = err.Error() } }
	_, execErr := pool.Exec(ctx, `UPDATE audit.outbox SET retry_count=retry_count+1, last_error=$2, next_retry_at=NOW()+make_interval(secs=>LEAST(1*POWER(2,retry_count),3600)) WHERE event_id=$1`, eventID, errMsg)
	return execErr
}
func (r *OutboxRepository) FetchPendingWithIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxEntryWithID, error) {
	rows, err := pool.Query(ctx, `SELECT id, event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM audit.outbox WHERE published_at IS NULL AND next_retry_at<=NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var entries []OutboxEntryWithID
	for rows.Next() {
		var entryID int64; var env port.EventEnvelope; var corID, traceID, aType, aID, aIP, aUA *string
		if err := rows.Scan(&entryID, &env.EventID, &env.EventType, &env.EventVersion, &env.SchemaVersion, &env.Payload, &env.OccurredAt, &env.Producer, &corID, &traceID, &aType, &aID, &aIP, &aUA); err != nil { return nil, err }
		if corID != nil { env.CorrelationID = *corID }; if traceID != nil { env.TraceID = *traceID }
		if aType != nil { env.ActorType = *aType }; if aID != nil { env.ActorID = *aID }
		if aIP != nil { env.ActorIP = *aIP }; if aUA != nil { env.ActorUA = *aUA }
		entries = append(entries, OutboxEntryWithID{EntryID: entryID, Envelope: env})
	}
	return entries, rows.Err()
}
type OutboxEntryWithID struct{ EntryID int64; Envelope port.EventEnvelope }
func (r *OutboxRepository) MarkPublishedByID(ctx context.Context, pool *pgxpool.Pool, entryID int64) error {
	_, err := pool.Exec(ctx, `UPDATE audit.outbox SET published_at=NOW(), last_error=NULL WHERE id=$1`, entryID)
	return err
}
