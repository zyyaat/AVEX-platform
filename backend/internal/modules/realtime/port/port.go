// Package port: Hub interface + ServicePort + DTOs for the realtime module.
//
// The Hub is the central message broker that manages connected WebSocket
// clients and routes messages to subscribed channels.
//
// Architecture:
//   - Transport layer accepts WebSocket connections and registers them with the Hub.
//   - Service layer (Hub impl) maintains the in-memory client map + channel subscriptions.
//   - Jobs layer (bus subscriber) listens to Redis Pub/Sub events from other
//     modules (orders, dispatch, financial) and broadcasts to the Hub.
//   - Transport layer reads subscribe/unsubscribe requests from the client and
//     forwards them to the Hub.
package port

import (
        "context"
        "encoding/json"
        "net/http"
        "sync"
        "time"

        "avex-backend/internal/modules/realtime/domain"
)

// ===== Hub Interface =====

// Hub is the central message broker for WebSocket clients.
//
// All methods are safe for concurrent use.
type Hub interface {
        // Register adds a new client connection to the hub.
        // sendFn is provided by the transport layer and is used to push messages
        // to the WebSocket. The Hub calls sendFn for each broadcast.
        // The client is initially subscribed to no channels.
        Register(ctx context.Context, clientID, userID string, role domain.ClientRole, sendFn SendFunc) error

        // Unregister removes a client from the hub and closes its send channel.
        Unregister(ctx context.Context, clientID string)

        // Subscribe adds a channel subscription for a client.
        // Returns ErrClientNotFound if the client is not registered,
        // ErrAlreadySubscribed if already subscribed.
        Subscribe(ctx context.Context, clientID, channel string) error

        // Unsubscribe removes a channel subscription for a client.
        Unsubscribe(ctx context.Context, clientID, channel string) error

        // Broadcast sends a message to all clients subscribed to the given channel.
        // Non-blocking — if a client's send buffer is full, the message is dropped
        // for that client (and the client may be disconnected).
        Broadcast(ctx context.Context, channel string, msg domain.Message) int

        // BroadcastMulti sends a message to all clients subscribed to any of the
        // given channels. Returns the total number of clients reached.
        BroadcastMulti(ctx context.Context, channels []string, msg domain.Message) int

        // GetClientCount returns the total number of connected clients.
        GetClientCount() int

        // GetChannelCount returns the number of channels with at least one subscriber.
        GetChannelCount() int

        // GetClientSubscriptions returns the list of channels a client is subscribed to.
        // Returns ErrClientNotFound if the client is not registered.
        GetClientSubscriptions(ctx context.Context, clientID string) ([]string, error)
}

// SendFunc writes a message to a connected WebSocket client.
// The transport layer provides this implementation; the Hub uses it to
// push messages to clients.
type SendFunc func(ctx context.Context, msg domain.Message) error

// ===== ServicePort =====

// ServicePort is the realtime module's service interface.
type ServicePort interface {
        // HandleWebSocket upgrades the HTTP connection to a WebSocket and
        // registers the client with the Hub. Blocks until the client disconnects.
        HandleWebSocket(w http.ResponseWriter, r *http.Request, userID string, role domain.ClientRole)

        // BroadcastOrderEvent broadcasts an order-related event to all subscribers
        // of the order's channel + the user's channel + the driver's channel.
        BroadcastOrderEvent(ctx context.Context, orderID, userID, driverID string, msgType domain.MessageType, data json.RawMessage)

        // BroadcastDriverEvent broadcasts a driver-related event to the driver's channel.
        BroadcastDriverEvent(ctx context.Context, driverID string, msgType domain.MessageType, data json.RawMessage)

        // BroadcastDispatchOffer broadcasts a new dispatch offer to the driver's channel.
        BroadcastDispatchOffer(ctx context.Context, driverID string, data json.RawMessage)

        // BroadcastWalletEvent broadcasts a wallet event to the owner's channel.
        BroadcastWalletEvent(ctx context.Context, ownerType, ownerID string, msgType domain.MessageType, data json.RawMessage)

        // BroadcastSystem broadcasts a system-wide notice to all connected clients.
        BroadcastSystem(ctx context.Context, msgType domain.MessageType, data json.RawMessage)

        // GetStats returns hub statistics.
        GetStats() Stats
}

// Stats holds realtime hub statistics.
type Stats struct {
        ConnectedClients int `json:"connected_clients"`
        ActiveChannels   int `json:"active_channels"`
}

// ===== Deps =====

// Deps holds all dependencies the realtime service layer needs.
type Deps struct {
        Hub     Hub
        Logger  Logger
        IDGen   IDGenerator
        Clock   Clock
        JWTAuth JWTAuthenticator
}

// Logger is a minimal logging interface.
type Logger interface {
        Debug(msg string, args ...any)
        Info(msg string, args ...any)
        Warn(msg string, args ...any)
        Error(msg string, args ...any)
}

// IDGenerator generates unique IDs.
type IDGenerator interface {
        NewID() string
}

// Clock provides the current time.
type Clock interface {
        Now() time.Time
}

// JWTAuthenticator authenticates a JWT token and returns the user ID + role.
// Implemented by the identity module.
type JWTAuthenticator interface {
        // Authenticate validates the token and returns (userID, role, error).
        // role is one of "user", "driver", "merchant", "admin".
        Authenticate(ctx context.Context, token string) (userID string, role string, err error)
}

// ===== Internal Hub Connection (used by transport layer) =====

// Connection represents a single client's connection state inside the Hub.
// This is exported so the transport layer can read from the client's receive
// channel and forward messages to the WebSocket.
type Connection struct {
        Client   domain.Client
        SendFunc SendFunc
        mu       sync.Mutex
}
