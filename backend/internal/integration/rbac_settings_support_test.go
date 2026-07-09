//go:build integration

package integration_test

import (
	"context"
	"testing"

	permissionsport "avex-backend/internal/modules/permissions/port"
	settingsport "avex-backend/internal/modules/settings/port"
	supportport "avex-backend/internal/modules/support/port"
)

// TestPermissions_RBAC tests role assignment + permission checking.
func TestPermissions_RBAC(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Get the admin role (seeded)
	adminRole, err := permissionsMod.Service().GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatalf("GetRoleByName(admin): %v", err)
	}

	// 2. Assign admin role to a user
	_, err = permissionsMod.Service().AssignRole(ctx, permissionsport.AssignRoleInput{
		UserID:     "rbac-test-user",
		RoleID:     adminRole.ID,
		AssignedBy: "system",
	})
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	// 3. Check that the user has admin permissions (wildcard)
	result, err := permissionsMod.Service().HasPermission(ctx, "rbac-test-user", "permissions.role.read")
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !result.Allowed {
		t.Fatal("expected admin user to have permissions.role.read permission")
	}

	// 4. Check that a non-admin user does NOT have admin permissions
	result, err = permissionsMod.Service().HasPermission(ctx, "non-admin-user", "permissions.role.read")
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if result.Allowed {
		t.Fatal("expected non-admin user to NOT have permissions.role.read permission")
	}

	// 5. Check the user's roles
	roles, err := permissionsMod.Service().ListRolesByUser(ctx, "rbac-test-user")
	if err != nil {
		t.Fatalf("ListRolesByUser: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	if roles[0].Name != "admin" {
		t.Fatalf("expected admin role, got %s", roles[0].Name)
	}
}

// TestSettings_VersionedConfig tests setting create → update → rollback.
func TestSettings_VersionedConfig(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create a setting
	setting, err := settingsMod.Service().CreateSetting(ctx, settingsport.CreateSettingInput{
		Key:         "integration.test.value",
		Description: "Test setting for integration tests",
		Type:        "int",
		Value:       "10",
	})
	if err != nil {
		t.Fatalf("CreateSetting: %v", err)
	}
	if setting.Version != 1 {
		t.Fatalf("expected version 1, got %d", setting.Version)
	}

	// 2. Update the setting
	setting, err = settingsMod.Service().UpdateSetting(ctx, setting.ID, settingsport.UpdateSettingInput{
		Value:      "20",
		ChangedBy:  "test-admin",
		ChangeNote: "Doubled the value",
	})
	if err != nil {
		t.Fatalf("UpdateSetting: %v", err)
	}
	if setting.Version != 2 {
		t.Fatalf("expected version 2, got %d", setting.Version)
	}
	if setting.Value != "20" {
		t.Fatalf("expected value '20', got %s", setting.Value)
	}

	// 3. List revisions
	revisions, err := settingsMod.Service().ListRevisions(ctx, setting.ID, settingsport.PageQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(revisions.Items) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revisions.Items))
	}

	// 4. Rollback to version 1
	setting, err = settingsMod.Service().RollbackSetting(ctx, setting.ID, 1, "test-admin")
	if err != nil {
		t.Fatalf("RollbackSetting: %v", err)
	}
	if setting.Value != "10" {
		t.Fatalf("expected value '10' after rollback, got %s", setting.Value)
	}
	if setting.Version != 3 {
		t.Fatalf("expected version 3 after rollback, got %d", setting.Version)
	}
}

// TestSettings_FeatureFlags tests feature flag toggle + checking.
func TestSettings_FeatureFlags(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create a feature flag (50% rollout)
	flag, err := settingsMod.Service().CreateFeatureFlag(ctx, settingsport.CreateFeatureFlagInput{
		Name:        "integration.test.flag",
		Description: "Test feature flag",
		Enabled:     true,
		TargetType:  "percent",
		RolloutPct:  50,
	})
	if err != nil {
		t.Fatalf("CreateFeatureFlag: %v", err)
	}

	// 2. Check that the flag exists
	result, err := settingsMod.Service().IsFeatureEnabled(ctx, "integration.test.flag", "test-user-1", nil)
	if err != nil {
		t.Fatalf("IsFeatureEnabled: %v", err)
	}
	// For 50% rollout, the result depends on the hash of "test-user-1"
	// We just verify it returns a result without error
	t.Logf("Feature flag for test-user-1: enabled=%v", result.Enabled)

	// 3. Update to 100% rollout — should be enabled for everyone
	flag, err = settingsMod.Service().UpdateFeatureFlag(ctx, flag.ID, settingsport.UpdateFeatureFlagInput{
		RolloutPct: intPtr(100),
	})
	if err != nil {
		t.Fatalf("UpdateFeatureFlag: %v", err)
	}

	// 4. Check — should be enabled for any user
	result, err = settingsMod.Service().IsFeatureEnabled(ctx, "integration.test.flag", "any-user", nil)
	if err != nil {
		t.Fatalf("IsFeatureEnabled: %v", err)
	}
	if !result.Enabled {
		t.Fatal("expected flag to be enabled for 100% rollout")
	}

	// 5. Disable the flag
	flag, err = settingsMod.Service().UpdateFeatureFlag(ctx, flag.ID, settingsport.UpdateFeatureFlagInput{
		Enabled: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("UpdateFeatureFlag: %v", err)
	}

	// 6. Check — should be disabled
	result, _ = settingsMod.Service().IsFeatureEnabled(ctx, "integration.test.flag", "any-user", nil)
	if result.Enabled {
		t.Fatal("expected flag to be disabled after toggle off")
	}
}

// TestSupport_TicketLifecycle tests the support ticket lifecycle.
func TestSupport_TicketLifecycle(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create ticket
	ticket, err := supportMod.Service().CreateTicket(ctx, supportport.CreateTicketInput{
		UserID:      "support-test-user",
		Subject:     "Order not delivered",
		Description: "My order was supposed to arrive 30 minutes ago",
		Category:    "order_issue",
		Priority:    "high",
		CreatedBy:   "user",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.Status != "open" {
		t.Fatalf("expected open, got %s", ticket.Status)
	}

	// 2. Assign to an agent
	ticket, err = supportMod.Service().AssignTicket(ctx, supportport.AssignTicketInput{
		TicketID: ticket.ID,
		AgentID:  "agent-test-1",
	})
	if err != nil {
		t.Fatalf("AssignTicket: %v", err)
	}
	if ticket.Status != "in_progress" {
		t.Fatalf("expected in_progress, got %s", ticket.Status)
	}
	if ticket.AssignedTo != "agent-test-1" {
		t.Fatalf("expected agent-test-1, got %s", ticket.AssignedTo)
	}

	// 3. Agent replies
	msg, err := supportMod.Service().ReplyToTicket(ctx, supportport.ReplyTicketInput{
		TicketID:   ticket.ID,
		SenderType: "agent",
		SenderID:   "agent-test-1",
		Body:       "I'm looking into this. Can you provide your order number?",
	})
	if err != nil {
		t.Fatalf("ReplyToTicket: %v", err)
	}
	if msg.Body == "" {
		t.Fatal("expected non-empty message body")
	}

	// 4. Ticket should be in "waiting" status (agent replied → waiting for user)
	ticket, _ = supportMod.Service().GetTicket(ctx, ticket.ID)
	if ticket.Status != "waiting" {
		t.Fatalf("expected waiting after agent reply, got %s", ticket.Status)
	}

	// 5. User replies
	supportMod.Service().ReplyToTicket(ctx, supportport.ReplyTicketInput{
		TicketID:   ticket.ID,
		SenderType: "user",
		SenderID:   "support-test-user",
		Body:       "Order number is AVEX-12345",
	})

	// 6. Ticket should be back to "in_progress" (user replied)
	ticket, _ = supportMod.Service().GetTicket(ctx, ticket.ID)
	if ticket.Status != "in_progress" {
		t.Fatalf("expected in_progress after user reply, got %s", ticket.Status)
	}

	// 7. Close the ticket
	ticket, err = supportMod.Service().CloseTicket(ctx, supportport.CloseTicketInput{
		TicketID: ticket.ID,
		ClosedBy: "agent",
		Reason:   "Issue resolved",
	})
	if err != nil {
		t.Fatalf("CloseTicket: %v", err)
	}
	if ticket.Status != "closed" {
		t.Fatalf("expected closed, got %s", ticket.Status)
	}

	// 8. List messages — should have 2 (agent + user)
	messages, err := supportMod.Service().ListMessages(ctx, ticket.ID, supportport.PageQuery{Limit: 50})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages.Items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages.Items))
	}
}

// ===== Helpers =====

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }
