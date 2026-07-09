// Package domain: Localization module types + errors.
//
// The Localization module provides multi-language support:
//   - Languages: registered languages (en, ar, fr, etc.)
//   - Translations: key-value pairs per language
//   - Fallback: if a key is missing in the requested language, fall back
//     to the default language (en), then to the key itself.
//
// Translation keys use a hierarchical naming convention:
//   "module.section.key"
// Examples:
//   "orders.status.pending"
//   "orders.status.confirmed"
//   "notifications.order_created.title"
//   "common.button.submit"
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ===== Errors =====

var ErrLanguageNotFound = errors.New("language not found")
var ErrLanguageAlreadyExists = errors.New("language already exists")
var ErrTranslationNotFound = errors.New("translation not found")
var ErrTranslationAlreadyExists = errors.New("translation already exists")
var ErrCannotDeleteDefaultLanguage = errors.New("cannot delete default language")
var ErrInvalidLanguageCode = errors.New("invalid language code (must be 2-letter ISO 639-1)")

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyKey = errors.New("translation key is required")
var ErrEmptyValue = errors.New("translation value is required")

// ===== Language =====

// Language represents a supported language.
type Language struct {
	id        string
	code      string // ISO 639-1, e.g. "en", "ar", "fr"
	name      string // display name, e.g. "English", "العربية"
	isRTL     bool   // right-to-left (Arabic, Hebrew)
	isDefault bool   // the default fallback language
	isActive  bool   // available for use
	createdAt time.Time
	updatedAt time.Time
}

// NewLanguage creates a new Language with validation.
func NewLanguage(id, code, name string, isRTL, isDefault, isActive bool, now time.Time) (Language, error) {
	if id == "" {
		return Language{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if len(code) != 2 {
		return Language{}, fmt.Errorf("%w: %q (must be 2 characters)", ErrInvalidLanguageCode, code)
	}
	if name == "" {
		return Language{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	code = strings.ToLower(code)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Language{
		id: id, code: code, name: name, isRTL: isRTL,
		isDefault: isDefault, isActive: isActive,
		createdAt: now, updatedAt: now,
	}, nil
}

func RehydrateLanguage(id, code, name string, isRTL, isDefault, isActive bool, createdAt, updatedAt time.Time) Language {
	return Language{id: id, code: strings.ToLower(code), name: name, isRTL: isRTL, isDefault: isDefault, isActive: isActive, createdAt: createdAt, updatedAt: updatedAt}
}

func (l Language) ID() string         { return l.id }
func (l Language) Code() string       { return l.code }
func (l Language) Name() string       { return l.name }
func (l Language) IsRTL() bool        { return l.isRTL }
func (l Language) IsDefault() bool    { return l.isDefault }
func (l Language) IsActive() bool     { return l.isActive }
func (l Language) CreatedAt() time.Time { return l.createdAt }
func (l Language) UpdatedAt() time.Time { return l.updatedAt }

// SetActive toggles the active flag.
func (l Language) SetActive(active bool, now time.Time) Language {
	l.isActive = active
	l.updatedAt = now
	return l
}

// ===== Translation =====

// Translation represents a single translated string for a key in a language.
type Translation struct {
	id         string
	languageCode string
	key        string  // e.g. "orders.status.pending"
	value      string  // the translated text
	createdAt  time.Time
	updatedAt  time.Time
}

// NewTranslation creates a new Translation with validation.
func NewTranslation(id, languageCode, key, value string, now time.Time) (Translation, error) {
	if id == "" {
		return Translation{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if languageCode == "" {
		return Translation{}, fmt.Errorf("%w: language code is required", ErrInvalidInput)
	}
	if key == "" {
		return Translation{}, ErrEmptyKey
	}
	if value == "" {
		return Translation{}, ErrEmptyValue
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Translation{
		id: id, languageCode: strings.ToLower(languageCode),
		key: key, value: value,
		createdAt: now, updatedAt: now,
	}, nil
}

func RehydrateTranslation(id, languageCode, key, value string, createdAt, updatedAt time.Time) Translation {
	return Translation{id: id, languageCode: strings.ToLower(languageCode), key: key, value: value, createdAt: createdAt, updatedAt: updatedAt}
}

func (t Translation) ID() string           { return t.id }
func (t Translation) LanguageCode() string { return t.languageCode }
func (t Translation) Key() string          { return t.key }
func (t Translation) Value() string        { return t.value }
func (t Translation) CreatedAt() time.Time { return t.createdAt }
func (t Translation) UpdatedAt() time.Time { return t.updatedAt }

// SetValue updates the value.
func (t Translation) SetValue(value string, now time.Time) (Translation, error) {
	if value == "" {
		return t, ErrEmptyValue
	}
	t.value = value
	t.updatedAt = now
	return t, nil
}

// ===== Default language codes =====

const (
	LangEnglish = "en"
	LangArabic  = "ar"
	LangFrench  = "fr"
)
