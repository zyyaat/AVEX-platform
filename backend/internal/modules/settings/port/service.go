// Package port service: ServicePort + DTOs + events.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/settings/domain"
)

// ===== Events =====
const (
	EventSettingCreated   = "settings.setting.created"
	EventSettingUpdated   = "settings.setting.updated"
	EventSettingDeleted   = "settings.setting.deleted"
	EventFeatureFlagToggled = "settings.feature_flag.toggled"
	EventFeatureFlagCreated = "settings.feature_flag.created"
	EventFeatureFlagDeleted = "settings.feature_flag.deleted"
)
const (
	SettingCreatedEventVersion = 1; SettingCreatedSchemaVersion = 1
	SettingUpdatedEventVersion = 1; SettingUpdatedSchemaVersion = 1
	SettingDeletedEventVersion = 1; SettingDeletedSchemaVersion = 1
	FeatureFlagToggledEventVersion = 1; FeatureFlagToggledSchemaVersion = 1
	FeatureFlagCreatedEventVersion = 1; FeatureFlagCreatedSchemaVersion = 1
	FeatureFlagDeletedEventVersion = 1; FeatureFlagDeletedSchemaVersion = 1
)

type SettingCreatedPayload struct {
	SettingID, Key, Type, Value string; Version int
}
type SettingUpdatedPayload struct {
	SettingID, Key string; OldVersion, NewVersion int; OldValue, NewValue string
}
type SettingDeletedPayload struct {
	SettingID, Key string
}
type FeatureFlagToggledPayload struct {
	FlagID, Name string; Enabled bool
}
type FeatureFlagCreatedPayload struct {
	FlagID, Name string; Enabled bool; TargetType string; RolloutPct int
}
type FeatureFlagDeletedPayload struct {
	FlagID, Name string
}

type EventMetadata struct{ CorrelationID, TraceID string; OccurredAt time.Time }
type EventContext struct{ Actor ActorContext; Metadata EventMetadata }
func BuildEnvelope(eventID, eventType string, eventVersion, schemaVersion int, payload []byte, ec EventContext) EventEnvelope {
	occurredAt := ec.Metadata.OccurredAt
	if occurredAt.IsZero() { occurredAt = time.Now().UTC() }
	return EventEnvelope{
		EventID: eventID, EventType: eventType, EventVersion: eventVersion, SchemaVersion: schemaVersion,
		OccurredAt: occurredAt, Producer: "settings",
		CorrelationID: ec.Metadata.CorrelationID, TraceID: ec.Metadata.TraceID,
		ActorType: ec.Actor.Type, ActorID: ec.Actor.ID, ActorIP: ec.Actor.IP, ActorUA: ec.Actor.UserAgent,
		Payload: payload,
	}
}

// ===== DTOs =====
type CreateSettingInput struct {
	Key, Description, Type, Value string
	IsProtected bool
}
type UpdateSettingInput struct {
	Value, ChangedBy, ChangeNote string
}
type CreateFeatureFlagInput struct {
	Name, Description string
	Enabled bool
	TargetType string
	TargetValue string
	RolloutPct int
}
type UpdateFeatureFlagInput struct {
	Enabled *bool
	TargetType, TargetValue string
	RolloutPct *int
}

type SettingDTO struct {
	ID string `json:"id"`; Key string `json:"key"`; Description string `json:"description,omitempty"`
	Type string `json:"type"`; Value string `json:"value"`; IsProtected bool `json:"is_protected"`
	Version int `json:"version"`; CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
type SettingRevisionDTO struct {
	ID string `json:"id"`; SettingID string `json:"setting_id"`; Version int `json:"version"`
	Value string `json:"value"`; ChangedBy string `json:"changed_by,omitempty"`
	ChangeNote string `json:"change_note,omitempty"`; CreatedAt time.Time `json:"created_at"`
}
type FeatureFlagDTO struct {
	ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`
	Enabled bool `json:"enabled"`; TargetType string `json:"target_type"`; TargetValue string `json:"target_value,omitempty"`
	RolloutPct int `json:"rollout_pct"`; CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
type CheckFlagResult struct {
	Name string `json:"name"`; Enabled bool `json:"enabled"`
}

// ===== ServicePort =====
type ServicePort interface {
	// Settings
	CreateSetting(ctx context.Context, input CreateSettingInput) (*SettingDTO, error)
	GetSetting(ctx context.Context, id string) (*SettingDTO, error)
	GetSettingByKey(ctx context.Context, key string) (*SettingDTO, error)
	UpdateSetting(ctx context.Context, id string, input UpdateSettingInput) (*SettingDTO, error)
	DeleteSetting(ctx context.Context, id string) error
	ListSettings(ctx context.Context, page PageQuery) (Page[SettingDTO], error)
	ListSettingsByType(ctx context.Context, settingType string) ([]SettingDTO, error)

	// Revisions
	ListRevisions(ctx context.Context, settingID string, page PageQuery) (Page[SettingRevisionDTO], error)
	RollbackSetting(ctx context.Context, settingID string, version int, changedBy string) (*SettingDTO, error)

	// Feature Flags
	CreateFeatureFlag(ctx context.Context, input CreateFeatureFlagInput) (*FeatureFlagDTO, error)
	GetFeatureFlag(ctx context.Context, id string) (*FeatureFlagDTO, error)
	GetFeatureFlagByName(ctx context.Context, name string) (*FeatureFlagDTO, error)
	UpdateFeatureFlag(ctx context.Context, id string, input UpdateFeatureFlagInput) (*FeatureFlagDTO, error)
	DeleteFeatureFlag(ctx context.Context, id string) error
	ListFeatureFlags(ctx context.Context, page PageQuery) (Page[FeatureFlagDTO], error)

	// Flag checking (used by other modules)
	IsFeatureEnabled(ctx context.Context, name, userID string, userRoles []string) (CheckFlagResult, error)
}

// ===== Mappers =====
func ToSettingDTO(s domain.Setting) SettingDTO {
	return SettingDTO{ID: s.ID(), Key: s.Key(), Description: s.Description(), Type: string(s.Type()), Value: s.Value(), IsProtected: s.IsProtected(), Version: s.Version(), CreatedAt: s.CreatedAt(), UpdatedAt: s.UpdatedAt()}
}
func ToSettingRevisionDTO(r domain.SettingRevision) SettingRevisionDTO {
	return SettingRevisionDTO{ID: r.ID(), SettingID: r.SettingID(), Version: r.Version(), Value: r.Value(), ChangedBy: r.ChangedBy(), ChangeNote: r.ChangeNote(), CreatedAt: r.CreatedAt()}
}
func ToFeatureFlagDTO(f domain.FeatureFlag) FeatureFlagDTO {
	return FeatureFlagDTO{ID: f.ID(), Name: f.Name(), Description: f.Description(), Enabled: f.Enabled(), TargetType: string(f.TargetType()), TargetValue: f.TargetValue(), RolloutPct: f.RolloutPct(), CreatedAt: f.CreatedAt(), UpdatedAt: f.UpdatedAt()}
}
func ToSettingDTOPtr(s domain.Setting) *SettingDTO { dto := ToSettingDTO(s); return &dto }
func ToFeatureFlagDTOPtr(f domain.FeatureFlag) *FeatureFlagDTO { dto := ToFeatureFlagDTO(f); return &dto }
