// Package service tests: Hub in-memory implementation.
package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"avex-backend/internal/modules/realtime/domain"
	"avex-backend/internal/modules/realtime/port"
)

// ===== Mock Adapters =====

type mockLogger struct{}

func (mockLogger) Debug(string, ...any) {}
func (mockLogger) Info(string, ...any)  {}
func (mockLogger) Warn(string, ...any)  {}
func (mockLogger) Error(string, ...any) {}

type mockClock struct{ t time.Time }

func (m *mockClock) Now() time.Time { return m.t }

// mockSendFn captures messages sent to a client.
type mockSendFn struct {
	mu       sync.Mutex
	messages []domain.Message
	failWith error
}

func (m *mockSendFn) send(ctx context.Context, msg domain.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWith != nil {
		return m.failWith
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockSendFn) getMessages() []domain.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]domain.Message, len(m.messages))
	copy(cp, m.messages)
	return cp
}

// ===== Tests =====

func newTestHub(t *testing.T) port.Hub {
	t.Helper()
	return NewHub(mockLogger{}, &mockClock{t: time.Now().UTC()})
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()
	sendFn := &mockSendFn{}

	// Register
	if err := hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, sendFn.send); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if hub.GetClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", hub.GetClientCount())
	}

	// Unregister
	hub.Unregister(ctx, "c-1")
	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", hub.GetClientCount())
	}
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()
	sendFn := &mockSendFn{}
	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, sendFn.send)

	// Subscribe
	if err := hub.Subscribe(ctx, "c-1", "order:ord-1"); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if hub.GetChannelCount() != 1 {
		t.Errorf("expected 1 channel, got %d", hub.GetChannelCount())
	}

	// Duplicate subscribe
	if err := hub.Subscribe(ctx, "c-1", "order:ord-1"); err == nil {
		t.Errorf("expected error on duplicate subscribe")
	}

	// Unsubscribe
	if err := hub.Unsubscribe(ctx, "c-1", "order:ord-1"); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if hub.GetChannelCount() != 0 {
		t.Errorf("expected 0 channels after unsubscribe, got %d", hub.GetChannelCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()

	// Register two clients subscribed to the same channel
	send1 := &mockSendFn{}
	send2 := &mockSendFn{}
	send3 := &mockSendFn{} // subscribed to a different channel

	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, send1.send)
	hub.Register(ctx, "c-2", "u-2", domain.ClientRoleUser, send2.send)
	hub.Register(ctx, "c-3", "u-3", domain.ClientRoleUser, send3.send)

	hub.Subscribe(ctx, "c-1", "order:ord-1")
	hub.Subscribe(ctx, "c-2", "order:ord-1")
	hub.Subscribe(ctx, "c-3", "order:ord-2")

	// Broadcast to "order:ord-1"
	data, _ := json.Marshal(map[string]string{"status": "confirmed"})
	msg, _ := domain.NewMessage("m-1", domain.MsgTypeOrderStatus, "order:ord-1", data, "2026-01-01T00:00:00Z")

	delivered := hub.Broadcast(ctx, "order:ord-1", msg)
	if delivered != 2 {
		t.Errorf("expected 2 delivered, got %d", delivered)
	}

	// Verify client 1 and 2 received the message, client 3 did not
	msgs1 := send1.getMessages()
	msgs2 := send2.getMessages()
	msgs3 := send3.getMessages()

	if len(msgs1) != 1 || len(msgs2) != 1 {
		t.Errorf("expected 1 message each for c-1 and c-2, got %d and %d", len(msgs1), len(msgs2))
	}
	if len(msgs3) != 0 {
		t.Errorf("expected 0 messages for c-3, got %d", len(msgs3))
	}
	if len(msgs1) > 0 && msgs1[0].ID != "m-1" {
		t.Errorf("wrong message ID: %s", msgs1[0].ID)
	}
}

func TestHubBroadcastMulti(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()

	send1 := &mockSendFn{}
	send2 := &mockSendFn{}

	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, send1.send)
	hub.Register(ctx, "c-2", "u-2", domain.ClientRoleDriver, send2.send)

	hub.Subscribe(ctx, "c-1", "order:ord-1") // user watching order
	hub.Subscribe(ctx, "c-2", "driver:d-1")  // driver's own channel

	// Broadcast to both channels
	data, _ := json.Marshal(map[string]string{"event": "test"})
	msg, _ := domain.NewMessage("m-1", domain.MsgTypeOrderStatus, "order:ord-1", data, "2026-01-01T00:00:00Z")

	delivered := hub.BroadcastMulti(ctx, []string{"order:ord-1", "driver:d-1"}, msg)
	if delivered != 2 {
		t.Errorf("expected 2 delivered, got %d", delivered)
	}

	// Verify no duplicate delivery (client subscribed to both channels would still get 1)
	// In this case, c-1 is on order:ord-1 and c-2 is on driver:d-1, so each gets 1.
	if len(send1.getMessages()) != 1 || len(send2.getMessages()) != 1 {
		t.Errorf("expected 1 message each, got %d and %d", len(send1.getMessages()), len(send2.getMessages()))
	}
}

func TestHubBroadcastNoSubscribers(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()

	data, _ := json.Marshal(map[string]string{"event": "test"})
	msg, _ := domain.NewMessage("m-1", domain.MsgTypeOrderStatus, "order:ord-1", data, "2026-01-01T00:00:00Z")

	delivered := hub.Broadcast(ctx, "order:nonexistent", msg)
	if delivered != 0 {
		t.Errorf("expected 0 delivered, got %d", delivered)
	}
}

func TestHubGetClientSubscriptions(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()
	sendFn := &mockSendFn{}
	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, sendFn.send)

	hub.Subscribe(ctx, "c-1", "order:ord-1")
	hub.Subscribe(ctx, "c-1", "user:u-1")

	subs, err := hub.GetClientSubscriptions(ctx, "c-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}

	// Non-existent client
	_, err = hub.GetClientSubscriptions(ctx, "nonexistent")
	if err == nil {
		t.Errorf("expected error for non-existent client")
	}
}

func TestHubUnregisterRemovesFromChannels(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()
	sendFn := &mockSendFn{}
	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, sendFn.send)
	hub.Subscribe(ctx, "c-1", "order:ord-1")

	if hub.GetChannelCount() != 1 {
		t.Errorf("expected 1 channel, got %d", hub.GetChannelCount())
	}

	// Unregister should remove from channels too
	hub.Unregister(ctx, "c-1")
	if hub.GetChannelCount() != 0 {
		t.Errorf("expected 0 channels after unregister, got %d", hub.GetChannelCount())
	}
	if hub.GetClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", hub.GetClientCount())
	}
}

func TestHubBroadcastFailingClient(t *testing.T) {
	hub := newTestHub(t)
	ctx := context.Background()

	// Client that always fails
	sendFail := &mockSendFn{failWith: domain.ErrConnectionClosed}
	hub.Register(ctx, "c-1", "u-1", domain.ClientRoleUser, sendFail.send)
	hub.Subscribe(ctx, "c-1", "order:ord-1")

	data, _ := json.Marshal(map[string]string{"event": "test"})
	msg, _ := domain.NewMessage("m-1", domain.MsgTypeOrderStatus, "order:ord-1", data, "2026-01-01T00:00:00Z")

	delivered := hub.Broadcast(ctx, "order:ord-1", msg)
	if delivered != 0 {
		t.Errorf("expected 0 delivered (client failed), got %d", delivered)
	}

	// The failing client should be unregistered (async)
	time.Sleep(100 * time.Millisecond)
	if hub.GetClientCount() != 0 {
		t.Errorf("expected failing client to be unregistered, got %d clients", hub.GetClientCount())
	}
}
