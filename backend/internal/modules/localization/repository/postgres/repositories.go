// Package postgres implements the localization module's repository interfaces.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/localization/domain"
	"avex-backend/internal/modules/localization/port"
	"avex-backend/internal/platform/database"
)

type Repositories struct{ langs *LanguageRepository; trans *TranslationRepository }

func NewRepositories() *Repositories {
	return &Repositories{langs: &LanguageRepository{}, trans: &TranslationRepository{}}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{Languages: r.langs, Translations: r.trans}
}

func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok { panic("postgres: port.Executor does not satisfy database.DBTX") }
	return dbtx
}
type scanner interface{ Scan(dest ...any) error }
func nilIfEmptyStr(s string) any { if s == "" { return nil }; return s }

// ===== LanguageRepository =====
type LanguageRepository struct{}
var _ port.LanguageRepository = (*LanguageRepository)(nil)

func (r *LanguageRepository) Create(ctx context.Context, exec port.Executor, l domain.Language) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `INSERT INTO localization.languages (id, code, name, is_rtl, is_default, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		l.ID(), l.Code(), l.Name(), l.IsRTL(), l.IsDefault(), l.IsActive(), l.CreatedAt(), l.UpdatedAt())
	return err
}
func (r *LanguageRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Language, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, code, name, is_rtl, is_default, is_active, created_at, updated_at FROM localization.languages WHERE id=$1`, id)
	l, err := scanLang(row)
	if err != nil { return nil, mapLangErr(err) }
	return &l, nil
}
func (r *LanguageRepository) GetByCode(ctx context.Context, exec port.Executor, code string) (*domain.Language, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, code, name, is_rtl, is_default, is_active, created_at, updated_at FROM localization.languages WHERE code=$1`, code)
	l, err := scanLang(row)
	if err != nil { return nil, mapLangErr(err) }
	return &l, nil
}
func (r *LanguageRepository) GetDefault(ctx context.Context, exec port.Executor) (*domain.Language, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, code, name, is_rtl, is_default, is_active, created_at, updated_at FROM localization.languages WHERE is_default=TRUE LIMIT 1`)
	l, err := scanLang(row)
	if err != nil { return nil, mapLangErr(err) }
	return &l, nil
}
func (r *LanguageRepository) Update(ctx context.Context, exec port.Executor, l domain.Language) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `UPDATE localization.languages SET name=$2, is_rtl=$3, is_active=$4, updated_at=$5 WHERE id=$1`,
		l.ID(), l.Name(), l.IsRTL(), l.IsActive(), l.UpdatedAt())
	return err
}
func (r *LanguageRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `DELETE FROM localization.languages WHERE id=$1 AND is_default=FALSE`, id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return domain.ErrCannotDeleteDefaultLanguage }
	return nil
}
func (r *LanguageRepository) ListAll(ctx context.Context, exec port.Executor) ([]domain.Language, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, code, name, is_rtl, is_default, is_active, created_at, updated_at FROM localization.languages ORDER BY code`)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.Language
	for rows.Next() {
		l, err := scanLang(rows)
		if err != nil { return nil, err }
		items = append(items, l)
	}
	return items, rows.Err()
}
func (r *LanguageRepository) ListActive(ctx context.Context, exec port.Executor) ([]domain.Language, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, code, name, is_rtl, is_default, is_active, created_at, updated_at FROM localization.languages WHERE is_active=TRUE ORDER BY code`)
	if err != nil { return nil, err }
	defer rows.Close()
	var items []domain.Language
	for rows.Next() {
		l, err := scanLang(rows)
		if err != nil { return nil, err }
		items = append(items, l)
	}
	return items, rows.Err()
}
func scanLang(s scanner) (domain.Language, error) {
	var id, code, name string; var isRTL, isDefault, isActive bool; var createdAt, updatedAt time.Time
	if err := s.Scan(&id, &code, &name, &isRTL, &isDefault, &isActive, &createdAt, &updatedAt); err != nil { return domain.Language{}, err }
	return domain.RehydrateLanguage(id, code, name, isRTL, isDefault, isActive, createdAt, updatedAt), nil
}
func mapLangErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) { return domain.ErrLanguageNotFound }
	return err
}

// ===== TranslationRepository =====
type TranslationRepository struct{}
var _ port.TranslationRepository = (*TranslationRepository)(nil)

func (r *TranslationRepository) Upsert(ctx context.Context, exec port.Executor, t domain.Translation) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO localization.translations (id, language_code, key, value, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (language_code, key) DO UPDATE SET value=EXCLUDED.value, updated_at=EXCLUDED.updated_at
	`, t.ID(), t.LanguageCode(), t.Key(), t.Value(), t.CreatedAt(), t.UpdatedAt())
	return err
}
func (r *TranslationRepository) GetByKey(ctx context.Context, exec port.Executor, languageCode, key string) (*domain.Translation, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT id, language_code, key, value, created_at, updated_at FROM localization.translations WHERE language_code=$1 AND key=$2`, languageCode, key)
	t, err := scanTrans(row)
	if err != nil { return nil, mapTransErr(err) }
	return &t, nil
}
func (r *TranslationRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `DELETE FROM localization.translations WHERE id=$1`, id)
	return err
}
func (r *TranslationRepository) DeleteByKey(ctx context.Context, exec port.Executor, languageCode, key string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `DELETE FROM localization.translations WHERE language_code=$1 AND key=$2`, languageCode, key)
	return err
}
func (r *TranslationRepository) ListByLanguage(ctx context.Context, exec port.Executor, languageCode string) ([]domain.Translation, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, language_code, key, value, created_at, updated_at FROM localization.translations WHERE language_code=$1 ORDER BY key`, languageCode)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanTransList(rows)
}
func (r *TranslationRepository) ListByPrefix(ctx context.Context, exec port.Executor, languageCode, prefix string) ([]domain.Translation, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, language_code, key, value, created_at, updated_at FROM localization.translations WHERE language_code=$1 AND key LIKE $2 ORDER BY key`, languageCode, prefix+"%")
	if err != nil { return nil, err }
	defer rows.Close()
	return scanTransList(rows)
}
func (r *TranslationRepository) BulkGet(ctx context.Context, exec port.Executor, languageCode string, keys []string) (map[string]domain.Translation, error) {
	if len(keys) == 0 { return map[string]domain.Translation{}, nil }
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `SELECT id, language_code, key, value, created_at, updated_at FROM localization.translations WHERE language_code=$1 AND key = ANY($2)`, languageCode, keys)
	if err != nil { return nil, err }
	defer rows.Close()
	items, err := scanTransList(rows)
	if err != nil { return nil, err }
	result := make(map[string]domain.Translation, len(items))
	for _, t := range items { result[t.Key()] = t }
	return result, nil
}
func scanTrans(s scanner) (domain.Translation, error) {
	var id, langCode, key, value string; var createdAt, updatedAt time.Time
	if err := s.Scan(&id, &langCode, &key, &value, &createdAt, &updatedAt); err != nil { return domain.Translation{}, err }
	return domain.RehydrateTranslation(id, langCode, key, value, createdAt, updatedAt), nil
}
func scanTransList(rows pgx.Rows) ([]domain.Translation, error) {
	var items []domain.Translation
	for rows.Next() {
		t, err := scanTrans(rows)
		if err != nil { return nil, err }
		items = append(items, t)
	}
	return items, rows.Err()
}
func mapTransErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) { return domain.ErrTranslationNotFound }
	return err
}

var _ = fmt.Sprintf
