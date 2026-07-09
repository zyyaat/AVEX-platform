// Package port service: ServicePort + DTOs.
package port

import (
	"context"

	"avex-backend/internal/modules/localization/domain"
)

// ===== DTOs =====
type CreateLanguageInput struct{ Code, Name string; IsRTL, IsDefault, IsActive bool }
type UpsertTranslationInput struct{ LanguageCode, Key, Value string }

type LanguageDTO struct {
	ID string `json:"id"`; Code string `json:"code"`; Name string `json:"name"`
	IsRTL bool `json:"is_rtl"`; IsDefault bool `json:"is_default"`; IsActive bool `json:"is_active"`
	CreatedAt string `json:"created_at"`; UpdatedAt string `json:"updated_at"`
}
type TranslationDTO struct {
	ID string `json:"id"`; LanguageCode string `json:"language_code"`
	Key string `json:"key"`; Value string `json:"value"`
	CreatedAt string `json:"created_at"`; UpdatedAt string `json:"updated_at"`
}
type TranslateResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Lang  string `json:"lang"`
	Found bool   `json:"found"`
}
type BulkTranslateInput struct {
	LanguageCode string   `json:"language_code"`
	Keys         []string `json:"keys"`
}
type BulkTranslateResult struct {
	LanguageCode string                    `json:"language_code"`
	Translations map[string]TranslateResult `json:"translations"`
}

// ===== ServicePort =====
type ServicePort interface {
	// Languages
	CreateLanguage(ctx context.Context, input CreateLanguageInput) (*LanguageDTO, error)
	GetLanguage(ctx context.Context, id string) (*LanguageDTO, error)
	GetLanguageByCode(ctx context.Context, code string) (*LanguageDTO, error)
	ListLanguages(ctx context.Context) ([]LanguageDTO, error)
	ListActiveLanguages(ctx context.Context) ([]LanguageDTO, error)
	DeleteLanguage(ctx context.Context, id string) error

	// Translations
	UpsertTranslation(ctx context.Context, input UpsertTranslationInput) (*TranslationDTO, error)
	DeleteTranslation(ctx context.Context, languageCode, key string) error
	ListTranslationsByLanguage(ctx context.Context, languageCode string) ([]TranslationDTO, error)
	ListTranslationsByPrefix(ctx context.Context, languageCode, prefix string) ([]TranslationDTO, error)

	// Translate (the main API used by other modules)
	Translate(ctx context.Context, languageCode, key string) (TranslateResult, error)
	BulkTranslate(ctx context.Context, input BulkTranslateInput) (BulkTranslateResult, error)
}

// ===== Mappers =====
func ToLanguageDTO(l domain.Language) LanguageDTO {
	return LanguageDTO{
		ID: l.ID(), Code: l.Code(), Name: l.Name(),
		IsRTL: l.IsRTL(), IsDefault: l.IsDefault(), IsActive: l.IsActive(),
		CreatedAt: l.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: l.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
func ToTranslationDTO(t domain.Translation) TranslationDTO {
	return TranslationDTO{
		ID: t.ID(), LanguageCode: t.LanguageCode(), Key: t.Key(), Value: t.Value(),
		CreatedAt: t.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
func ToLanguageDTOPtr(l domain.Language) *LanguageDTO { dto := ToLanguageDTO(l); return &dto }
func ToTranslationDTOPtr(t domain.Translation) *TranslationDTO { dto := ToTranslationDTO(t); return &dto }
