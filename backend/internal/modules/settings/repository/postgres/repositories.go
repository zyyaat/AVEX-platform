// Package postgres implements the settings module's repository interfaces.
package postgres

import (
        "context"
        "errors"
        "time"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/settings/domain"
        "avex-backend/internal/modules/settings/port"
        "avex-backend/internal/platform/database"
)

type Repositories struct {
        settings  *SettingRepository
        revisions *RevisionRepository
        flags     *FeatureFlagRepository
        outbox    *OutboxRepository
}

func NewRepositories() *Repositories {
        return &Repositories{settings: &SettingRepository{}, revisions: &RevisionRepository{}, flags: &FeatureFlagRepository{}, outbox: &OutboxRepository{}}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
        return port.RepositorySet{Settings: r.settings, Revisions: r.revisions, Flags: r.flags, Outbox: r.outbox}
}

func toDBTX(exec port.Executor) database.DBTX {
        dbtx, ok := exec.(database.DBTX)
        if !ok { panic("postgres: port.Executor does not satisfy database.DBTX") }
        return dbtx
}
type scanner interface{ Scan(dest ...any) error }
func nilIfEmptyStr(s string) any { if s == "" { return nil }; return s }

// ===== SettingRepository =====
type SettingRepository struct{}
var _ port.SettingRepository = (*SettingRepository)(nil)

func (r *SettingRepository) Create(ctx context.Context, exec port.Executor, s domain.Setting) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO settings.settings (id, key, description, setting_type, value, is_protected, version, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
                s.ID(), s.Key(), s.Description(), string(s.Type()), s.Value(), s.IsProtected(), s.Version(), s.CreatedAt(), s.UpdatedAt())
        return err
}
func (r *SettingRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Setting, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT id, key, description, setting_type, value, is_protected, version, created_at, updated_at FROM settings.settings WHERE id=$1`, id)
        s, err := scanSetting(row)
        if err != nil { return nil, mapSettingErr(err) }
        return &s, nil
}
func (r *SettingRepository) GetByKey(ctx context.Context, exec port.Executor, key string) (*domain.Setting, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT id, key, description, setting_type, value, is_protected, version, created_at, updated_at FROM settings.settings WHERE key=$1`, key)
        s, err := scanSetting(row)
        if err != nil { return nil, mapSettingErr(err) }
        return &s, nil
}
func (r *SettingRepository) Update(ctx context.Context, exec port.Executor, s domain.Setting) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `UPDATE settings.settings SET value=$2, version=$3, updated_at=$4 WHERE id=$1`,
                s.ID(), s.Value(), s.Version(), s.UpdatedAt())
        return err
}
func (r *SettingRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
        dbtx := toDBTX(exec)
        tag, err := dbtx.Exec(ctx, `DELETE FROM settings.settings WHERE id=$1 AND is_protected=FALSE`, id)
        if err != nil { return err }
        if tag.RowsAffected() == 0 { return domain.ErrCannotDeleteProtected }
        return nil
}
func (r *SettingRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Setting], error) {
        page = page.Normalize()
        dbtx := toDBTX(exec)
        var total int64
        if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM settings.settings`).Scan(&total); err != nil { return port.Page[domain.Setting]{}, err }
        rows, err := dbtx.Query(ctx, `SELECT id, key, description, setting_type, value, is_protected, version, created_at, updated_at FROM settings.settings ORDER BY key LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
        if err != nil { return port.Page[domain.Setting]{}, err }
        defer rows.Close()
        var items []domain.Setting
        for rows.Next() {
                s, err := scanSetting(rows)
                if err != nil { return port.Page[domain.Setting]{}, err }
                items = append(items, s)
        }
        return port.Page[domain.Setting]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}
func (r *SettingRepository) ListByType(ctx context.Context, exec port.Executor, settingType string) ([]domain.Setting, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT id, key, description, setting_type, value, is_protected, version, created_at, updated_at FROM settings.settings WHERE setting_type=$1 ORDER BY key`, settingType)
        if err != nil { return nil, err }
        defer rows.Close()
        var items []domain.Setting
        for rows.Next() {
                s, err := scanSetting(rows)
                if err != nil { return nil, err }
                items = append(items, s)
        }
        return items, rows.Err()
}
func scanSetting(s scanner) (domain.Setting, error) {
        var id, key, settingType, value string; var desc *string; var isProtected bool; var version int; var createdAt, updatedAt time.Time
        if err := s.Scan(&id, &key, &desc, &settingType, &value, &isProtected, &version, &createdAt, &updatedAt); err != nil { return domain.Setting{}, err }
        var descStr string; if desc != nil { descStr = *desc }
        return domain.RehydrateSetting(id, key, descStr, domain.SettingType(settingType), value, isProtected, version, createdAt, updatedAt), nil
}
func mapSettingErr(err error) error {
        if errors.Is(err, pgx.ErrNoRows) { return domain.ErrSettingNotFound }
        return err
}

// ===== RevisionRepository =====
type RevisionRepository struct{}
var _ port.RevisionRepository = (*RevisionRepository)(nil)

func (r *RevisionRepository) Create(ctx context.Context, exec port.Executor, rev domain.SettingRevision) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO settings.setting_revisions (id, setting_id, version, value, changed_by, change_note, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
                rev.ID(), rev.SettingID(), rev.Version(), rev.Value(), nilIfEmptyStr(rev.ChangedBy()), nilIfEmptyStr(rev.ChangeNote()), rev.CreatedAt())
        return err
}
func (r *RevisionRepository) ListBySetting(ctx context.Context, exec port.Executor, settingID string, page port.PageQuery) (port.Page[domain.SettingRevision], error) {
        page = page.Normalize()
        dbtx := toDBTX(exec)
        var total int64
        if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM settings.setting_revisions WHERE setting_id=$1`, settingID).Scan(&total); err != nil { return port.Page[domain.SettingRevision]{}, err }
        rows, err := dbtx.Query(ctx, `SELECT id, setting_id, version, value, changed_by, change_note, created_at FROM settings.setting_revisions WHERE setting_id=$1 ORDER BY version DESC LIMIT $2 OFFSET $3`, settingID, page.Limit, page.Offset)
        if err != nil { return port.Page[domain.SettingRevision]{}, err }
        defer rows.Close()
        var items []domain.SettingRevision
        for rows.Next() {
                var id, settingID, value string; var version int; var changedBy, note *string; var createdAt time.Time
                if err := rows.Scan(&id, &settingID, &version, &value, &changedBy, &note, &createdAt); err != nil { return port.Page[domain.SettingRevision]{}, err }
                var byStr, noteStr string; if changedBy != nil { byStr = *changedBy }; if note != nil { noteStr = *note }
                items = append(items, domain.RehydrateSettingRevision(id, settingID, version, value, byStr, noteStr, createdAt))
        }
        return port.Page[domain.SettingRevision]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}
func (r *RevisionRepository) GetBySettingAndVersion(ctx context.Context, exec port.Executor, settingID string, version int) (*domain.SettingRevision, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT id, setting_id, version, value, changed_by, change_note, created_at FROM settings.setting_revisions WHERE setting_id=$1 AND version=$2`, settingID, version)
        var id, settingIDStr, value string; var changedBy, note *string; var createdAt time.Time
        if err := row.Scan(&id, &settingIDStr, &version, &value, &changedBy, &note, &createdAt); err != nil {
                if errors.Is(err, pgx.ErrNoRows) { return nil, domain.ErrRevisionNotFound }
                return nil, err
        }
        var byStr, noteStr string; if changedBy != nil { byStr = *changedBy }; if note != nil { noteStr = *note }
        rev := domain.RehydrateSettingRevision(id, settingIDStr, version, value, byStr, noteStr, createdAt)
        return &rev, nil
}

// ===== FeatureFlagRepository =====
type FeatureFlagRepository struct{}
var _ port.FeatureFlagRepository = (*FeatureFlagRepository)(nil)

func (r *FeatureFlagRepository) Create(ctx context.Context, exec port.Executor, f domain.FeatureFlag) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO settings.feature_flags (id, name, description, enabled, target_type, target_value, rollout_pct, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
                f.ID(), f.Name(), f.Description(), f.Enabled(), string(f.TargetType()), nilIfEmptyStr(f.TargetValue()), f.RolloutPct(), f.CreatedAt(), f.UpdatedAt())
        return err
}
func (r *FeatureFlagRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.FeatureFlag, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT id, name, description, enabled, target_type, target_value, rollout_pct, created_at, updated_at FROM settings.feature_flags WHERE id=$1`, id)
        f, err := scanFlag(row)
        if err != nil { return nil, mapFlagErr(err) }
        return &f, nil
}
func (r *FeatureFlagRepository) GetByName(ctx context.Context, exec port.Executor, name string) (*domain.FeatureFlag, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT id, name, description, enabled, target_type, target_value, rollout_pct, created_at, updated_at FROM settings.feature_flags WHERE name=$1`, name)
        f, err := scanFlag(row)
        if err != nil { return nil, mapFlagErr(err) }
        return &f, nil
}
func (r *FeatureFlagRepository) Update(ctx context.Context, exec port.Executor, f domain.FeatureFlag) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `UPDATE settings.feature_flags SET enabled=$2, target_type=$3, target_value=$4, rollout_pct=$5, updated_at=$6 WHERE id=$1`,
                f.ID(), f.Enabled(), string(f.TargetType()), nilIfEmptyStr(f.TargetValue()), f.RolloutPct(), f.UpdatedAt())
        return err
}
func (r *FeatureFlagRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `DELETE FROM settings.feature_flags WHERE id=$1`, id)
        return err
}
func (r *FeatureFlagRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.FeatureFlag], error) {
        page = page.Normalize()
        dbtx := toDBTX(exec)
        var total int64
        if err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM settings.feature_flags`).Scan(&total); err != nil { return port.Page[domain.FeatureFlag]{}, err }
        rows, err := dbtx.Query(ctx, `SELECT id, name, description, enabled, target_type, target_value, rollout_pct, created_at, updated_at FROM settings.feature_flags ORDER BY name LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
        if err != nil { return port.Page[domain.FeatureFlag]{}, err }
        defer rows.Close()
        var items []domain.FeatureFlag
        for rows.Next() {
                f, err := scanFlag(rows)
                if err != nil { return port.Page[domain.FeatureFlag]{}, err }
                items = append(items, f)
        }
        return port.Page[domain.FeatureFlag]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, rows.Err()
}
func (r *FeatureFlagRepository) ListEnabled(ctx context.Context, exec port.Executor) ([]domain.FeatureFlag, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT id, name, description, enabled, target_type, target_value, rollout_pct, created_at, updated_at FROM settings.feature_flags WHERE enabled=TRUE ORDER BY name`)
        if err != nil { return nil, err }
        defer rows.Close()
        var items []domain.FeatureFlag
        for rows.Next() {
                f, err := scanFlag(rows)
                if err != nil { return nil, err }
                items = append(items, f)
        }
        return items, rows.Err()
}
func scanFlag(s scanner) (domain.FeatureFlag, error) {
        var id, name, targetType string; var desc, targetValue *string; var enabled bool; var rolloutPct int; var createdAt, updatedAt time.Time
        if err := s.Scan(&id, &name, &desc, &enabled, &targetType, &targetValue, &rolloutPct, &createdAt, &updatedAt); err != nil { return domain.FeatureFlag{}, err }
        var descStr, tvStr string; if desc != nil { descStr = *desc }; if targetValue != nil { tvStr = *targetValue }
        return domain.RehydrateFeatureFlag(id, name, descStr, enabled, domain.TargetType(targetType), tvStr, rolloutPct, createdAt, updatedAt), nil
}
func mapFlagErr(err error) error {
        if errors.Is(err, pgx.ErrNoRows) { return domain.ErrFeatureFlagNotFound }
        return err
}

// ===== OutboxRepository =====
type OutboxRepository struct{}
var _ port.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Save(ctx context.Context, exec port.Executor, env port.EventEnvelope) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO settings.outbox (event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent, next_retry_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW())`,
                env.EventID, env.EventType, env.EventVersion, env.SchemaVersion, env.Payload, env.OccurredAt, env.Producer,
                nilIfEmptyStr(env.CorrelationID), nilIfEmptyStr(env.TraceID), nilIfEmptyStr(env.ActorType), nilIfEmptyStr(env.ActorID), nilIfEmptyStr(env.ActorIP), nilIfEmptyStr(env.ActorUA))
        return err
}
func (r *OutboxRepository) GetPending(ctx context.Context, exec port.Executor, limit int) ([]port.EventEnvelope, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM settings.outbox WHERE published_at IS NULL AND next_retry_at <= NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
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
        _, err := dbtx.Exec(ctx, `UPDATE settings.outbox SET published_at=NOW(), last_error=NULL WHERE event_id=$1`, eventID)
        return err
}

// Worker helpers
func (r *OutboxRepository) MarkFailed(ctx context.Context, pool *pgxpool.Pool, eventID string, err error) error {
        errMsg := ""; if err != nil { if len(err.Error()) > 2000 { errMsg = err.Error()[:2000] } else { errMsg = err.Error() } }
        _, execErr := pool.Exec(ctx, `UPDATE settings.outbox SET retry_count=retry_count+1, last_error=$2, next_retry_at=NOW()+make_interval(secs=>LEAST(1*POWER(2,retry_count),3600)) WHERE event_id=$1`, eventID, errMsg)
        return execErr
}
func (r *OutboxRepository) FetchPendingWithIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]OutboxEntryWithID, error) {
        rows, err := pool.Query(ctx, `SELECT id, event_id, event_type, event_version, schema_version, payload, occurred_at, producer, correlation_id, trace_id, actor_type, actor_id, actor_ip, actor_user_agent FROM settings.outbox WHERE published_at IS NULL AND next_retry_at<=NOW() ORDER BY next_retry_at ASC LIMIT $1`, limit)
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
        _, err := pool.Exec(ctx, `UPDATE settings.outbox SET published_at=NOW(), last_error=NULL WHERE id=$1`, entryID)
        return err
}
