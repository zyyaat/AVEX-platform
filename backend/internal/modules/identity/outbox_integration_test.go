// Package identity integration tests: outbox worker end-to-end test.
//
// This test verifies the full outbox flow:
//   1. Service registers a user (creates outbox event in DB transaction)
//   2. Outbox worker polls the outbox table
//   3. Worker publishes the event to Redis Pub/Sub
//   4. A Redis subscriber receives the event
//   5. The outbox entry is marked as published
//
// Requires: PostgreSQL + Redis running.
//
//go:build integration

package identity_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"avex-backend/internal/modules/identity"
	"avex-backend/internal/modules/identity/port"
	"avex-backend/internal/platform/bus"
	"avex-backend/internal/platform/outbox"
)

// TestOutboxFlow_RegisterUser_PublishesEvent tests the full outbox flow:
// register user → outbox event → worker publishes → subscriber receives.
func TestOutboxFlow_RegisterUser_PublishesEvent(t *testing.T) {
	cleanupIntegTables(t)
	ctx := context.Background()

	// Create identity module (wires service + event publisher + outbox).
	mod := identity.New(integCfg, integDBPool, slog.Default())
	defer mod.Close()
	svc := mod.Service()

	// Start a Redis subscriber BEFORE registering, so we don't miss the event.
	received := make(chan bus.EventEnvelope, 1)
	err := integRedis.Subscribe(ctx, port.EventUserRegistered, func(ctx context.Context, env bus.EventEnvelope) error {
		received <- env
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Create outbox worker.
	identityOutbox := outbox.NewPostgresOutbox(integDBPool, outbox.Config{
		Table:          "identity.outbox",
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
	})
	worker := outbox.NewPublisherWorker(identityOutbox, integRedis, 100*time.Millisecond, 10, slog.Default())

	// Start worker.
	workerCtx, workerCancel := context.WithCancel(ctx)
	go worker.Run(workerCtx)
	defer workerCancel()

	// Register a user — this creates an outbox event in the DB.
	result, err := svc.RegisterUser(ctx, port.RegisterUserInput{
		Name:     "Outbox Test User",
		Phone:    "01012345678",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	if result.Token == "" {
		t.Error("expected non-empty token")
	}

	// Wait for the subscriber to receive the event (timeout after 5s).
	select {
	case env := <-received:
		// Verify the event envelope.
		if env.EventType != port.EventUserRegistered {
			t.Errorf("EventType = %q, want %q", env.EventType, port.EventUserRegistered)
		}
		if env.Producer != "identity" {
			t.Errorf("Producer = %q, want 'identity'", env.Producer)
		}
		if env.EventID == "" {
			t.Error("EventID should not be empty")
		}
		// Verify payload.
		var payload port.UserRegisteredPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.Name != "Outbox Test User" {
			t.Errorf("payload Name = %q", payload.Name)
		}
		if payload.PhoneMasked == "01012345678" {
			t.Error("payload phone should be masked")
		}
		t.Logf("✅ received event: %s, payload: %+v", env.EventID, payload)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout: did not receive event from Redis within 5s")
	}

	// Verify the outbox entry is marked as published.
	var publishedCount int
	err = integDBPool.QueryRow(ctx, `
                SELECT COUNT(*) FROM identity.outbox WHERE published_at IS NOT NULL
        `).Scan(&publishedCount)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if publishedCount == 0 {
		t.Error("expected at least 1 published outbox entry")
	}
	t.Logf("✅ %d outbox entries marked as published", publishedCount)
}

// TestOutboxFlow_MultipleEvents tests that multiple events are published.
func TestOutboxFlow_MultipleEvents(t *testing.T) {
	cleanupIntegTables(t)
	ctx := context.Background()

	mod := identity.New(integCfg, integDBPool, slog.Default())
	defer mod.Close()
	svc := mod.Service()

	// Subscriber for both event types.
	receivedChan := make(chan bus.EventEnvelope, 10)
	err := integRedis.SubscribePattern(ctx, "identity.*", func(ctx context.Context, env bus.EventEnvelope) error {
		receivedChan <- env
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribePattern: %v", err)
	}

	// Create and start worker.
	identityOutbox := outbox.NewPostgresOutbox(integDBPool, outbox.Config{
		Table: "identity.outbox", MaxRetries: 3, RetryBaseDelay: 1 * time.Second,
	})
	worker := outbox.NewPublisherWorker(identityOutbox, integRedis, 100*time.Millisecond, 10, slog.Default())
	workerCtx, workerCancel := context.WithCancel(ctx)
	go worker.Run(workerCtx)
	defer workerCancel()

	// Register + Login = 2 events.
	_, err = svc.RegisterUser(ctx, port.RegisterUserInput{
		Name: "Multi Event", Phone: "01012345678", Password: "password123",
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	_, err = svc.LoginUser(ctx, port.LoginInput{
		Phone: "01012345678", Password: "password123",
	})
	if err != nil {
		t.Fatalf("LoginUser: %v", err)
	}

	// Wait for 2 events.
	received := 0
	timeout := time.After(5 * time.Second)
	for received < 2 {
		select {
		case env := <-receivedChan:
			received++
			t.Logf("✅ received event %d: %s", received, env.EventType)
		case <-timeout:
			t.Fatalf("timeout: received only %d events, expected 2", received)
		}
	}
}

// TestOutboxFlow_TransactionalRollback tests that if a transaction fails,
// no event is published.
func TestOutboxFlow_TransactionalRollback(t *testing.T) {
	cleanupIntegTables(t)
	ctx := context.Background()

	mod := identity.New(integCfg, integDBPool, slog.Default())
	defer mod.Close()
	svc := mod.Service()

	// Create worker.
	identityOutbox := outbox.NewPostgresOutbox(integDBPool, outbox.Config{
		Table: "identity.outbox", MaxRetries: 3, RetryBaseDelay: 1 * time.Second,
	})
	worker := outbox.NewPublisherWorker(identityOutbox, integRedis, 100*time.Millisecond, 10, slog.Default())
	workerCtx, workerCancel := context.WithCancel(ctx)
	go worker.Run(workerCtx)
	defer workerCancel()

	// Register a user successfully.
	_, err := svc.RegisterUser(ctx, port.RegisterUserInput{
		Name: "First User", Phone: "01012345678", Password: "password123",
	})
	if err != nil {
		t.Fatalf("first RegisterUser: %v", err)
	}

	// Try to register the same phone again — should fail.
	_, err = svc.RegisterUser(ctx, port.RegisterUserInput{
		Name: "Second User", Phone: "01012345678", Password: "password456",
	})
	if err == nil {
		t.Fatal("expected duplicate phone error")
	}

	// Wait a moment for the worker to process any pending events.
	time.Sleep(500 * time.Millisecond)

	// Verify only 1 outbox entry exists (the successful registration).
	var count int
	err = integDBPool.QueryRow(ctx, `SELECT COUNT(*) FROM identity.outbox`).Scan(&count)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 outbox entry, got %d", count)
	}
}
