// Package jobs subscriber: listens to bus events and creates notifications.
//
// Subscribes to events from orders, dispatch, and financial modules.
// For each event, creates a notification via the service layer.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"avex-backend/internal/modules/notifications/port"
	"avex-backend/internal/platform/bus"
	"avex-backend/internal/platform/inbox"
)

type Subscriber struct {
	svc    port.ServicePort
	bus    bus.Subscriber
	inbox  inbox.Inbox
	logger *slog.Logger
}

func NewSubscriber(svc port.ServicePort, bus bus.Subscriber, inbox inbox.Inbox, logger *slog.Logger) *Subscriber {
	return &Subscriber{svc: svc, bus: bus, inbox: inbox, logger: logger}
}

func (s *Subscriber) Start(ctx context.Context) error {
	subs := []struct {
		eventType   string
		handlerName string
		handler     bus.Handler
	}{
		// Orders events
		{"orders.order.created", "notifications.on_order_created", s.onOrderCreated},
		{"orders.order.confirmed", "notifications.on_order_confirmed", s.onOrderConfirmed},
		{"orders.order.preparing", "notifications.on_order_preparing", s.onOrderPreparing},
		{"orders.order.ready_for_pickup", "notifications.on_order_ready", s.onOrderReady},
		{"orders.order.assigned", "notifications.on_order_assigned", s.onOrderAssigned},
		{"orders.order.picked_up", "notifications.on_order_picked_up", s.onOrderPickedUp},
		{"orders.order.delivered", "notifications.on_order_delivered", s.onOrderDelivered},
		{"orders.order.cancelled", "notifications.on_order_cancelled", s.onOrderCancelled},

		// Dispatch events
		{"dispatch.offer.created", "notifications.on_dispatch_offer", s.onDispatchOffer},
		{"dispatch.offer.accepted", "notifications.on_dispatch_accepted", s.onDispatchAccepted},

		// Financial events
		{"financial.wallet.credited", "notifications.on_wallet_credited", s.onWalletCredited},
		{"financial.wallet.debited", "notifications.on_wallet_debited", s.onWalletDebited},
		{"financial.promotion.redeemed", "notifications.on_promo_redeemed", s.onPromoRedeemed},
	}

	for _, sub := range subs {
		dedupHandler := inbox.Dedup(s.inbox, sub.handlerName, sub.handler, s.logger)
		if err := s.bus.Subscribe(ctx, sub.eventType, dedupHandler); err != nil {
			return fmt.Errorf("subscribe to %s: %w", sub.eventType, err)
		}
		s.logger.Info("subscribed to event", "event_type", sub.eventType, "handler", sub.handlerName)
	}
	return nil
}

// ===== Helper: send a notification =====

func (s *Subscriber) send(ctx context.Context, recipientType, recipientID, notifType, title, body string, data map[string]any) {
	_, err := s.svc.SendNotification(ctx, port.SendNotificationInput{
		RecipientType: recipientType,
		RecipientID:   recipientID,
		Type:          notifType,
		Channels:      []string{"push", "sms"},
		Title:         title,
		Body:          body,
		Data:          data,
		Priority:      "normal",
	})
	if err != nil {
		s.logger.Debug("notification send skipped",
			"recipient", recipientID,
			"type", notifType,
			"error", err,
		)
	}
}

// ===== Order Event Handlers =====

type OrderPayload struct {
	OrderID      string `json:"order_id"`
	OrderNumber  string `json:"order_number"`
	UserID       string `json:"user_id"`
	RestaurantID string `json:"restaurant_id"`
	TotalCents   int64  `json:"total_cents"`
	Currency     string `json:"currency"`
}

func (s *Subscriber) onOrderCreated(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	// Notify the user that their order was placed
	s.send(ctx, "user", p.UserID, "order_created",
		"Order Placed",
		fmt.Sprintf("Your order %s has been placed successfully.", p.OrderNumber),
		map[string]any{"order_id": p.OrderID, "order_number": p.OrderNumber},
	)
	return nil
}

func (s *Subscriber) onOrderConfirmed(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "order_confirmed",
		"Order Confirmed",
		fmt.Sprintf("The restaurant confirmed your order %s.", p.OrderNumber),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onOrderPreparing(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "order_preparing",
		"Order Being Prepared",
		fmt.Sprintf("Your order %s is now being prepared.", p.OrderNumber),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onOrderReady(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "order_ready",
		"Order Ready for Pickup",
		fmt.Sprintf("Your order %s is ready. A driver is on the way.", p.OrderNumber),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onOrderAssigned(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		OrderID  string `json:"order_id"`
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	// Notify the user that a driver has been assigned
	s.send(ctx, "user", "", "order_assigned",
		"Driver Assigned",
		"A driver has been assigned to your order.",
		map[string]any{"order_id": p.OrderID, "driver_id": p.DriverID},
	)
	return nil
}

func (s *Subscriber) onOrderPickedUp(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "order_picked_up",
		"Order Picked Up",
		fmt.Sprintf("Your order %s has been picked up and is on the way.", p.OrderNumber),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onOrderDelivered(ctx context.Context, envelope bus.EventEnvelope) error {
	var p OrderPayload
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "order_delivered",
		"Order Delivered",
		fmt.Sprintf("Your order %s has been delivered. Enjoy!", p.OrderNumber),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onOrderCancelled(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		OrderID     string `json:"order_id"`
		CancelledBy string `json:"cancelled_by"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", "", "order_cancelled",
		"Order Cancelled",
		fmt.Sprintf("Your order has been cancelled. Reason: %s", p.Reason),
		map[string]any{"order_id": p.OrderID},
	)
	return nil
}

// ===== Dispatch Event Handlers =====

func (s *Subscriber) onDispatchOffer(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		OfferID  string `json:"offer_id"`
		OrderID  string `json:"order_id"`
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "driver", p.DriverID, "dispatch_offer",
		"New Delivery Offer",
		"You have a new delivery offer. Tap to accept.",
		map[string]any{"offer_id": p.OfferID, "order_id": p.OrderID},
	)
	return nil
}

func (s *Subscriber) onDispatchAccepted(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		OfferID  string `json:"offer_id"`
		OrderID  string `json:"order_id"`
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "driver", p.DriverID, "dispatch_assigned",
		"Offer Accepted",
		"You accepted the offer. Head to the restaurant to pick up the order.",
		map[string]any{"offer_id": p.OfferID, "order_id": p.OrderID},
	)
	return nil
}

// ===== Financial Event Handlers =====

func (s *Subscriber) onWalletCredited(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		WalletID    string `json:"wallet_id"`
		OwnerType   string `json:"owner_type"`
		OwnerID     string `json:"owner_id"`
		AmountCents int64  `json:"amount_cents"`
		Currency    string `json:"currency"`
		NewBalance  int64  `json:"new_balance_cents"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	if p.OwnerType == "" || p.OwnerID == "" {
		return nil
	}
	s.send(ctx, p.OwnerType, p.OwnerID, "wallet_credited",
		"Wallet Credited",
		fmt.Sprintf("Your wallet has been credited. New balance: %d %s", p.NewBalance, p.Currency),
		map[string]any{"amount_cents": p.AmountCents, "currency": p.Currency},
	)
	return nil
}

func (s *Subscriber) onWalletDebited(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		WalletID    string `json:"wallet_id"`
		OwnerType   string `json:"owner_type"`
		OwnerID     string `json:"owner_id"`
		AmountCents int64  `json:"amount_cents"`
		Currency    string `json:"currency"`
		NewBalance  int64  `json:"new_balance_cents"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	if p.OwnerType == "" || p.OwnerID == "" {
		return nil
	}
	s.send(ctx, p.OwnerType, p.OwnerID, "wallet_debited",
		"Wallet Debited",
		fmt.Sprintf("Your wallet has been debited. New balance: %d %s", p.NewBalance, p.Currency),
		map[string]any{"amount_cents": p.AmountCents, "currency": p.Currency},
	)
	return nil
}

func (s *Subscriber) onPromoRedeemed(ctx context.Context, envelope bus.EventEnvelope) error {
	var p struct {
		RedemptionID  string `json:"redemption_id"`
		PromotionID   string `json:"promotion_id"`
		PromotionCode string `json:"promotion_code"`
		UserID        string `json:"user_id"`
		OrderID       string `json:"order_id"`
		DiscountCents int64  `json:"discount_cents"`
		Currency      string `json:"currency"`
	}
	if err := json.Unmarshal(envelope.Payload, &p); err != nil {
		return err
	}
	s.send(ctx, "user", p.UserID, "promotion_redeemed",
		"Promotion Applied",
		fmt.Sprintf("Promo code %s applied. You saved %d %s.", p.PromotionCode, p.DiscountCents, p.Currency),
		map[string]any{"promotion_code": p.PromotionCode, "discount_cents": p.DiscountCents},
	)
	return nil
}
