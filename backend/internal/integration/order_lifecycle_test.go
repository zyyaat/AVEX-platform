//go:build integration

package integration_test

import (
	"context"
	"testing"

	ordersport "avex-backend/internal/modules/orders/port"
)

// TestOrderLifecycle_FullFlow tests the complete order lifecycle:
//   create → confirm → prepare → ready → assign driver → pickup → deliver
func TestOrderLifecycle_FullFlow(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create order
	order, err := ordersMod.Service().CreateOrder(ctx, ordersport.CreateOrderInput{
		UserID:        "user-int-1",
		RestaurantID:  "rest-int-1",
		CustomerName:  "Integration Test User",
		CustomerPhone: "+201234567890",
		DeliveryLat:   30.0444,
		DeliveryLng:   31.2357,
		DeliveryAddress: "Cairo, Egypt",
		RestaurantLat: 30.0500,
		RestaurantLng: 31.2400,
		Items: []ordersport.CreateOrderItemInput{
			{MenuItemID: "item-1", Name: "Pizza", PriceCents: 5000, Quantity: 2},
		},
		SubtotalCents:    10000,
		DeliveryFeeCents: 300,
		DiscountCents:    0,
		TaxCents:         1400,
		TotalCents:       11700,
		Currency:         "EGP",
		PaymentMethod:    "cash",
		ZoneID:           "zone-cairo",
		IdempotencyKey:   "int-test-order-1",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if order.Status != "pending" {
		t.Fatalf("expected pending, got %s", order.Status)
	}
	t.Logf("Order created: %s (number: %s)", order.ID, order.OrderNumber)

	// 2. Confirm order (merchant)
	order, err = ordersMod.Service().ConfirmOrder(ctx, order.ID, "merchant-int-1")
	if err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	if order.Status != "confirmed" {
		t.Fatalf("expected confirmed, got %s", order.Status)
	}

	// 3. Start preparing
	order, err = ordersMod.Service().StartPreparing(ctx, order.ID, "merchant-int-1")
	if err != nil {
		t.Fatalf("StartPreparing: %v", err)
	}
	if order.Status != "preparing" {
		t.Fatalf("expected preparing, got %s", order.Status)
	}

	// 4. Mark ready for pickup
	order, err = ordersMod.Service().MarkReadyForPickup(ctx, order.ID, "merchant-int-1")
	if err != nil {
		t.Fatalf("MarkReadyForPickup: %v", err)
	}
	if order.Status != "ready_for_pickup" {
		t.Fatalf("expected ready_for_pickup, got %s", order.Status)
	}

	// 5. Assign driver (simulating dispatch acceptance)
	order, err = ordersMod.Service().AssignDriver(ctx, ordersport.AssignDriverInput{
		OrderID:  order.ID,
		DriverID: "driver-int-1",
	})
	if err != nil {
		t.Fatalf("AssignDriver: %v", err)
	}
	if order.Status != "assigned" {
		t.Fatalf("expected assigned, got %s", order.Status)
	}
	if order.DriverID != "driver-int-1" {
		t.Fatalf("expected driver-int-1, got %s", order.DriverID)
	}

	// 6. Pick up
	order, err = ordersMod.Service().MarkPickedUp(ctx, ordersport.MarkPickedUpInput{
		OrderID:        order.ID,
		DriverID:       "driver-int-1",
		PickupPhotoURL: "https://example.com/pickup.jpg",
	})
	if err != nil {
		t.Fatalf("MarkPickedUp: %v", err)
	}
	if order.Status != "picked_up" {
		t.Fatalf("expected picked_up, got %s", order.Status)
	}

	// 7. Deliver
	order, err = ordersMod.Service().MarkDelivered(ctx, ordersport.MarkDeliveredInput{
		OrderID:          order.ID,
		DriverID:         "driver-int-1",
		DeliveryPhotoURL: "https://example.com/delivery.jpg",
	})
	if err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}
	if order.Status != "delivered" {
		t.Fatalf("expected delivered, got %s", order.Status)
	}

	t.Logf("Order %s delivered successfully", order.OrderNumber)
}

// TestOrderLifecycle_ParallelDispatch tests the parallel dispatch flow:
//   create → assign driver while order is still pending
func TestOrderLifecycle_ParallelDispatch(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create order (status = pending)
	order, err := ordersMod.Service().CreateOrder(ctx, ordersport.CreateOrderInput{
		UserID:        "user-int-2",
		RestaurantID:  "rest-int-2",
		CustomerName:  "Parallel Test User",
		CustomerPhone: "+201234567891",
		DeliveryLat:   30.0444,
		DeliveryLng:   31.2357,
		DeliveryAddress: "Cairo, Egypt",
		RestaurantLat: 30.0500,
		RestaurantLng: 31.2400,
		Items: []ordersport.CreateOrderItemInput{
			{MenuItemID: "item-2", Name: "Burger", PriceCents: 3000, Quantity: 1},
		},
		SubtotalCents:    3000,
		DeliveryFeeCents: 300,
		DiscountCents:    0,
		TaxCents:         420,
		TotalCents:       3720,
		Currency:         "EGP",
		PaymentMethod:    "card",
		ZoneID:           "zone-cairo",
		IdempotencyKey:   "int-test-order-parallel",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if order.Status != "pending" {
		t.Fatalf("expected pending, got %s", order.Status)
	}

	// 2. Assign driver WHILE order is still pending (parallel dispatch)
	order, err = ordersMod.Service().AssignDriver(ctx, ordersport.AssignDriverInput{
		OrderID:  order.ID,
		DriverID: "driver-int-2",
	})
	if err != nil {
		t.Fatalf("AssignDriver from pending: %v", err)
	}
	if order.Status != "assigned" {
		t.Fatalf("expected assigned (parallel dispatch), got %s", order.Status)
	}
	if order.DriverID != "driver-int-2" {
		t.Fatalf("expected driver-int-2, got %s", order.DriverID)
	}

	t.Logf("Parallel dispatch works: order assigned while still pending")
}

// TestOrderLifecycle_CancelFromPending tests order cancellation.
func TestOrderLifecycle_CancelFromPending(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	order, err := ordersMod.Service().CreateOrder(ctx, ordersport.CreateOrderInput{
		UserID:        "user-int-3",
		RestaurantID:  "rest-int-3",
		CustomerName:  "Cancel Test User",
		CustomerPhone: "+201234567892",
		DeliveryLat:   30.0444,
		DeliveryLng:   31.2357,
		DeliveryAddress: "Cairo, Egypt",
		RestaurantLat: 30.0500,
		RestaurantLng: 31.2400,
		Items: []ordersport.CreateOrderItemInput{
			{MenuItemID: "item-3", Name: "Salad", PriceCents: 2000, Quantity: 1},
		},
		SubtotalCents:    2000,
		DeliveryFeeCents: 300,
		DiscountCents:    0,
		TaxCents:         280,
		TotalCents:       2580,
		Currency:         "EGP",
		PaymentMethod:    "cash",
		ZoneID:           "zone-cairo",
		IdempotencyKey:   "int-test-order-cancel",
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Cancel
	order, err = ordersMod.Service().CancelOrder(ctx, ordersport.CancelOrderInput{
		OrderID:     order.ID,
		CancelledBy: "user",
		Reason:      "Changed my mind",
	})
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if order.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %s", order.Status)
	}
}

// TestOrderLifecycle_Idempotency tests that creating an order with the same
// idempotency key returns the same order.
func TestOrderLifecycle_Idempotency(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	input := ordersport.CreateOrderInput{
		UserID:        "user-int-4",
		RestaurantID:  "rest-int-4",
		CustomerName:  "Idempotency Test User",
		CustomerPhone: "+201234567893",
		DeliveryLat:   30.0444,
		DeliveryLng:   31.2357,
		DeliveryAddress: "Cairo, Egypt",
		RestaurantLat: 30.0500,
		RestaurantLng: 31.2400,
		Items: []ordersport.CreateOrderItemInput{
			{MenuItemID: "item-4", Name: "Pasta", PriceCents: 4000, Quantity: 1},
		},
		SubtotalCents:    4000,
		DeliveryFeeCents: 300,
		DiscountCents:    0,
		TaxCents:         560,
		TotalCents:       4860,
		Currency:         "EGP",
		PaymentMethod:    "wallet",
		ZoneID:           "zone-cairo",
		IdempotencyKey:   "int-test-idempotency-key",
	}

	// First call
	order1, err := ordersMod.Service().CreateOrder(ctx, input)
	if err != nil {
		t.Fatalf("first CreateOrder: %v", err)
	}

	// Second call with same idempotency key — should return same order
	order2, err := ordersMod.Service().CreateOrder(ctx, input)
	if err != nil {
		t.Fatalf("second CreateOrder: %v", err)
	}
	if order1.ID != order2.ID {
		t.Fatalf("idempotency failed: %s != %s", order1.ID, order2.ID)
	}

	t.Logf("Idempotency works: same order returned (%s)", order1.ID)
}
