// Package service: in-memory Hub implementation.
//
// The Hub maintains:
//   - clients: map[clientID] -> *clientEntry (client metadata + send function)
//   - channels: map[channelName] -> map[clientID] -> struct{} (subscribers)
//
// All operations are guarded by a single RWMutex for simplicity. For high
// throughput, we could shard by channel name, but for ~10K clients this is fine.
package service

import (
        "context"
        "sync"
        "time"

        "avex-backend/internal/modules/realtime/domain"
        "avex-backend/internal/modules/realtime/port"
)

// clientEntry holds a connected client's state inside the Hub.
type clientEntry struct {
        client   domain.Client
        sendFn   port.SendFunc
        lastSeen time.Time
}

// hub implements port.Hub.
type hub struct {
        mu       sync.RWMutex
        clients  map[string]*clientEntry        // clientID -> entry
        channels map[string]map[string]struct{} // channelName -> set of clientIDs
        logger   port.Logger
        clock    port.Clock
}

// NewHub creates a new in-memory Hub.
func NewHub(logger port.Logger, clock port.Clock) port.Hub {
        return &hub{
                clients:  make(map[string]*clientEntry),
                channels: make(map[string]map[string]struct{}),
                logger:   logger,
                clock:    clock,
        }
}

// Register adds a new client to the hub with the given sendFn.
func (h *hub) Register(_ context.Context, clientID, userID string, role domain.ClientRole, sendFn port.SendFunc) error {
        c, err := domain.NewClient(clientID, userID, role)
        if err != nil {
                return err
        }

        h.mu.Lock()
        defer h.mu.Unlock()

        if _, exists := h.clients[clientID]; exists {
                return domain.ErrClientNotFound // reuse — means "already exists" semantically
        }

        h.clients[clientID] = &clientEntry{
                client:   c,
                sendFn:   sendFn,
                lastSeen: h.clock.Now(),
        }
        return nil
}

// Unregister removes a client from the hub.
func (h *hub) Unregister(_ context.Context, clientID string) {
        h.mu.Lock()
        defer h.mu.Unlock()

        entry, ok := h.clients[clientID]
        if !ok {
                return
        }

        // Remove from all subscribed channels
        for _, channel := range entry.client.Channels() {
                if subs, ok := h.channels[channel]; ok {
                        delete(subs, clientID)
                        if len(subs) == 0 {
                                delete(h.channels, channel)
                        }
                }
        }

        delete(h.clients, clientID)
}

// Subscribe adds a channel subscription for a client.
func (h *hub) Subscribe(_ context.Context, clientID, channel string) error {
        h.mu.Lock()
        defer h.mu.Unlock()

        entry, ok := h.clients[clientID]
        if !ok {
                return domain.ErrClientNotFound
        }

        if err := entry.client.Subscribe(channel); err != nil {
                return err
        }

        if h.channels[channel] == nil {
                h.channels[channel] = make(map[string]struct{})
        }
        h.channels[channel][clientID] = struct{}{}

        return nil
}

// Unsubscribe removes a channel subscription for a client.
func (h *hub) Unsubscribe(_ context.Context, clientID, channel string) error {
        h.mu.Lock()
        defer h.mu.Unlock()

        entry, ok := h.clients[clientID]
        if !ok {
                return domain.ErrClientNotFound
        }

        if err := entry.client.Unsubscribe(channel); err != nil {
                return err
        }

        if subs, ok := h.channels[channel]; ok {
                delete(subs, clientID)
                if len(subs) == 0 {
                        delete(h.channels, channel)
                }
        }

        return nil
}

// Broadcast sends a message to all clients subscribed to the given channel.
func (h *hub) Broadcast(ctx context.Context, channel string, msg domain.Message) int {
        h.mu.RLock()
        subs, ok := h.channels[channel]
        if !ok {
                h.mu.RUnlock()
                return 0
        }
        // Copy subscriber IDs to avoid holding the lock during sends.
        subIDs := make([]string, 0, len(subs))
        for id := range subs {
                subIDs = append(subIDs, id)
        }
        h.mu.RUnlock()

        if len(subIDs) == 0 {
                return 0
        }

        delivered := 0
        for _, id := range subIDs {
                h.mu.RLock()
                entry, ok := h.clients[id]
                h.mu.RUnlock()
                if !ok || entry.sendFn == nil {
                        continue
                }
                // Use a short timeout to avoid blocking on slow clients.
                sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                if err := entry.sendFn(sendCtx, msg); err != nil {
                        h.logger.Warn("send to client failed, disconnecting",
                                "client_id", id,
                                "error", err,
                        )
                        go h.Unregister(context.Background(), id)
                } else {
                        delivered++
                }
                cancel()
        }

        return delivered
}

// BroadcastMulti sends a message to all clients subscribed to any of the given channels.
func (h *hub) BroadcastMulti(ctx context.Context, channels []string, msg domain.Message) int {
        h.mu.RLock()
        seen := make(map[string]struct{})
        for _, ch := range channels {
                subs, ok := h.channels[ch]
                if !ok {
                        continue
                }
                for id := range subs {
                        seen[id] = struct{}{}
                }
        }
        h.mu.RUnlock()

        if len(seen) == 0 {
                return 0
        }

        delivered := 0
        for id := range seen {
                h.mu.RLock()
                entry, ok := h.clients[id]
                h.mu.RUnlock()
                if !ok || entry.sendFn == nil {
                        continue
                }
                sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                if err := entry.sendFn(sendCtx, msg); err != nil {
                        h.logger.Warn("send to client failed, disconnecting",
                                "client_id", id,
                                "error", err,
                        )
                        go h.Unregister(context.Background(), id)
                } else {
                        delivered++
                }
                cancel()
        }

        return delivered
}

// GetClientCount returns the total number of connected clients.
func (h *hub) GetClientCount() int {
        h.mu.RLock()
        defer h.mu.RUnlock()
        return len(h.clients)
}

// GetChannelCount returns the number of channels with at least one subscriber.
func (h *hub) GetChannelCount() int {
        h.mu.RLock()
        defer h.mu.RUnlock()
        return len(h.channels)
}

// GetClientSubscriptions returns the list of channels a client is subscribed to.
func (h *hub) GetClientSubscriptions(_ context.Context, clientID string) ([]string, error) {
        h.mu.RLock()
        defer h.mu.RUnlock()
        entry, ok := h.clients[clientID]
        if !ok {
                return nil, domain.ErrClientNotFound
        }
        return entry.client.Channels(), nil
}
