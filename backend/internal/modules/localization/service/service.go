// Package service: localization service implementation.
package service

import (
	"context"
	"strings"

	"avex-backend/internal/modules/localization/domain"
	"avex-backend/internal/modules/localization/port"
)

type Service struct {
	deps port.Deps
	pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service { return &Service{deps: deps, pool: pool} }

// ===== Languages =====

func (s *Service) CreateLanguage(ctx context.Context, input port.CreateLanguageInput) (*port.LanguageDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	l, err := domain.NewLanguage(id, input.Code, input.Name, input.IsRTL, input.IsDefault, input.IsActive, now)
	if err != nil { return nil, err }
	if err := s.deps.Repos.Languages.Create(ctx, s.pool, l); err != nil { return nil, err }
	return port.ToLanguageDTOPtr(l), nil
}

func (s *Service) GetLanguage(ctx context.Context, id string) (*port.LanguageDTO, error) {
	l, err := s.deps.Repos.Languages.GetByID(ctx, s.pool, id)
	if err != nil { return nil, err }
	return port.ToLanguageDTOPtr(*l), nil
}

func (s *Service) GetLanguageByCode(ctx context.Context, code string) (*port.LanguageDTO, error) {
	l, err := s.deps.Repos.Languages.GetByCode(ctx, s.pool, strings.ToLower(code))
	if err != nil { return nil, err }
	return port.ToLanguageDTOPtr(*l), nil
}

func (s *Service) ListLanguages(ctx context.Context) ([]port.LanguageDTO, error) {
	langs, err := s.deps.Repos.Languages.ListAll(ctx, s.pool)
	if err != nil { return nil, err }
	dtos := make([]port.LanguageDTO, 0, len(langs))
	for _, l := range langs { dtos = append(dtos, port.ToLanguageDTO(l)) }
	return dtos, nil
}

func (s *Service) ListActiveLanguages(ctx context.Context) ([]port.LanguageDTO, error) {
	langs, err := s.deps.Repos.Languages.ListActive(ctx, s.pool)
	if err != nil { return nil, err }
	dtos := make([]port.LanguageDTO, 0, len(langs))
	for _, l := range langs { dtos = append(dtos, port.ToLanguageDTO(l)) }
	return dtos, nil
}

func (s *Service) DeleteLanguage(ctx context.Context, id string) error {
	return s.deps.Repos.Languages.Delete(ctx, s.pool, id)
}

// ===== Translations =====

func (s *Service) UpsertTranslation(ctx context.Context, input port.UpsertTranslationInput) (*port.TranslationDTO, error) {
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()
	t, err := domain.NewTranslation(id, strings.ToLower(input.LanguageCode), input.Key, input.Value, now)
	if err != nil { return nil, err }
	if err := s.deps.Repos.Translations.Upsert(ctx, s.pool, t); err != nil { return nil, err }
	return port.ToTranslationDTOPtr(t), nil
}

func (s *Service) DeleteTranslation(ctx context.Context, languageCode, key string) error {
	return s.deps.Repos.Translations.DeleteByKey(ctx, s.pool, strings.ToLower(languageCode), key)
}

func (s *Service) ListTranslationsByLanguage(ctx context.Context, languageCode string) ([]port.TranslationDTO, error) {
	trans, err := s.deps.Repos.Translations.ListByLanguage(ctx, s.pool, strings.ToLower(languageCode))
	if err != nil { return nil, err }
	dtos := make([]port.TranslationDTO, 0, len(trans))
	for _, t := range trans { dtos = append(dtos, port.ToTranslationDTO(t)) }
	return dtos, nil
}

func (s *Service) ListTranslationsByPrefix(ctx context.Context, languageCode, prefix string) ([]port.TranslationDTO, error) {
	trans, err := s.deps.Repos.Translations.ListByPrefix(ctx, s.pool, strings.ToLower(languageCode), prefix)
	if err != nil { return nil, err }
	dtos := make([]port.TranslationDTO, 0, len(trans))
	for _, t := range trans { dtos = append(dtos, port.ToTranslationDTO(t)) }
	return dtos, nil
}

// ===== Translate =====

// Translate looks up a key in the requested language, with fallback to the
// default language, then to the key itself.
func (s *Service) Translate(ctx context.Context, languageCode, key string) (port.TranslateResult, error) {
	langCode := strings.ToLower(languageCode)

	// 1. Try the requested language
	t, err := s.deps.Repos.Translations.GetByKey(ctx, s.pool, langCode, key)
	if err == nil && t != nil {
		return port.TranslateResult{Key: key, Value: t.Value(), Lang: langCode, Found: true}, nil
	}

	// 2. Fallback to default language (en)
	if langCode != domain.LangEnglish {
		t, err := s.deps.Repos.Translations.GetByKey(ctx, s.pool, domain.LangEnglish, key)
		if err == nil && t != nil {
			return port.TranslateResult{Key: key, Value: t.Value(), Lang: domain.LangEnglish, Found: true}, nil
		}
	}

	// 3. Key not found — return the key itself
	return port.TranslateResult{Key: key, Value: key, Lang: langCode, Found: false}, nil
}

// BulkTranslate looks up multiple keys at once.
func (s *Service) BulkTranslate(ctx context.Context, input port.BulkTranslateInput) (port.BulkTranslateResult, error) {
	langCode := strings.ToLower(input.LanguageCode)
	result := port.BulkTranslateResult{
		LanguageCode: langCode,
		Translations: make(map[string]port.TranslateResult, len(input.Keys)),
	}

	// 1. Bulk fetch from requested language
	requested, err := s.deps.Repos.Translations.BulkGet(ctx, s.pool, langCode, input.Keys)
	if err != nil { return result, err }

	// Determine which keys are missing
	var missingKeys []string
	for _, key := range input.Keys {
		if t, ok := requested[key]; ok {
			result.Translations[key] = port.TranslateResult{Key: key, Value: t.Value(), Lang: langCode, Found: true}
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	// 2. Fallback to default language for missing keys
	if len(missingKeys) > 0 && langCode != domain.LangEnglish {
		defaults, err := s.deps.Repos.Translations.BulkGet(ctx, s.pool, domain.LangEnglish, missingKeys)
		if err == nil {
			for key, t := range defaults {
				result.Translations[key] = port.TranslateResult{Key: key, Value: t.Value(), Lang: domain.LangEnglish, Found: true}
			}
			// Update missing list
			var stillMissing []string
			for _, key := range missingKeys {
				if _, ok := defaults[key]; !ok { stillMissing = append(stillMissing, key) }
			}
			missingKeys = stillMissing
		}
	}

	// 3. Key not found — return the key itself
	for _, key := range missingKeys {
		result.Translations[key] = port.TranslateResult{Key: key, Value: key, Lang: langCode, Found: false}
	}

	return result, nil
}
