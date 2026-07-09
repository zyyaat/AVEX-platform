// Package domain tests: Setting + FeatureFlag.
package domain

import (
        "testing"
        "time"
)

func testNowSettings() time.Time {
        t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
        return t
}

// ===== Setting Tests =====

func TestNewSetting(t *testing.T) {
        now := testNowSettings()
        tests := []struct {
                name        string
                id          string
                key         string
                settingType SettingType
                value       string
                wantErr     error
        }{
                {"valid string", "s1", "app.name", SettingTypeString, "AVEX", nil},
                {"valid int", "s2", "delivery.radius_km", SettingTypeInt, "5", nil},
                {"valid float", "s3", "delivery.fee", SettingTypeFloat, "3.99", nil},
                {"valid bool", "s4", "app.maintenance", SettingTypeBool, "false", nil},
                {"valid json", "s5", "app.config", SettingTypeJSON, `{"key":"value"}`, nil},
                {"valid empty json", "s6", "app.config2", SettingTypeJSON, "", nil},
                {"empty id", "", "x", SettingTypeString, "v", ErrInvalidID},
                {"empty key", "s7", "", SettingTypeString, "v", ErrEmptyKey},
                {"invalid type", "s8", "x", SettingType("bogus"), "v", ErrInvalidSettingType},
                {"invalid int value", "s9", "x", SettingTypeInt, "not-a-number", ErrInvalidSettingValue},
                {"invalid float value", "s10", "x", SettingTypeFloat, "abc", ErrInvalidSettingValue},
                {"invalid bool value", "s11", "x", SettingTypeBool, "yes", ErrInvalidSettingValue},
                {"invalid json value", "s12", "x", SettingTypeJSON, "{invalid", ErrInvalidSettingValue},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        _, err := NewSetting(tt.id, tt.key, "desc", tt.settingType, tt.value, false, now)
                        if tt.wantErr != nil {
                                if err == nil || !errIs(err, tt.wantErr) {
                                        t.Fatalf("expected %v, got %v", tt.wantErr, err)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                })
        }
}

func TestSettingSetValue(t *testing.T) {
        now := testNowSettings()
        s, _ := NewSetting("s1", "delivery.radius", "", SettingTypeInt, "5", false, now)
        if s.Version() != 1 {
                t.Fatalf("expected version 1, got %d", s.Version())
        }

        // Update value
        updated, err := s.SetValue("10", now)
        if err != nil {
                t.Fatalf("set value: %v", err)
        }
        if updated.Value() != "10" {
                t.Errorf("expected '10', got %q", updated.Value())
        }
        if updated.Version() != 2 {
                t.Errorf("expected version 2, got %d", updated.Version())
        }

        // Invalid value
        _, err = updated.SetValue("not-a-number", now)
        if !errIs(err, ErrInvalidSettingValue) {
                t.Fatalf("expected ErrInvalidSettingValue, got %v", err)
        }
}

func TestSettingTypedValue(t *testing.T) {
        now := testNowSettings()

        // Int
        sInt, _ := NewSetting("s1", "x", "", SettingTypeInt, "42", false, now)
        if v := sInt.TypedValue().(int64); v != 42 {
                t.Errorf("expected 42, got %v", v)
        }

        // Float
        sFloat, _ := NewSetting("s2", "x", "", SettingTypeFloat, "3.14", false, now)
        if v := sFloat.TypedValue().(float64); v != 3.14 {
                t.Errorf("expected 3.14, got %v", v)
        }

        // Bool
        sBool, _ := NewSetting("s3", "x", "", SettingTypeBool, "true", false, now)
        if v := sBool.TypedValue().(bool); !v {
                t.Errorf("expected true, got %v", v)
        }

        // String
        sStr, _ := NewSetting("s4", "x", "", SettingTypeString, "hello", false, now)
        if v := sStr.TypedValue().(string); v != "hello" {
                t.Errorf("expected 'hello', got %v", v)
        }

        // JSON
        sJSON, _ := NewSetting("s5", "x", "", SettingTypeJSON, `{"a":1}`, false, now)
        v := sJSON.TypedValue()
        m, ok := v.(map[string]any)
        if !ok {
                t.Errorf("expected map, got %T", v)
        }
        if m["a"].(float64) != 1 {
                t.Errorf("expected a=1, got %v", m["a"])
        }
}

// ===== FeatureFlag Tests =====

func TestNewFeatureFlag(t *testing.T) {
        now := testNowSettings()
        tests := []struct {
                name        string
                id          string
                flagName    string
                enabled     bool
                targetType  TargetType
                rolloutPct  int
                wantErr     error
        }{
                {"valid all", "f1", "new_ui", true, TargetAll, 0, nil},
                {"valid percent", "f2", "beta_feature", true, TargetPercent, 50, nil},
                {"valid users", "f3", "admin_only", true, TargetUsers, 0, nil},
                {"valid roles", "f4", "agent_only", true, TargetRoles, 0, nil},
                {"empty id", "", "x", true, TargetAll, 0, ErrInvalidID},
                {"empty name", "f5", "", true, TargetAll, 0, ErrEmptyName},
                {"invalid target type", "f6", "x", true, TargetType("bogus"), 0, ErrInvalidTargetType},
                {"negative rollout", "f7", "x", true, TargetPercent, -1, ErrInvalidRolloutPercentage},
                {"rollout > 100", "f8", "x", true, TargetPercent, 101, ErrInvalidRolloutPercentage},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        _, err := NewFeatureFlag(tt.id, tt.flagName, "desc", tt.enabled, tt.targetType, "", tt.rolloutPct, now)
                        if tt.wantErr != nil {
                                if err == nil || !errIs(err, tt.wantErr) {
                                        t.Fatalf("expected %v, got %v", tt.wantErr, err)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                })
        }
}

func TestFeatureFlagIsEnabledFor(t *testing.T) {
        now := testNowSettings()

        // Disabled flag
        f, _ := NewFeatureFlag("f1", "x", "", false, TargetAll, "", 0, now)
        if f.IsEnabledFor("u1", nil) {
                t.Error("expected false for disabled flag")
        }

        // TargetAll
        f, _ = NewFeatureFlag("f2", "x", "", true, TargetAll, "", 0, now)
        if !f.IsEnabledFor("u1", nil) {
                t.Error("expected true for TargetAll")
        }

        // TargetUsers — user in list
        f, _ = NewFeatureFlag("f3", "x", "", true, TargetUsers, "u1, u2, u3", 0, now)
        if !f.IsEnabledFor("u1", nil) {
                t.Error("expected true for user u1 in list")
        }
        if f.IsEnabledFor("u4", nil) {
                t.Error("expected false for user u4 not in list")
        }

        // TargetRoles — user has matching role
        f, _ = NewFeatureFlag("f4", "x", "", true, TargetRoles, "admin, agent", 0, now)
        if !f.IsEnabledFor("u1", []string{"admin"}) {
                t.Error("expected true for user with admin role")
        }
        if f.IsEnabledFor("u1", []string{"driver"}) {
                t.Error("expected false for user with only driver role")
        }

        // TargetPercent — 100%
        f, _ = NewFeatureFlag("f5", "x", "", true, TargetPercent, "", 100, now)
        if !f.IsEnabledFor("any-user", nil) {
                t.Error("expected true for 100% rollout")
        }

        // TargetPercent — 0%
        f, _ = NewFeatureFlag("f6", "x", "", true, TargetPercent, "", 0, now)
        if f.IsEnabledFor("any-user", nil) {
                t.Error("expected false for 0% rollout")
        }

        // TargetPercent — 50% (deterministic based on hash)
        f, _ = NewFeatureFlag("f7", "x", "", true, TargetPercent, "", 50, now)
        // Test that the same user always gets the same result (deterministic)
        result1 := f.IsEnabledFor("user-abc", nil)
        result2 := f.IsEnabledFor("user-abc", nil)
        if result1 != result2 {
                t.Error("expected deterministic result for same user")
        }
}

func TestFeatureFlagSetEnabled(t *testing.T) {
        now := testNowSettings()
        f, _ := NewFeatureFlag("f1", "x", "", false, TargetAll, "", 0, now)

        enabled := f.SetEnabled(true, now)
        if !enabled.Enabled() {
                t.Error("expected enabled=true")
        }

        disabled := enabled.SetEnabled(false, now)
        if disabled.Enabled() {
                t.Error("expected enabled=false")
        }
}

func TestSplitCSV(t *testing.T) {
        tests := []struct {
                input string
                want  []string
        }{
                {"", nil},
                {"a", []string{"a"}},
                {"a,b,c", []string{"a", "b", "c"}},
                {"a, b , c", []string{"a", "b", "c"}},
                {"  spaced  ", []string{"spaced"}},
                {",,", nil},
                {"a,,b", []string{"a", "b"}},
        }
        for _, tt := range tests {
                got := splitCSV(tt.input)
                if len(got) != len(tt.want) {
                        t.Errorf("splitCSV(%q) = %v, want %v", tt.input, got, tt.want)
                        continue
                }
                for i := range got {
                        if got[i] != tt.want[i] {
                                t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
                        }
                }
        }
}

func TestHashToBucket(t *testing.T) {
        // Same input → same output (deterministic)
        b1 := hashToBucket("user-123")
        b2 := hashToBucket("user-123")
        if b1 != b2 {
                t.Error("expected deterministic bucket")
        }

        // Different inputs → likely different buckets (not guaranteed, but statistically)
        // We just check they're in range 0-99
        for i := 0; i < 100; i++ {
                b := hashToBucket("user-" + itoaSimple(i))
                if b < 0 || b > 99 {
                        t.Errorf("bucket out of range: %d", b)
                }
        }
}

func itoaSimple(n int) string {
        if n == 0 {
                return "0"
        }
        var buf [20]byte
        i := len(buf)
        for n > 0 {
                i--
                buf[i] = byte('0' + n%10)
                n /= 10
        }
        return string(buf[i:])
}

// errIs helper
func errIs(err, target error) bool {
        if err == target {
                return true
        }
        for {
                type unwrapper interface{ Unwrap() error }
                u, ok := err.(unwrapper)
                if !ok {
                        return false
                }
                err = u.Unwrap()
                if err == target {
                        return true
                }
                if err == nil {
                        return false
                }
        }
}
