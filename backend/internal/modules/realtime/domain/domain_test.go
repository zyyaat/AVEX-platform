// Package domain tests: channel types + message + client.
package domain

import (
	"encoding/json"
	"testing"
)

func TestChannelName(t *testing.T) {
	tests := []struct {
		name      string
		ct        ChannelType
		entityID  string
		want      string
		wantErr   error
	}{
		{"user channel", ChannelTypeUser, "u-1", "user:u-1", nil},
		{"driver channel", ChannelTypeDriver, "d-1", "driver:d-1", nil},
		{"order channel", ChannelTypeOrder, "ord-1", "order:ord-1", nil},
		{"zone channel", ChannelTypeZone, "zone-1", "zone:zone-1", nil},
		{"admin channel", ChannelTypeAdmin, "", "admin", nil},
		{"admin ignores entityID", ChannelTypeAdmin, "ignored", "admin", nil},
		{"invalid type", ChannelType("bogus"), "x", "", ErrInvalidChannelType},
		{"empty entityID for user", ChannelTypeUser, "", "", ErrInvalidInput},
		{"empty entityID for driver", ChannelTypeDriver, "", "", ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChannelName(tt.ct, tt.entityID)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseChannel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType ChannelType
		wantID   string
		wantErr  error
	}{
		{"user", "user:u-1", ChannelTypeUser, "u-1", nil},
		{"driver", "driver:d-1", ChannelTypeDriver, "d-1", nil},
		{"admin", "admin", ChannelTypeAdmin, "", nil},
		{"order", "order:ord-1", ChannelTypeOrder, "ord-1", nil},
		{"empty", "", "", "", ErrEmptyChannelName},
		{"missing colon", "user", "", "", ErrInvalidMessageFormat},
		{"empty entity", "user:", "", "", ErrInvalidInput},
		{"invalid type", "bogus:x", "", "", ErrInvalidChannelType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, id, err := ParseChannel(tt.input)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ct != tt.wantType {
				t.Errorf("type: expected %s, got %s", tt.wantType, ct)
			}
			if id != tt.wantID {
				t.Errorf("id: expected %q, got %q", tt.wantID, id)
			}
		})
	}
}

func TestNewMessage(t *testing.T) {
	data, _ := json.Marshal(map[string]string{"hello": "world"})
	m, err := NewMessage("m-1", MsgTypeOrderStatus, "order:ord-1", data, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID != "m-1" || m.Type != MsgTypeOrderStatus || m.Channel != "order:ord-1" {
		t.Errorf("message fields wrong: %+v", m)
	}

	// Invalid: empty id
	_, err = NewMessage("", MsgTypeOrderStatus, "", nil, "2026-01-01T00:00:00Z")
	if !errIs(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	// Invalid: empty type
	_, err = NewMessage("m-2", "", "", nil, "2026-01-01T00:00:00Z")
	if !errIs(err, ErrEmptyEventType) {
		t.Fatalf("expected ErrEmptyEventType, got %v", err)
	}
}

func TestSubscribeRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		channel string
		wantErr error
	}{
		{"valid subscribe", "subscribe", "order:ord-1", nil},
		{"valid unsubscribe", "unsubscribe", "user:u-1", nil},
		{"invalid action", "bogus", "user:u-1", ErrInvalidInput},
		{"empty channel", "subscribe", "", ErrEmptyChannelName},
		{"invalid channel", "subscribe", "bogus", ErrInvalidMessageFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SubscribeRequest{Action: tt.action, Channel: tt.channel}
			err := req.Validate()
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

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		userID  string
		role    ClientRole
		wantErr error
	}{
		{"valid user", "c-1", "u-1", ClientRoleUser, nil},
		{"valid driver", "c-2", "d-1", ClientRoleDriver, nil},
		{"valid merchant", "c-3", "m-1", ClientRoleMerchant, nil},
		{"valid admin", "c-4", "a-1", ClientRoleAdmin, nil},
		{"empty id", "", "u-1", ClientRoleUser, ErrInvalidID},
		{"empty user", "c-5", "", ClientRoleUser, ErrInvalidInput},
		{"invalid role", "c-6", "u-1", ClientRole("bogus"), ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.id, tt.userID, tt.role)
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

func TestClientSubscribeUnsubscribe(t *testing.T) {
	c, _ := NewClient("c-1", "u-1", ClientRoleUser)

	// Subscribe
	if err := c.Subscribe("order:ord-1"); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if !c.IsSubscribedTo("order:ord-1") {
		t.Errorf("expected subscribed")
	}
	if len(c.Channels()) != 1 {
		t.Errorf("expected 1 channel, got %d", len(c.Channels()))
	}

	// Duplicate subscribe
	err := c.Subscribe("order:ord-1")
	if !errIs(err, ErrAlreadySubscribed) {
		t.Fatalf("expected ErrAlreadySubscribed, got %v", err)
	}

	// Subscribe to second channel
	c.Subscribe("user:u-1")
	if len(c.Channels()) != 2 {
		t.Errorf("expected 2 channels, got %d", len(c.Channels()))
	}

	// Unsubscribe
	if err := c.Unsubscribe("order:ord-1"); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if c.IsSubscribedTo("order:ord-1") {
		t.Errorf("expected not subscribed")
	}

	// Unsubscribe not-subscribed
	err = c.Unsubscribe("order:ord-1")
	if !errIs(err, ErrNotSubscribed) {
		t.Fatalf("expected ErrNotSubscribed, got %v", err)
	}

	// Unsubscribe all
	c.UnsubscribeAll()
	if len(c.Channels()) != 0 {
		t.Errorf("expected 0 channels after unsubscribe all")
	}
}

func TestClientCanSubscribeTo(t *testing.T) {
	user, _ := NewClient("c-1", "u-1", ClientRoleUser)
	driver, _ := NewClient("c-2", "d-1", ClientRoleDriver)
	admin, _ := NewClient("c-3", "a-1", ClientRoleAdmin)

	tests := []struct {
		name    string
		client  Client
		channel string
		want    bool
	}{
		// User channels
		{"user own", user, "user:u-1", true},
		{"user other", user, "user:u-2", false},
		{"user admin channel", user, "admin", false},

		// Driver channels
		{"driver own", driver, "driver:d-1", true},
		{"driver other", driver, "driver:d-2", false},

		// Admin
		{"admin user channel", admin, "user:u-1", true},
		{"admin driver channel", admin, "driver:d-1", true},
		{"admin admin channel", admin, "admin", true},

		// Order/zone channels — always true at this layer
		{"user order", user, "order:ord-1", true},
		{"driver zone", driver, "zone:zone-1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.client.CanSubscribeTo(tt.channel)
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
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
