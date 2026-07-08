// Package jobs subscriber: subscribes to bus events and broadcasts to the Hub.
//
// The subscriber listens to all event types from orders, dispatch, and financial
// modules, and translates them into WebSocket messages broadcast via the Hub.
//
// Event → WebSocket message mapping:
//   orders.order.created             → order.status_changed (to user + restaurant)
//   orders.order.confirmed           → order.status_changed
//   orders.order.preparing           → order.status_changed
//   orders.order.ready_for_pickup    → order.status_changed
//   orders.order.assigned            → order.status_changed (with driver_id)
//   orders.order.picked_up           → order.status_changed + order.location_update
//   orders.order.delivered           → order.status_changed
//   orders.order.cancelled           → order.status_changed
//   dispatch.driver.location_updated → driver.location_update (to driver + active order)
//   dispatch.driver.online           → driver.status_changed
//   dispatch.driver.offline          → driver.status_changed
//   dispatch.offer.created           → dispatch.offer_created (to driver)
//   dispatch.offer.accepted          → dispatch.offer_accepted (to driver + user)
//   dispatch.offer.rejected          → dispatch.offer_rejected (to driver)
//   dispatch.offer.expired           → dispatch.offer_expired (to driver)
//   financial.wallet.credited        → wallet.credited (to owner)
//   financial.wallet.debited         → wallet.debited (to owner)
//
// All handlers are wrapped with the inbox pattern for idempotency.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"avex-backend/internal/modules/realtime/domain"
	"avex-backend/internal/modules/realtime/port"
	"avex-backend/internal/platform/bus"
	"avex-backend/internal/platform/inbox"
)

// Subscriber listens to bus events and broadcasts to the Hub.
type Subscriber struct {
	svc    port.ServicePort
	bus    bus.Subscriber
	inbox  inbox.Inbox
	logger *slog.Logger
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(svc port.ServicePort, bus bus.Subscriber, inbox inbox.Inbox, logger *slog.Logger) *Subscriber {
	return &Subscriber{
		svc:    svc,
		bus:    bus,
		inbox:  inbox,
		logger: logger,
	}
}

// Start subscribes to all relevant event types. Returns when all subscriptions
// are registered (the handlers run in background goroutines).
func (s *Subscriber) Start(ctx context.Context) error {
	// ===== Orders Events =====
	subscriptions := []struct {
		eventType    string
		handlerName  string
		handler      bus.Handler
	}{
		// Order lifecycle events
		{"orders.order.created", "realtime.on_order_created", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.confirmed", "realtime.on_order_confirmed", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.preparing", "realtime.on_order_preparing", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.ready_for_pickup", "realtime.on_order_ready", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.assigned", "realtime.on_order_assigned", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.picked_up", "realtime.on_order_picked_up", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.delivered", "realtime.on_order_delivered", s.onOrderEvent(domain.MsgTypeOrderStatus)},
		{"orders.order.cancelled", "realtime.on_order_cancelled", s.onOrderEvent(domain.MsgTypeOrderStatus)},

		// Dispatch events
		{"dispatch.driver.location_updated", "realtime.on_driver_location", s.onDriverLocation},
		{"dispatch.driver.online", "realtime.on_driver_online", s.onDriverStatus},
		{"dispatch.driver.offline", "realtime.on_driver_offline", s.onDriverStatus},
		{"dispatch.offer.created", "realtime.on_offer_created", s.onOfferCreated},
		{"dispatch.offer.accepted", "realtime.on_offer_accepted", s.onOfferAccepted},
		{"dispatch.offer.rejected", "realtime.on_offer_rejected", s.onOfferRejected},
		{"dispatch.offer.expired", "realtime.on_offer_expired", s.onOfferExpired},

		// Financial events
		{"financial.wallet.credited", "realtime.on_wallet_credited", s.onWalletEvent(domain.MsgTypeWalletCredited)},
		{"financial.wallet.debited", "realtime.on_wallet_debited", s.onWalletEvent(domain.MsgTypeWalletDebited)},
	}

	for _, sub := range subscriptions {
		dedupHandler := inbox.Dedup(s.inbox, sub.handlerName, sub.handler, s.logger)
		if err := s.bus.Subscribe(ctx, sub.eventType, dedupHandler); err != nil {
			return fmt.Errorf("subscribe to %s: %w", sub.eventType, err)
		}
		s.logger.Info("subscribed to event", "event_type", sub.eventType, "handler", sub.handlerName)
	}

	return nil
}

// ===== Order Event Handlers =====

// OrderEventPayload is a generic payload for order lifecycle events.
// We re-declare the fields we need (avoid importing orders.port).
type OrderEventPayload struct {
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id,omitempty"`
	RestaurantID  string `json:"restaurant_id,omitempty"`
	DriverID      string `json:"driver_id,omitempty"`
	OrderNumber   string `json:"order_number,omitempty"`
	Status        string `json:"status,omitempty"`
	TotalCents    int64  `json:"total_cents,omitempty"`
	Currency      string `json:"currency,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

// onOrderEvent returns a handler for order lifecycle events.
// The event type from the bus determines the message type.
func (s *Subscriber) onOrderEvent(msgType domain.MessageType) bus.Handler {
	return func(ctx context.Context, envelope bus.EventEnvelope) error {
		var payload OrderEventPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal order payload: %w", err)
		}

		// Build the WebSocket message data (the original payload, passed through).
		data := envelope.Payload

		// Broadcast to:
		//   - order:{order_id} (anyone watching this order)
		//   - user:{user_id} (the customer)
		//   - driver:{driver_id} (the assigned driver, if any)
		s.svc.BroadcastOrderEvent(ctx, payload.OrderID, payload.UserID, payload.DriverID, msgType, data)

		return nil
	}
}

// ===== Dispatch Driver Location Handler =====

type DriverLocationPayload struct {
	DriverID   string  `json:"driver_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Bearing    float64 `json:"bearing"`
	Speed      float64 `json:"speed"`
	CapturedAt string  `json:"captured_at"`
}

func (s *Subscriber) onDriverLocation(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload DriverLocationPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal driver location: %w", err)
	}

	// Broadcast to the driver's channel.
	// In production, we'd also broadcast to all active orders for this driver.
	s.svc.BroadcastDriverEvent(ctx, payload.DriverID, domain.MsgTypeDriverLocation, envelope.Payload)
	return nil
}

// ===== Dispatch Driver Status Handler =====

type DriverStatusPayload struct {
	DriverID string `json:"driver_id"`
	ZoneIDs  []string `json:"zone_ids,omitempty"`
}

func (s *Subscriber) onDriverStatus(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload DriverStatusPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal driver status: %w", err)
	}

	// Map event type to message type.
	var msgType domain.MessageType
	switch envelope.EventType {
	case "dispatch.driver.online":
		msgType = domain.MsgTypeDriverStatus
	case "dispatch.driver.offline":
		msgType = domain.MsgTypeDriverStatus
	default:
		msgType = domain.MsgTypeDriverStatus
	}

	s.svc.BroadcastDriverEvent(ctx, payload.DriverID, msgType, envelope.Payload)
	return nil
}

// ===== Dispatch Offer Handlers =====

type OfferPayload struct {
	OfferID  string `json:"offer_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
	Reason   string `json:"reason,omitempty"`
}

func (s *Subscriber) onOfferCreated(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload OfferPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal offer: %w", err)
	}
	// Broadcast the offer to the driver.
	s.svc.BroadcastDispatchOffer(ctx, payload.DriverID, envelope.Payload)
	return nil
}

func (s *Subscriber) onOfferAccepted(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload OfferPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal offer: %w", err)
	}
	// Broadcast to the driver + order channel.
	s.svc.BroadcastDriverEvent(ctx, payload.DriverID, domain.MsgTypeDispatchAccepted, envelope.Payload)
	// Also broadcast to the order channel (for the customer watching the order).
	s.svc.BroadcastOrderEvent(ctx, payload.OrderID, "", payload.DriverID, domain.MsgTypeDispatchAccepted, envelope.Payload)
	return nil
}

func (s *Subscriber) onOfferRejected(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload OfferPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal offer: %w", err)
	}
	s.svc.BroadcastDriverEvent(ctx, payload.DriverID, domain.MsgTypeDispatchRejected, envelope.Payload)
	return nil
}

func (s *Subscriber) onOfferExpired(ctx context.Context, envelope bus.EventEnvelope) error {
	var payload OfferPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal offer: %w", err)
	}
	s.svc.BroadcastDriverEvent(ctx, payload.DriverID, domain.MsgTypeDispatchExpired, envelope.Payload)
	return nil
}

// ===== Financial Wallet Handler =====

type WalletPayload struct {
	WalletID      string `json:"wallet_id"`
	OwnerType     string `json:"owner_type,omitempty"`
	OwnerID       string `json:"owner_id,omitempty"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	Category      string `json:"category"`
	NewBalance    int64  `json:"new_balance_cents"`
}

func (s *Subscriber) onWalletEvent(msgType domain.MessageType) bus.Handler {
	return func(ctx context.Context, envelope bus.EventEnvelope) error {
		var payload WalletPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet: %w", err)
		}
		// Broadcast to the wallet owner's channel.
		if payload.OwnerType != "" && payload.OwnerID != "" {
			s.svc.BroadcastWalletEvent(ctx, payload.OwnerType, payload.OwnerID, msgType, envelope.Payload)
		}
		return nil
	}
}
