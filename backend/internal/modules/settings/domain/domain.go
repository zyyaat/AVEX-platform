// Package domain: Settings module errors + types.
//
// Settings are versioned key-value pairs. Every change creates a new revision
// so we can roll back to any previous version. The current value is the latest
// revision.
//
// Feature flags are boolean toggles with optional rollout percentage (0-100)
// and optional target audience (users/roles/percent).
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ===== Errors =====

var ErrSettingNotFound = errors.New("setting not found")
var ErrSettingAlreadyExists = errors.New("setting already exists")
var ErrRevisionNotFound = errors.New("revision not found")
var ErrInvalidSettingType = errors.New("invalid setting type")
var ErrInvalidSettingValue = errors.New("invalid setting value for this type")
var ErrCannotDeleteProtected = errors.New("cannot delete protected setting")

var ErrFeatureFlagNotFound = errors.New("feature flag not found")
var ErrFeatureFlagAlreadyExists = errors.New("feature flag already exists")
var ErrInvalidRolloutPercentage = errors.New("rollout percentage must be 0-100")
var ErrInvalidTargetType = errors.New("invalid target type")

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyKey = errors.New("setting key is required")
var ErrEmptyName = errors.New("feature flag name is required")

// SettingType enumerates the value types a setting can hold.
type SettingType string

const (
	SettingTypeString  SettingType = "string"
	SettingTypeInt     SettingType = "int"
	SettingTypeFloat   SettingType = "float"
	SettingTypeBool    SettingType = "bool"
	SettingTypeJSON    SettingType = "json"
)

func (t SettingType) IsValid() bool {
	switch t {
	case SettingTypeString, SettingTypeInt, SettingTypeFloat, SettingTypeBool, SettingTypeJSON:
		return true
	}
	return false
}

// Setting is a versioned key-value configuration entry.
type Setting struct {
	id          string
	key         string // unique, e.g. "delivery.default_radius_km"
	description string
	settingType SettingType
	value       string // stored as string; interpreted based on type
	isProtected bool   // protected settings cannot be deleted
	updatedAt   time.Time
	createdAt   time.Time
	version     int // current version number (increments on each change)
}

// NewSetting creates a new Setting with validation.
func NewSetting(
	id, key, description string,
	settingType SettingType,
	value string,
	isProtected bool,
	now time.Time,
) (Setting, error) {
	if id == "" {
		return Setting{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if key == "" {
		return Setting{}, ErrEmptyKey
	}
	if !settingType.IsValid() {
		return Setting{}, fmt.Errorf("%w: %s", ErrInvalidSettingType, settingType)
	}
	if err := ValidateValue(settingType, value); err != nil {
		return Setting{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Setting{
		id:          id,
		key:         key,
		description: description,
		settingType: settingType,
		value:       value,
		isProtected: isProtected,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func RehydrateSetting(
	id, key, description string,
	settingType SettingType,
	value string,
	isProtected bool,
	version int,
	createdAt, updatedAt time.Time,
) Setting {
	return Setting{id: id, key: key, description: description, settingType: settingType, value: value, isProtected: isProtected, version: version, createdAt: createdAt, updatedAt: updatedAt}
}

func (s Setting) ID() string            { return s.id }
func (s Setting) Key() string           { return s.key }
func (s Setting) Description() string   { return s.description }
func (s Setting) Type() SettingType     { return s.settingType }
func (s Setting) Value() string         { return s.value }
func (s Setting) IsProtected() bool     { return s.isProtected }
func (s Setting) Version() int          { return s.version }
func (s Setting) CreatedAt() time.Time  { return s.createdAt }
func (s Setting) UpdatedAt() time.Time  { return s.updatedAt }

// SetValue updates the value + bumps version.
func (s Setting) SetValue(value string, now time.Time) (Setting, error) {
	if err := ValidateValue(s.settingType, value); err != nil {
		return s, err
	}
	s.value = value
	s.version++
	s.updatedAt = now
	return s, nil
}

// ValidateValue checks that a string value is valid for the given type.
func ValidateValue(t SettingType, value string) error {
	switch t {
	case SettingTypeString:
		return nil // any string is valid
	case SettingTypeInt:
		if value == "" {
			return fmt.Errorf("%w: empty int value", ErrInvalidSettingValue)
		}
		// Simple integer parse check
		var n int64
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return fmt.Errorf("%w: %q is not an int", ErrInvalidSettingValue, value)
		}
		return nil
	case SettingTypeFloat:
		if value == "" {
			return fmt.Errorf("%w: empty float value", ErrInvalidSettingValue)
		}
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err != nil {
			return fmt.Errorf("%w: %q is not a float", ErrInvalidSettingValue, value)
		}
		return nil
	case SettingTypeBool:
		if value != "true" && value != "false" {
			return fmt.Errorf("%w: %q is not a bool (must be 'true' or 'false')", ErrInvalidSettingValue, value)
		}
		return nil
	case SettingTypeJSON:
		if value == "" {
			return nil
		}
		var v any
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return fmt.Errorf("%w: %q is not valid JSON", ErrInvalidSettingValue, value)
		}
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidSettingType, t)
}

// TypedValue returns the value converted to the appropriate Go type.
func (s Setting) TypedValue() any {
	switch s.settingType {
	case SettingTypeInt:
		var n int64
		_, _ = fmt.Sscanf(s.value, "%d", &n)
		return n
	case SettingTypeFloat:
		var f float64
		_, _ = fmt.Sscanf(s.value, "%f", &f)
		return f
	case SettingTypeBool:
		return s.value == "true"
	case SettingTypeJSON:
		var v any
		_ = json.Unmarshal([]byte(s.value), &v)
		return v
	}
	return s.value
}

// ===== SettingRevision =====

// SettingRevision is a historical snapshot of a setting's value.
// Each SetValue creates a new revision for audit + rollback.
type SettingRevision struct {
	id         string
	settingID  string
	version    int
	value      string
	changedBy  string
	changeNote string
	createdAt  time.Time
}

func NewSettingRevision(id, settingID string, version int, value, changedBy, note string, now time.Time) (SettingRevision, error) {
	if id == "" {
		return SettingRevision{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if settingID == "" {
		return SettingRevision{}, fmt.Errorf("%w: setting id is required", ErrInvalidInput)
	}
	if version < 1 {
		version = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return SettingRevision{id: id, settingID: settingID, version: version, value: value, changedBy: changedBy, changeNote: note, createdAt: now}, nil
}

func RehydrateSettingRevision(id, settingID string, version int, value, changedBy, note string, createdAt time.Time) SettingRevision {
	return SettingRevision{id: id, settingID: settingID, version: version, value: value, changedBy: changedBy, changeNote: note, createdAt: createdAt}
}

func (r SettingRevision) ID() string        { return r.id }
func (r SettingRevision) SettingID() string { return r.settingID }
func (r SettingRevision) Version() int      { return r.version }
func (r SettingRevision) Value() string     { return r.value }
func (r SettingRevision) ChangedBy() string { return r.changedBy }
func (r SettingRevision) ChangeNote() string { return r.changeNote }
func (r SettingRevision) CreatedAt() time.Time { return r.createdAt }

// ===== FeatureFlag =====

// TargetType enumerates who a feature flag applies to.
type TargetType string

const (
	TargetAll      TargetType = "all"
	TargetUsers    TargetType = "users"
	TargetRoles    TargetType = "roles"
	TargetPercent  TargetType = "percent"
)

func (t TargetType) IsValid() bool {
	switch t {
	case TargetAll, TargetUsers, TargetRoles, TargetPercent:
		return true
	}
	return false
}

// FeatureFlag is a boolean toggle with optional rollout rules.
type FeatureFlag struct {
	id          string
	name        string // unique, e.g. "new_checkout_ui"
	description string
	enabled     bool
	targetType  TargetType
	targetValue string // comma-separated user IDs, role names, or percentage number
	rolloutPct  int    // 0-100 (used when targetType = percent)
	createdAt   time.Time
	updatedAt   time.Time
}

func NewFeatureFlag(
	id, name, description string,
	enabled bool,
	targetType TargetType,
	targetValue string,
	rolloutPct int,
	now time.Time,
) (FeatureFlag, error) {
	if id == "" {
		return FeatureFlag{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if name == "" {
		return FeatureFlag{}, ErrEmptyName
	}
	if !targetType.IsValid() {
		return FeatureFlag{}, fmt.Errorf("%w: %s", ErrInvalidTargetType, targetType)
	}
	if rolloutPct < 0 || rolloutPct > 100 {
		return FeatureFlag{}, ErrInvalidRolloutPercentage
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return FeatureFlag{
		id: id, name: name, description: description, enabled: enabled,
		targetType: targetType, targetValue: targetValue, rolloutPct: rolloutPct,
		createdAt: now, updatedAt: now,
	}, nil
}

func RehydrateFeatureFlag(
	id, name, description string,
	enabled bool,
	targetType TargetType,
	targetValue string,
	rolloutPct int,
	createdAt, updatedAt time.Time,
) FeatureFlag {
	return FeatureFlag{id: id, name: name, description: description, enabled: enabled, targetType: targetType, targetValue: targetValue, rolloutPct: rolloutPct, createdAt: createdAt, updatedAt: updatedAt}
}

func (f FeatureFlag) ID() string          { return f.id }
func (f FeatureFlag) Name() string        { return f.name }
func (f FeatureFlag) Description() string { return f.description }
func (f FeatureFlag) Enabled() bool       { return f.enabled }
func (f FeatureFlag) TargetType() TargetType { return f.targetType }
func (f FeatureFlag) TargetValue() string { return f.targetValue }
func (f FeatureFlag) RolloutPct() int     { return f.rolloutPct }
func (f FeatureFlag) CreatedAt() time.Time { return f.createdAt }
func (f FeatureFlag) UpdatedAt() time.Time { return f.updatedAt }

// SetEnabled toggles the flag.
func (f FeatureFlag) SetEnabled(enabled bool, now time.Time) FeatureFlag {
	f.enabled = enabled
	f.updatedAt = now
	return f
}

// SetRollout updates the target type + value + rollout percentage.
func (f FeatureFlag) SetRollout(targetType TargetType, targetValue string, rolloutPct int, now time.Time) (FeatureFlag, error) {
	if !targetType.IsValid() {
		return f, fmt.Errorf("%w: %s", ErrInvalidTargetType, targetType)
	}
	if rolloutPct < 0 || rolloutPct > 100 {
		return f, ErrInvalidRolloutPercentage
	}
	f.targetType = targetType
	f.targetValue = targetValue
	f.rolloutPct = rolloutPct
	f.updatedAt = now
	return f, nil
}

// IsEnabledFor checks if the flag is enabled for a given user.
// userID is used for TargetUsers and TargetPercent (hash-based).
// userRoles is used for TargetRoles.
func (f FeatureFlag) IsEnabledFor(userID string, userRoles []string) bool {
	if !f.enabled {
		return false
	}
	switch f.targetType {
	case TargetAll:
		return true
	case TargetUsers:
		// targetValue is comma-separated user IDs
		for _, uid := range splitCSV(f.targetValue) {
			if uid == userID {
				return true
			}
		}
		return false
	case TargetRoles:
		// targetValue is comma-separated role names
		targetRoles := splitCSV(f.targetValue)
		for _, targetRole := range targetRoles {
			for _, userRole := range userRoles {
				if targetRole == userRole {
					return true
				}
			}
		}
		return false
	case TargetPercent:
		if f.rolloutPct >= 100 {
			return true
		}
		if f.rolloutPct <= 0 {
			return false
		}
		// Hash the userID to a 0-99 bucket
		bucket := hashToBucket(userID)
		return bucket < f.rolloutPct
	}
	return false
}

// splitCSV splits a comma-separated string, trimming whitespace.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				// trim spaces
				seg := s[start:i]
				for len(seg) > 0 && (seg[0] == ' ' || seg[0] == '\t') {
					seg = seg[1:]
				}
				for len(seg) > 0 && (seg[len(seg)-1] == ' ' || seg[len(seg)-1] == '\t') {
					seg = seg[:len(seg)-1]
				}
				if seg != "" {
					result = append(result, seg)
				}
			}
			start = i + 1
		}
	}
	return result
}

// hashToBucket hashes a string to a 0-99 bucket using FNV-1a.
func hashToBucket(s string) int {
	const (
		offsetBasis uint32 = 2166136261
		prime       uint32 = 16777619
	)
	hash := offsetBasis
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= prime
	}
	return int(hash % 100)
}
