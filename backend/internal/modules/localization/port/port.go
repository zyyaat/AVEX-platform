// Package port: repository + service interfaces + DTOs.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/localization/domain"
)

type Executor interface{}
type TxRunner interface{ WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error }
type Row interface{ Scan(dest ...any) error }
type Rows interface{ Next() bool; Scan(dest ...any) error; Err() error; Close() }
type PageQuery struct{ Limit, Offset int }
const (DefaultPageLimit = 50; MaxPageLimit = 100)
func (p PageQuery) Normalize() PageQuery {
	if p.Limit <= 0 { p.Limit = DefaultPageLimit }
	if p.Limit > MaxPageLimit { p.Limit = MaxPageLimit }
	if p.Offset < 0 { p.Offset = 0 }
	return p
}
type Page[T any] struct{ Items []T; Total int64; Limit, Offset int }

// ===== Repository Interfaces =====
type LanguageRepository interface {
	Create(ctx context.Context, exec Executor, l domain.Language) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Language, error)
	GetByCode(ctx context.Context, exec Executor, code string) (*domain.Language, error)
	GetDefault(ctx context.Context, exec Executor) (*domain.Language, error)
	Update(ctx context.Context, exec Executor, l domain.Language) error
	Delete(ctx context.Context, exec Executor, id string) error
	ListAll(ctx context.Context, exec Executor) ([]domain.Language, error)
	ListActive(ctx context.Context, exec Executor) ([]domain.Language, error)
}
type TranslationRepository interface {
	Upsert(ctx context.Context, exec Executor, t domain.Translation) error
	GetByKey(ctx context.Context, exec Executor, languageCode, key string) (*domain.Translation, error)
	Delete(ctx context.Context, exec Executor, id string) error
	DeleteByKey(ctx context.Context, exec Executor, languageCode, key string) error
	ListByLanguage(ctx context.Context, exec Executor, languageCode string) ([]domain.Translation, error)
	ListByPrefix(ctx context.Context, exec Executor, languageCode, prefix string) ([]domain.Translation, error)
	BulkGet(ctx context.Context, exec Executor, languageCode string, keys []string) (map[string]domain.Translation, error)
}
type RepositorySet struct{ Languages LanguageRepository; Translations TranslationRepository }

// ===== Infra =====
type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Logger interface {
	Debug(msg string, args ...any); Info(msg string, args ...any)
	Warn(msg string, args ...any); Error(msg string, args ...any)
}
type Deps struct{ Clock Clock; IDGenerator IDGenerator; Logger Logger; TxRunner TxRunner; Repos RepositorySet }
