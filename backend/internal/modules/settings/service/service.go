// Package service: settings service implementation.
package service

import (
        "context"
        "fmt"

        "avex-backend/internal/modules/settings/domain"
        "avex-backend/internal/modules/settings/events"
        "avex-backend/internal/modules/settings/port"
)

type Service struct {
        deps port.Deps
        pool port.Executor
}

var _ port.ServicePort = (*Service)(nil)

func New(deps port.Deps, pool port.Executor) *Service { return &Service{deps: deps, pool: pool} }

func (s *Service) eventContext(_ context.Context, actor port.ActorContext) port.EventContext {
        return port.EventContext{Actor: actor, Metadata: port.EventMetadata{OccurredAt: s.deps.Clock.Now()}}
}

// ===== Settings =====

func (s *Service) CreateSetting(ctx context.Context, input port.CreateSettingInput) (*port.SettingDTO, error) {
        now := s.deps.Clock.Now()
        id := s.deps.IDGenerator.NewID()
        setting, err := domain.NewSetting(id, input.Key, input.Description, domain.SettingType(input.Type), input.Value, input.IsProtected, now)
        if err != nil { return nil, err }
        err = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                if err := s.deps.Repos.Settings.Create(ctx, exec, setting); err != nil { return err }
                // Create initial revision
                revID := s.deps.IDGenerator.NewID()
                rev, _ := domain.NewSettingRevision(revID, setting.ID(), 1, setting.Value(), "system", "Initial value", now)
                if err := s.deps.Repos.Revisions.Create(ctx, exec, rev); err != nil { return err }
                // Publish event
                ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
                env, _ := events.SettingCreatedEnvelope(port.SettingCreatedPayload{SettingID: setting.ID(), Key: setting.Key(), Type: string(setting.Type()), Value: setting.Value(), Version: setting.Version()}, ec)
                return s.deps.EventPublisher.Publish(ctx, exec, env)
        })
        if err != nil { return nil, err }
        return port.ToSettingDTOPtr(setting), nil
}

func (s *Service) GetSetting(ctx context.Context, id string) (*port.SettingDTO, error) {
        setting, err := s.deps.Repos.Settings.GetByID(ctx, s.pool, id)
        if err != nil { return nil, err }
        return port.ToSettingDTOPtr(*setting), nil
}

func (s *Service) GetSettingByKey(ctx context.Context, key string) (*port.SettingDTO, error) {
        setting, err := s.deps.Repos.Settings.GetByKey(ctx, s.pool, key)
        if err != nil { return nil, err }
        return port.ToSettingDTOPtr(*setting), nil
}

func (s *Service) UpdateSetting(ctx context.Context, id string, input port.UpdateSettingInput) (*port.SettingDTO, error) {
        var updated *domain.Setting
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                setting, err := s.deps.Repos.Settings.GetByID(ctx, exec, id)
                if err != nil { return err }
                oldVersion := setting.Version()
                oldValue := setting.Value()
                newSetting, err := setting.SetValue(input.Value, s.deps.Clock.Now())
                if err != nil { return err }
                if err := s.deps.Repos.Settings.Update(ctx, exec, newSetting); err != nil { return err }
                // Create revision
                revID := s.deps.IDGenerator.NewID()
                rev, _ := domain.NewSettingRevision(revID, newSetting.ID(), newSetting.Version(), newSetting.Value(), input.ChangedBy, input.ChangeNote, s.deps.Clock.Now())
                if err := s.deps.Repos.Revisions.Create(ctx, exec, rev); err != nil { return err }
                // Publish event
                ec := s.eventContext(ctx, port.ActorContext{Type: "admin", ID: input.ChangedBy})
                env, _ := events.SettingUpdatedEnvelope(port.SettingUpdatedPayload{
                        SettingID: newSetting.ID(), Key: newSetting.Key(),
                        OldVersion: oldVersion, NewVersion: newSetting.Version(),
                        OldValue: oldValue, NewValue: newSetting.Value(),
                }, ec)
                if err := s.deps.EventPublisher.Publish(ctx, exec, env); err != nil { return err }
                updated = &newSetting
                return nil
        })
        if err != nil { return nil, err }
        return port.ToSettingDTOPtr(*updated), nil
}

func (s *Service) DeleteSetting(ctx context.Context, id string) error {
        setting, err := s.deps.Repos.Settings.GetByID(ctx, s.pool, id)
        if err != nil { return err }
        if setting.IsProtected() { return domain.ErrCannotDeleteProtected }
        if err := s.deps.Repos.Settings.Delete(ctx, s.pool, id); err != nil { return err }
        // Publish event
        ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
        env, _ := events.SettingDeletedEnvelope(port.SettingDeletedPayload{SettingID: id, Key: setting.Key()}, ec)
        _ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
        return nil
}

func (s *Service) ListSettings(ctx context.Context, page port.PageQuery) (port.Page[port.SettingDTO], error) {
        result, err := s.deps.Repos.Settings.ListAll(ctx, s.pool, page)
        if err != nil { return port.Page[port.SettingDTO]{}, err }
        dtos := make([]port.SettingDTO, 0, len(result.Items))
        for _, s := range result.Items { dtos = append(dtos, port.ToSettingDTO(s)) }
        return port.Page[port.SettingDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

func (s *Service) ListSettingsByType(ctx context.Context, settingType string) ([]port.SettingDTO, error) {
        settings, err := s.deps.Repos.Settings.ListByType(ctx, s.pool, settingType)
        if err != nil { return nil, err }
        dtos := make([]port.SettingDTO, 0, len(settings))
        for _, s := range settings { dtos = append(dtos, port.ToSettingDTO(s)) }
        return dtos, nil
}

// ===== Revisions =====

func (s *Service) ListRevisions(ctx context.Context, settingID string, page port.PageQuery) (port.Page[port.SettingRevisionDTO], error) {
        result, err := s.deps.Repos.Revisions.ListBySetting(ctx, s.pool, settingID, page)
        if err != nil { return port.Page[port.SettingRevisionDTO]{}, err }
        dtos := make([]port.SettingRevisionDTO, 0, len(result.Items))
        for _, r := range result.Items { dtos = append(dtos, port.ToSettingRevisionDTO(r)) }
        return port.Page[port.SettingRevisionDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

func (s *Service) RollbackSetting(ctx context.Context, settingID string, version int, changedBy string) (*port.SettingDTO, error) {
        var updated *domain.Setting
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                setting, err := s.deps.Repos.Settings.GetByID(ctx, exec, settingID)
                if err != nil { return err }
                rev, err := s.deps.Repos.Revisions.GetBySettingAndVersion(ctx, exec, settingID, version)
                if err != nil { return err }
                // Roll back to the revision's value
                newSetting, err := setting.SetValue(rev.Value(), s.deps.Clock.Now())
                if err != nil { return err }
                if err := s.deps.Repos.Settings.Update(ctx, exec, newSetting); err != nil { return err }
                // Create a new revision noting the rollback
                revID := s.deps.IDGenerator.NewID()
                rollbackRev, _ := domain.NewSettingRevision(revID, newSetting.ID(), newSetting.Version(), newSetting.Value(), changedBy,
                        fmt.Sprintf("Rolled back to version %d", version), s.deps.Clock.Now())
                if err := s.deps.Repos.Revisions.Create(ctx, exec, rollbackRev); err != nil { return err }
                updated = &newSetting
                return nil
        })
        if err != nil { return nil, err }
        return port.ToSettingDTOPtr(*updated), nil
}

// ===== Feature Flags =====

func (s *Service) CreateFeatureFlag(ctx context.Context, input port.CreateFeatureFlagInput) (*port.FeatureFlagDTO, error) {
        now := s.deps.Clock.Now()
        id := s.deps.IDGenerator.NewID()
        flag, err := domain.NewFeatureFlag(id, input.Name, input.Description, input.Enabled, domain.TargetType(input.TargetType), input.TargetValue, input.RolloutPct, now)
        if err != nil { return nil, err }
        if err := s.deps.Repos.Flags.Create(ctx, s.pool, flag); err != nil { return nil, err }
        // Publish event
        ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
        env, _ := events.FeatureFlagCreatedEnvelope(port.FeatureFlagCreatedPayload{FlagID: flag.ID(), Name: flag.Name(), Enabled: flag.Enabled(), TargetType: string(flag.TargetType()), RolloutPct: flag.RolloutPct()}, ec)
        _ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
        return port.ToFeatureFlagDTOPtr(flag), nil
}

func (s *Service) GetFeatureFlag(ctx context.Context, id string) (*port.FeatureFlagDTO, error) {
        f, err := s.deps.Repos.Flags.GetByID(ctx, s.pool, id)
        if err != nil { return nil, err }
        return port.ToFeatureFlagDTOPtr(*f), nil
}

func (s *Service) GetFeatureFlagByName(ctx context.Context, name string) (*port.FeatureFlagDTO, error) {
        f, err := s.deps.Repos.Flags.GetByName(ctx, s.pool, name)
        if err != nil { return nil, err }
        return port.ToFeatureFlagDTOPtr(*f), nil
}

func (s *Service) UpdateFeatureFlag(ctx context.Context, id string, input port.UpdateFeatureFlagInput) (*port.FeatureFlagDTO, error) {
        var updated *domain.FeatureFlag
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                flag, err := s.deps.Repos.Flags.GetByID(ctx, exec, id)
                if err != nil { return err }
                now := s.deps.Clock.Now()
                updatedFlag := *flag
                if input.Enabled != nil {
                        updatedFlag = updatedFlag.SetEnabled(*input.Enabled, now)
                }
                if input.TargetType != "" || input.TargetValue != "" || input.RolloutPct != nil {
                        tt := domain.TargetType(input.TargetType)
                        if tt == "" { tt = updatedFlag.TargetType() }
                        tv := input.TargetValue
                        if tv == "" { tv = updatedFlag.TargetValue() }
                        rp := updatedFlag.RolloutPct()
                        if input.RolloutPct != nil { rp = *input.RolloutPct }
                        updatedFlag, err = updatedFlag.SetRollout(tt, tv, rp, now)
                        if err != nil { return err }
                }
                if err := s.deps.Repos.Flags.Update(ctx, exec, updatedFlag); err != nil { return err }
                // Publish toggle event if enabled changed
                if input.Enabled != nil && *input.Enabled != flag.Enabled() {
                        ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
                        env, _ := events.FeatureFlagToggledEnvelope(port.FeatureFlagToggledPayload{FlagID: updatedFlag.ID(), Name: updatedFlag.Name(), Enabled: updatedFlag.Enabled()}, ec)
                        _ = s.deps.EventPublisher.Publish(ctx, exec, env)
                }
                updated = &updatedFlag
                return nil
        })
        if err != nil { return nil, err }
        return port.ToFeatureFlagDTOPtr(*updated), nil
}

func (s *Service) DeleteFeatureFlag(ctx context.Context, id string) error {
        flag, err := s.deps.Repos.Flags.GetByID(ctx, s.pool, id)
        if err != nil { return err }
        if err := s.deps.Repos.Flags.Delete(ctx, s.pool, id); err != nil { return err }
        ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
        env, _ := events.FeatureFlagDeletedEnvelope(port.FeatureFlagDeletedPayload{FlagID: id, Name: flag.Name()}, ec)
        _ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error { return s.deps.EventPublisher.Publish(ctx, exec, env) })
        return nil
}

func (s *Service) ListFeatureFlags(ctx context.Context, page port.PageQuery) (port.Page[port.FeatureFlagDTO], error) {
        result, err := s.deps.Repos.Flags.ListAll(ctx, s.pool, page)
        if err != nil { return port.Page[port.FeatureFlagDTO]{}, err }
        dtos := make([]port.FeatureFlagDTO, 0, len(result.Items))
        for _, f := range result.Items { dtos = append(dtos, port.ToFeatureFlagDTO(f)) }
        return port.Page[port.FeatureFlagDTO]{Items: dtos, Total: result.Total, Limit: result.Limit, Offset: result.Offset}, nil
}

// ===== Flag Checking =====

func (s *Service) IsFeatureEnabled(ctx context.Context, name, userID string, userRoles []string) (port.CheckFlagResult, error) {
        flag, err := s.deps.Repos.Flags.GetByName(ctx, s.pool, name)
        if err != nil {
                if err == domain.ErrFeatureFlagNotFound {
                        return port.CheckFlagResult{Name: name, Enabled: false}, nil
                }
                return port.CheckFlagResult{}, err
        }
        return port.CheckFlagResult{Name: name, Enabled: flag.IsEnabledFor(userID, userRoles)}, nil
}

// IsMaintenanceMode checks if the "app.maintenance_mode" setting is true.
func (s *Service) IsMaintenanceMode(ctx context.Context) bool {
        setting, err := s.deps.Repos.Settings.GetByKey(ctx, s.pool, "app.maintenance_mode")
        if err != nil {
                return false
        }
        return setting.TypedValue().(bool)
}
