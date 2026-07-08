// Package jobs subscriber: subscribes to orders events and triggers dispatch.
//
// The subscriber listens on the Redis bus for:
//   - orders.order.assignment_requested → triggers CreateOfferInternal
//   - orders.order.cancelled → cancels any pending offer for that order
//
// All handlers are wrapped with the inbox pattern for idempotency.
package jobs

import (
        "context"
        "encoding/json"
        "fmt"
        "log/slog"
        "time"

        "avex-backend/internal/modules/dispatch/port"
        "avex-backend/internal/platform/bus"
        "avex-backend/internal/platform/inbox"
)

// Subscriber listens to orders events and dispatches them to the service.
type Subscriber struct {
        svc       port.ServicePort
        bus       bus.Subscriber
        inbox     inbox.Inbox
        logger    *slog.Logger
        handlerName string // e.g. "dispatch.on_order_assignment_requested"
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(svc port.ServicePort, bus bus.Subscriber, inbox inbox.Inbox, logger *slog.Logger) *Subscriber {
        return &Subscriber{
                svc:         svc,
                bus:         bus,
                inbox:       inbox,
                logger:      logger,
                handlerName: "dispatch.on_order_assignment_requested",
        }
}

// Start subscribes to all relevant event types. Blocks until ctx is cancelled.
func (s *Subscriber) Start(ctx context.Context) error {
        // Subscribe to orders.order.assignment_requested
        assignmentHandler := inbox.Dedup(s.inbox, s.handlerName, s.handleAssignmentRequested, s.logger)
        if err := s.bus.Subscribe(ctx, "orders.order.assignment_requested", assignmentHandler); err != nil {
                return fmt.Errorf("subscribe to assignment_requested: %w", err)
        }
        s.logger.Info("subscribed to orders.order.assignment_requested")

        // Subscribe to orders.order.cancelled (cancel any pending dispatch offer)
        cancelHandler := inbox.Dedup(s.inbox, "dispatch.on_order_cancelled", s.handleOrderCancelled, s.logger)
        if err := s.bus.Subscribe(ctx, "orders.order.cancelled", cancelHandler); err != nil {
                return fmt.Errorf("subscribe to order_cancelled: %w", err)
        }
        s.logger.Info("subscribed to orders.order.cancelled")

        return nil
}

// AssignmentRequestedPayload mirrors orders.port.OrderAssignmentRequestedPayload.
// We re-declare it here to avoid importing the orders module (architecture rule).
type AssignmentRequestedPayload struct {
        OrderID       string  `json:"order_id"`
        RestaurantID  string  `json:"restaurant_id"`
        ZoneID        string  `json:"zone_id,omitempty"`
        DeliveryLat   float64 `json:"delivery_lat"`
        DeliveryLng   float64 `json:"delivery_lng"`
        RestaurantLat float64 `json:"restaurant_lat,omitempty"`
        RestaurantLng float64 `json:"restaurant_lng,omitempty"`
}

// OrderCancelledPayload mirrors orders.port.OrderCancelledPayload.
type OrderCancelledPayload struct {
        OrderID     string `json:"order_id"`
        CancelledBy string `json:"cancelled_by"`
        Reason      string `json:"reason"`
        RefundDue   bool   `json:"refund_due"`
}

// handleAssignmentRequested is the handler for orders.order.assignment_requested.
//
// This event is published IMMEDIATELY when an order is created (in parallel
// with merchant confirmation + food prep). The dispatch engine starts looking
// for a driver right away so the driver can travel to the restaurant while
// the food is being cooked.
//
// Pickup point: RestaurantLat/Lng (where the driver picks up the food).
//   - If RestaurantLat/Lng is zero (legacy orders), fall back to DeliveryLat/Lng.
// Dropoff point: DeliveryLat/Lng (where the driver delivers to the customer).
func (s *Subscriber) handleAssignmentRequested(ctx context.Context, envelope bus.EventEnvelope) error {
        var payload AssignmentRequestedPayload
        if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
                return fmt.Errorf("unmarshal payload: %w", err)
        }

        // Determine the pickup point.
        // Prefer the restaurant location (where the driver picks up the food).
        // Fall back to the delivery location if the restaurant location is not set
        // (e.g. legacy orders created before this field was added).
        pickupLat := payload.RestaurantLat
        pickupLng := payload.RestaurantLng
        if pickupLat == 0 || pickupLng == 0 {
                pickupLat = payload.DeliveryLat
                pickupLng = payload.DeliveryLng
                s.logger.Warn("restaurant location not set in assignment request, using delivery location as pickup",
                        "order_id", payload.OrderID,
                )
        }

        s.logger.Info("assignment requested",
                "order_id", payload.OrderID,
                "zone_id", payload.ZoneID,
                "restaurant_id", payload.RestaurantID,
                "pickup_lat", pickupLat,
                "pickup_lng", pickupLng,
                "correlation_id", envelope.CorrelationID,
        )

        if err := s.svc.HandleOrderAssignmentRequested(
                ctx,
                payload.OrderID,
                payload.ZoneID,
                pickupLat, pickupLng,           // pickup = restaurant location
                payload.DeliveryLat, payload.DeliveryLng, // dropoff = customer location
        ); err != nil {
                s.logger.Error("dispatch failed",
                        "order_id", payload.OrderID,
                        "error", err,
                )
                // Don't return error — the inbox has already marked this as processed.
                // We don't want to retry indefinitely for permanent failures (e.g. no drivers available).
        }
        return nil
}

// handleOrderCancelled cancels any pending dispatch offer for the order.
func (s *Subscriber) handleOrderCancelled(ctx context.Context, envelope bus.EventEnvelope) error {
        var payload OrderCancelledPayload
        if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
                return fmt.Errorf("unmarshal payload: %w", err)
        }
        s.logger.Info("order cancelled, cancelling pending offer", "order_id", payload.OrderID)

        // Find the active offer for this order and cancel it.
        // We use the service's internal logic by listing offers and cancelling pending ones.
        offers, err := s.svc.ListOffersByOrder(ctx, payload.OrderID)
        if err != nil {
                return fmt.Errorf("list offers: %w", err)
        }
        for _, offer := range offers {
                if offer.Status == "pending" {
                        if _, err := s.svc.CancelOffer(ctx, offer.ID); err != nil {
                                s.logger.Error("cancel offer failed",
                                        "offer_id", offer.ID,
                                        "order_id", payload.OrderID,
                                        "error", err,
                                )
                        }
                }
        }
        return nil
}

// suppress unused import
var _ = time.Now
