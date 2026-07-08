// Package domain contains pure domain entities for the realtime module.
//
// This file: typed domain errors + message types + channel types.
//
// Imports stdlib only.
package domain

import (
        "encoding/json"
        "errors"
        "fmt"
)

// ===== Connection / Channel Errors =====

var ErrChannelNotFound = errors.New("channel not found")
var ErrAlreadySubscribed = errors.New("client already subscribed to channel")
var ErrNotSubscribed = errors.New("client not subscribed to channel")
var ErrClientNotFound = errors.New("client not found")
var ErrInvalidChannelType = errors.New("invalid channel type")
var ErrInvalidMessage = errors.New("invalid message")
var ErrInvalidMessageFormat = errors.New("invalid message format")
var ErrConnectionClosed = errors.New("connection closed")
var ErrWriteTimeout = errors.New("write timeout")
var ErrBufferFull = errors.New("client send buffer is full")

// ===== Validation Errors =====

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptyChannelName = errors.New("channel name is required")
var ErrEmptyEventType = errors.New("event type is required")

// ChannelType enumerates the kinds of channels clients can subscribe to.
type ChannelType string

const (
        ChannelTypeUser     ChannelType = "user"      // user:{user_id}
        ChannelTypeDriver   ChannelType = "driver"    // driver:{driver_id}
        ChannelTypeMerchant ChannelType = "merchant"  // merchant:{merchant_id}
        ChannelTypeOrder    ChannelType = "order"     // order:{order_id}
        ChannelTypeZone     ChannelType = "zone"      // zone:{zone_id}
        ChannelTypeAdmin    ChannelType = "admin"     // admin (broadcast)
)

// IsValid reports whether the channel type is recognized.
func (c ChannelType) IsValid() bool {
        switch c {
        case ChannelTypeUser, ChannelTypeDriver, ChannelTypeMerchant,
                ChannelTypeOrder, ChannelTypeZone, ChannelTypeAdmin:
                return true
        }
        return false
}

// ChannelName constructs the canonical channel name string from a type + entity ID.
// For ChannelTypeAdmin, the entityID is ignored (returns "admin").
func ChannelName(t ChannelType, entityID string) (string, error) {
        if !t.IsValid() {
                return "", fmt.Errorf("%w: %s", ErrInvalidChannelType, t)
        }
        if t == ChannelTypeAdmin {
                return "admin", nil
        }
        if entityID == "" {
                return "", fmt.Errorf("%w: entity id required for channel type %s", ErrInvalidInput, t)
        }
        return string(t) + ":" + entityID, nil
}

// ParseChannel parses a channel name string back into (type, entityID).
// Examples:
//   "user:abc-123" → (ChannelTypeUser, "abc-123")
//   "admin" → (ChannelTypeAdmin, "")
func ParseChannel(name string) (ChannelType, string, error) {
        if name == "" {
                return "", "", ErrEmptyChannelName
        }
        if name == "admin" {
                return ChannelTypeAdmin, "", nil
        }
        for i := 0; i < len(name); i++ {
                if name[i] == ':' {
                        t := ChannelType(name[:i])
                        if !t.IsValid() {
                                return "", "", fmt.Errorf("%w: %s", ErrInvalidChannelType, t)
                        }
                        entityID := name[i+1:]
                        if entityID == "" {
                                return "", "", fmt.Errorf("%w: empty entity id in channel %q", ErrInvalidInput, name)
                        }
                        return t, entityID, nil
                }
        }
        return "", "", fmt.Errorf("%w: missing ':' separator in %q", ErrInvalidMessageFormat, name)
}

// ===== Message Types =====

// MessageType enumerates the kinds of messages sent over WebSocket.
type MessageType string

const (
        MsgTypeOrderStatus       MessageType = "order.status_changed"
        MsgTypeOrderLocation     MessageType = "order.location_update"
        MsgTypeDispatchOffer     MessageType = "dispatch.offer_created"
        MsgTypeDispatchAccepted  MessageType = "dispatch.offer_accepted"
        MsgTypeDispatchRejected  MessageType = "dispatch.offer_rejected"
        MsgTypeDispatchExpired   MessageType = "dispatch.offer_expired"
        MsgTypeDriverLocation    MessageType = "driver.location_update"
        MsgTypeDriverStatus      MessageType = "driver.status_changed"
        MsgTypeWalletCredited    MessageType = "wallet.credited"
        MsgTypeWalletDebited     MessageType = "wallet.debited"
        MsgTypeNotification      MessageType = "notification"
        MsgTypeSystemNotice      MessageType = "system.notice"
        MsgTypePing              MessageType = "_ping" // internal, not broadcast
        MsgTypePong              MessageType = "_pong"
)

// Message is the wire format for all WebSocket messages.
//
// Design:
//   - Type identifies the message kind (used by clients to dispatch to handlers).
//   - Channel is the channel the message was broadcast on (for filtering).
//   - Data is the JSON payload specific to the message type.
//   - Timestamp is when the message was created (UTC RFC3339).
//   - ID is a unique message ID for idempotency / dedup on the client.
type Message struct {
        ID        string          `json:"id"`
        Type      MessageType     `json:"type"`
        Channel   string          `json:"channel,omitempty"`
        Data      json.RawMessage `json:"data,omitempty"`
        Timestamp string          `json:"timestamp"` // RFC3339
}

// NewMessage constructs a Message with a generated ID + current UTC timestamp.
// The caller must marshal the Data field separately.
func NewMessage(id string, msgType MessageType, channel string, data []byte, timestamp string) (Message, error) {
        if id == "" {
                return Message{}, fmt.Errorf("%w: id is required", ErrInvalidID)
        }
        if msgType == "" {
                return Message{}, ErrEmptyEventType
        }
        if timestamp == "" {
                return Message{}, fmt.Errorf("%w: timestamp is required", ErrInvalidInput)
        }
        return Message{
                ID:        id,
                Type:      msgType,
                Channel:   channel,
                Data:      data,
                Timestamp: timestamp,
        }, nil
}

// ===== Subscription Request (from client) =====

// SubscribeRequest is the JSON format clients send to subscribe/unsubscribe.
type SubscribeRequest struct {
        Action    string `json:"action"` // "subscribe" | "unsubscribe"
        Channel   string `json:"channel"` // e.g. "order:abc-123"
}

// Validate checks the request is well-formed.
func (r SubscribeRequest) Validate() error {
        if r.Action != "subscribe" && r.Action != "unsubscribe" {
                return fmt.Errorf("%w: action must be 'subscribe' or 'unsubscribe', got %q", ErrInvalidInput, r.Action)
        }
        if r.Channel == "" {
                return ErrEmptyChannelName
        }
        if _, _, err := ParseChannel(r.Channel); err != nil {
                return err
        }
        return nil
}

// ===== Client Entity =====

// ClientRole enumerates the role of a connected client.
type ClientRole string

const (
        ClientRoleUser     ClientRole = "user"
        ClientRoleDriver   ClientRole = "driver"
        ClientRoleMerchant ClientRole = "merchant"
        ClientRoleAdmin    ClientRole = "admin"
)

// Client is a connected WebSocket client.
// The Hub tracks clients by ID + role, and routes messages to subscribed channels.
type Client struct {
        id       string
        userID   string
        role     ClientRole
        channels map[string]struct{} // subscribed channel names
}

// NewClient creates a new Client.
func NewClient(id, userID string, role ClientRole) (Client, error) {
        if id == "" {
                return Client{}, fmt.Errorf("%w: client id is required", ErrInvalidID)
        }
        if userID == "" {
                return Client{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
        }
        if role != ClientRoleUser && role != ClientRoleDriver && role != ClientRoleMerchant && role != ClientRoleAdmin {
                return Client{}, fmt.Errorf("%w: %s", ErrInvalidInput, role)
        }
        return Client{
                id:       id,
                userID:   userID,
                role:     role,
                channels: make(map[string]struct{}),
        }, nil
}

// ===== Accessors =====

func (c Client) ID() string        { return c.id }
func (c Client) UserID() string    { return c.userID }
func (c Client) Role() ClientRole  { return c.role }
func (c Client) Channels() []string {
        out := make([]string, 0, len(c.channels))
        for ch := range c.channels {
                out = append(out, ch)
        }
        return out
}

// IsSubscribedTo reports whether the client is subscribed to the given channel.
func (c Client) IsSubscribedTo(channel string) bool {
        _, ok := c.channels[channel]
        return ok
}

// Subscribe adds a channel to the client's subscription set.
// Returns ErrAlreadySubscribed if already subscribed.
func (c *Client) Subscribe(channel string) error {
        if channel == "" {
                return ErrEmptyChannelName
        }
        if _, ok := c.channels[channel]; ok {
                return ErrAlreadySubscribed
        }
        c.channels[channel] = struct{}{}
        return nil
}

// Unsubscribe removes a channel from the client's subscription set.
// Returns ErrNotSubscribed if not subscribed.
func (c *Client) Unsubscribe(channel string) error {
        if channel == "" {
                return ErrEmptyChannelName
        }
        if _, ok := c.channels[channel]; !ok {
                return ErrNotSubscribed
        }
        delete(c.channels, channel)
        return nil
}

// UnsubscribeAll removes all channel subscriptions.
func (c *Client) UnsubscribeAll() {
        c.channels = make(map[string]struct{})
}

// ===== Authorization =====

// CanSubscribeTo reports whether the client is authorized to subscribe to
// the given channel. This enforces per-role access control.
//
// Rules:
//   - user:{id}     — only the user themselves (or admin)
//   - driver:{id}   — only the driver themselves (or admin)
//   - merchant:{id} — only the merchant themselves (or admin)
//   - order:{id}    — the order's user, driver, merchant, or admin
//                     (the Hub verifies the order relationship via the orders module)
//   - zone:{id}     — drivers serving the zone (or admin)
//   - admin         — admin role only
//
// For order:{id} and zone:{id}, the Hub performs additional checks using
// the orders / dispatch modules. This method only checks the simple cases.
func (c Client) CanSubscribeTo(channel string) bool {
        t, entityID, err := ParseChannel(channel)
        if err != nil {
                return false
        }
        switch t {
        case ChannelTypeAdmin:
                return c.role == ClientRoleAdmin
        case ChannelTypeUser:
                return c.role == ClientRoleAdmin || c.userID == entityID
        case ChannelTypeDriver:
                return c.role == ClientRoleAdmin || (c.role == ClientRoleDriver && c.userID == entityID)
        case ChannelTypeMerchant:
                return c.role == ClientRoleAdmin || (c.role == ClientRoleMerchant && c.userID == entityID)
        case ChannelTypeOrder, ChannelTypeZone:
                // The Hub does additional verification for these.
                // For now, allow all authenticated clients; the Hub will check order ownership.
                return true
        }
        return false
}
