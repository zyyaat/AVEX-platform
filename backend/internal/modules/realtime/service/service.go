// Package service: realtime service implementation.
//
// The Service wraps the Hub and provides higher-level broadcast helpers
// for the jobs layer (bus subscriber) to use.
package service

import (
        "context"
        "encoding/json"
        "net/http"
        "time"

        "avex-backend/internal/modules/realtime/domain"
        "avex-backend/internal/modules/realtime/port"
)

// Service implements port.ServicePort.
type Service struct {
        hub    port.Hub
        logger port.Logger
        idGen  port.IDGenerator
        clock  port.Clock
}

var _ port.ServicePort = (*Service)(nil)

// New creates a new realtime Service.
func New(deps port.Deps) *Service {
        return &Service{
                hub:    deps.Hub,
                logger: deps.Logger,
                idGen:  deps.IDGen,
                clock:  deps.Clock,
        }
}

// ===== HandleWebSocket =====
//
// The WebSocket upgrade logic lives in the transport layer (it needs direct
// access to the HTTP request + WebSocket connection). The Service delegates
// to a function set by the transport layer during wiring.

// wsHandler is set by the transport layer during module wiring.
var wsHandler http.HandlerFunc

// SetWSHandler is called by the transport layer to inject the WebSocket handler.
func SetWSHandler(fn http.HandlerFunc) {
        wsHandler = fn
}

func (s *Service) HandleWebSocket(w http.ResponseWriter, r *http.Request, _ string, _ domain.ClientRole) {
        if wsHandler == nil {
                http.Error(w, "websocket handler not configured", http.StatusInternalServerError)
                return
        }
        wsHandler(w, r)
}

// ===== Broadcast Helpers =====

// newMessage constructs a domain.Message with a fresh ID + timestamp.
func (s *Service) newMessage(msgType domain.MessageType, channel string, data json.RawMessage) domain.Message {
        msg, _ := domain.NewMessage(
                s.idGen.NewID(),
                msgType,
                channel,
                data,
                s.clock.Now().UTC().Format(time.RFC3339),
        )
        return msg
}

// BroadcastOrderEvent broadcasts to order:{orderID} + user:{userID} + driver:{driverID}.
func (s *Service) BroadcastOrderEvent(ctx context.Context, orderID, userID, driverID string, msgType domain.MessageType, data json.RawMessage) {
        channels := []string{"order:" + orderID}
        if userID != "" {
                channels = append(channels, "user:"+userID)
        }
        if driverID != "" {
                channels = append(channels, "driver:"+driverID)
        }
        msg := s.newMessage(msgType, "order:"+orderID, data)
        delivered := s.hub.BroadcastMulti(ctx, channels, msg)
        s.logger.Debug("order event broadcast",
                "order_id", orderID,
                "msg_type", string(msgType),
                "channels", channels,
                "delivered", delivered,
        )
}

// BroadcastDriverEvent broadcasts to driver:{driverID}.
func (s *Service) BroadcastDriverEvent(ctx context.Context, driverID string, msgType domain.MessageType, data json.RawMessage) {
        channel := "driver:" + driverID
        msg := s.newMessage(msgType, channel, data)
        delivered := s.hub.Broadcast(ctx, channel, msg)
        s.logger.Debug("driver event broadcast",
                "driver_id", driverID,
                "msg_type", string(msgType),
                "delivered", delivered,
        )
}

// BroadcastDispatchOffer broadcasts to driver:{driverID}.
func (s *Service) BroadcastDispatchOffer(ctx context.Context, driverID string, data json.RawMessage) {
        s.BroadcastDriverEvent(ctx, driverID, domain.MsgTypeDispatchOffer, data)
}

// BroadcastWalletEvent broadcasts to user/driver/merchant:{ownerID} based on ownerType.
func (s *Service) BroadcastWalletEvent(ctx context.Context, ownerType, ownerID string, msgType domain.MessageType, data json.RawMessage) {
        channel := ownerType + ":" + ownerID
        msg := s.newMessage(msgType, channel, data)
        delivered := s.hub.Broadcast(ctx, channel, msg)
        s.logger.Debug("wallet event broadcast",
                "owner_type", ownerType,
                "owner_id", ownerID,
                "msg_type", string(msgType),
                "delivered", delivered,
        )
}

// BroadcastSystem broadcasts to the "admin" channel.
func (s *Service) BroadcastSystem(ctx context.Context, msgType domain.MessageType, data json.RawMessage) {
        msg := s.newMessage(msgType, "admin", data)
        delivered := s.hub.Broadcast(ctx, "admin", msg)
        s.logger.Debug("system broadcast",
                "msg_type", string(msgType),
                "delivered", delivered,
        )
}

// GetStats returns hub statistics.
func (s *Service) GetStats() port.Stats {
        return port.Stats{
                ConnectedClients: s.hub.GetClientCount(),
                ActiveChannels:   s.hub.GetChannelCount(),
        }
}
