// Package domain tests: Order aggregate root — lifecycle, invariants, cancellation.
package domain

import (
	"errors"
	"testing"
	"time"
)

var orderNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func validOrderParams() OrderParams {
	price, _ := NewMoney(1299, "EGP")
	item, _ := NewOrderItem("item-001", "Burger", "برجر", price, 2)
	delivery, _ := NewDeliveryInfo(30.05, 31.36, "Nasr City, Cairo", "Apt 3")
	subtotal, _ := NewMoney(2598, "EGP") // 1299 * 2
	deliveryFee, _ := NewMoney(399, "EGP")
	total, _ := NewMoney(2997, "EGP")
	dist := 1500

	return OrderParams{
		ID:             "order-001",
		OrderNumber:    "AVEX-20260115-00001",
		UserID:         "user-001",
		RestaurantID:   "rest-001",
		CustomerName:   "Ahmed Ali",
		CustomerPhone:  "01012345678",
		DeliveryInfo:   delivery,
		Items:          []OrderItem{item},
		Subtotal:       subtotal,
		DeliveryFee:    deliveryFee,
		Discount:       ZeroMoney("EGP"),
		Tax:            ZeroMoney("EGP"),
		Total:          total,
		PaymentMethod:  PaymentCash,
		ZoneID:         "zone-nasr",
		DeliveryDistM:  &dist,
		IdempotencyKey: "idem-key-001",
		Now:            orderNow,
	}
}

// ===== Constructor Tests =====

func TestNewOrder_Success(t *testing.T) {
	order, err := NewOrder(validOrderParams())
	if err != nil {
		t.Fatalf("NewOrder error: %v", err)
	}
	if order.Status() != StatusPending {
		t.Errorf("Status = %q, want 'pending'", order.Status())
	}
	if order.ItemsCount() != 2 {
		t.Errorf("ItemsCount = %d, want 2", order.ItemsCount())
	}
	if !order.IsPending() {
		t.Error("new order should be pending")
	}
	if order.IdempotencyKey() != "idem-key-001" {
		t.Errorf("IdempotencyKey = %q", order.IdempotencyKey())
	}
}

func TestNewOrder_EmptyID(t *testing.T) {
	p := validOrderParams()
	p.ID = ""
	_, err := NewOrder(p)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("error = %v, want ErrInvalidID", err)
	}
}

func TestNewOrder_EmptyOrderNumber(t *testing.T) {
	p := validOrderParams()
	p.OrderNumber = ""
	_, err := NewOrder(p)
	if !errors.Is(err, ErrInvalidOrderNumber) {
		t.Errorf("error = %v, want ErrInvalidOrderNumber", err)
	}
}

func TestNewOrder_EmptyUserID(t *testing.T) {
	p := validOrderParams()
	p.UserID = ""
	_, err := NewOrder(p)
	if !errors.Is(err, ErrUserIDRequired) {
		t.Errorf("error = %v, want ErrUserIDRequired", err)
	}
}

func TestNewOrder_EmptyRestaurantID(t *testing.T) {
	p := validOrderParams()
	p.RestaurantID = ""
	_, err := NewOrder(p)
	if !errors.Is(err, ErrRestaurantIDRequired) {
		t.Errorf("error = %v, want ErrRestaurantIDRequired", err)
	}
}

func TestNewOrder_EmptyCustomerName(t *testing.T) {
	p := validOrderParams()
	p.CustomerName = ""
	_, err := NewOrder(p)
	if !errors.Is(err, ErrCustomerNameRequired) {
		t.Errorf("error = %v, want ErrCustomerNameRequired", err)
	}
}

func TestNewOrder_EmptyItems(t *testing.T) {
	p := validOrderParams()
	p.Items = []OrderItem{}
	_, err := NewOrder(p)
	if !errors.Is(err, ErrEmptyOrderItems) {
		t.Errorf("error = %v, want ErrEmptyOrderItems", err)
	}
}

func TestNewOrder_InvalidPaymentMethod(t *testing.T) {
	p := validOrderParams()
	p.PaymentMethod = PaymentMethod("bitcoin")
	_, err := NewOrder(p)
	if !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Errorf("error = %v, want ErrInvalidPaymentMethod", err)
	}
}

func TestNewOrder_CurrencyMismatch(t *testing.T) {
	p := validOrderParams()
	// Change an item's currency to USD (mismatch with EGP subtotal).
	usdPrice, _ := NewMoney(500, "USD")
	usdItem, _ := NewOrderItem("item-002", "Fries", "بطاطس", usdPrice, 1)
	p.Items = append(p.Items, usdItem)
	_, err := NewOrder(p)
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("error = %v, want ErrCurrencyMismatch", err)
	}
}

// ===== Lifecycle Tests =====

func TestOrder_FullLifecycle(t *testing.T) {
	order, _ := NewOrder(validOrderParams())

	// pending → confirmed
	if err := order.Confirm(orderNow.Add(1 * time.Minute)); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if !order.IsConfirmed() {
		t.Error("should be confirmed")
	}

	// confirmed → preparing
	if err := order.StartPreparing(orderNow.Add(2 * time.Minute)); err != nil {
		t.Fatalf("StartPreparing: %v", err)
	}
	if !order.IsPreparing() {
		t.Error("should be preparing")
	}

	// preparing → ready_for_pickup
	if err := order.MarkReadyForPickup(orderNow.Add(15 * time.Minute)); err != nil {
		t.Fatalf("MarkReadyForPickup: %v", err)
	}
	if !order.IsReadyForPickup() {
		t.Error("should be ready for pickup")
	}

	// ready_for_pickup → dispatching
	if err := order.StartDispatch(orderNow.Add(16 * time.Minute)); err != nil {
		t.Fatalf("StartDispatch: %v", err)
	}
	if !order.IsDispatching() {
		t.Error("should be dispatching")
	}

	// dispatching → assigned
	if err := order.AssignDriver("driver-001", orderNow.Add(16*time.Minute+15*time.Second)); err != nil {
		t.Fatalf("AssignDriver: %v", err)
	}
	if !order.IsAssigned() {
		t.Error("should be assigned")
	}
	if !order.HasDriver() {
		t.Error("should have driver")
	}
	if order.DriverID() != "driver-001" {
		t.Errorf("DriverID = %q", order.DriverID())
	}

	// assigned → picked_up
	if err := order.MarkPickedUp("photo-url", orderNow.Add(20*time.Minute)); err != nil {
		t.Fatalf("MarkPickedUp: %v", err)
	}
	if !order.IsPickedUp() {
		t.Error("should be picked up")
	}

	// picked_up → delivered
	if err := order.MarkDelivered("delivery-photo", orderNow.Add(35*time.Minute)); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}
	if !order.IsDelivered() {
		t.Error("should be delivered")
	}
	if !order.IsTerminal() {
		t.Error("delivered should be terminal")
	}
}

func TestOrder_Confirm_FromNonPending(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))

	// Try to confirm again — should fail.
	err := order.Confirm(orderNow.Add(2 * time.Minute))
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestOrder_StartPreparing_FromPending(t *testing.T) {
	order, _ := NewOrder(validOrderParams())

	// pending → preparing should fail (must go through confirmed first).
	err := order.StartPreparing(orderNow.Add(1 * time.Minute))
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestOrder_RetryDispatch(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	_ = order.StartPreparing(orderNow.Add(2 * time.Minute))
	_ = order.MarkReadyForPickup(orderNow.Add(15 * time.Minute))
	_ = order.StartDispatch(orderNow.Add(16 * time.Minute))

	// dispatching → ready_for_pickup (retry)
	err := order.RetryDispatch(orderNow.Add(17 * time.Minute))
	if err != nil {
		t.Fatalf("RetryDispatch: %v", err)
	}
	if !order.IsReadyForPickup() {
		t.Error("should be back to ready_for_pickup")
	}
}

func TestOrder_AssignDriver_EmptyDriverID(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	_ = order.StartPreparing(orderNow.Add(2 * time.Minute))
	_ = order.MarkReadyForPickup(orderNow.Add(15 * time.Minute))
	_ = order.StartDispatch(orderNow.Add(16 * time.Minute))

	err := order.AssignDriver("", orderNow.Add(17*time.Minute))
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}

// ===== Cancellation Tests =====

func TestOrder_Cancel_FromPending(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	err := order.Cancel("user", "changed my mind", orderNow.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("Cancel from pending: %v", err)
	}
	if !order.IsCancelled() {
		t.Error("should be cancelled")
	}
	if order.CancelledBy() != "user" {
		t.Errorf("CancelledBy = %q", order.CancelledBy())
	}
	if order.CancelReason() != "changed my mind" {
		t.Errorf("CancelReason = %q", order.CancelReason())
	}
}

func TestOrder_Cancel_FromConfirmed(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	err := order.Cancel("user", "found better option", orderNow.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Cancel from confirmed: %v", err)
	}
}

func TestOrder_Cancel_FromPreparing(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	_ = order.StartPreparing(orderNow.Add(2 * time.Minute))
	err := order.Cancel("support", "kitchen fire", orderNow.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Cancel from preparing: %v", err)
	}
}

func TestOrder_Cancel_FromAssigned(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	_ = order.StartPreparing(orderNow.Add(2 * time.Minute))
	_ = order.MarkReadyForPickup(orderNow.Add(15 * time.Minute))
	_ = order.StartDispatch(orderNow.Add(16 * time.Minute))
	_ = order.AssignDriver("driver-001", orderNow.Add(17*time.Minute))

	err := order.Cancel("support", "driver no-show", orderNow.Add(20*time.Minute))
	if err != nil {
		t.Fatalf("Cancel from assigned: %v", err)
	}
}

func TestOrder_Cancel_FromDelivered(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Confirm(orderNow.Add(1 * time.Minute))
	_ = order.StartPreparing(orderNow.Add(2 * time.Minute))
	_ = order.MarkReadyForPickup(orderNow.Add(15 * time.Minute))
	_ = order.StartDispatch(orderNow.Add(16 * time.Minute))
	_ = order.AssignDriver("driver-001", orderNow.Add(17*time.Minute))
	_ = order.MarkPickedUp("photo", orderNow.Add(20*time.Minute))
	_ = order.MarkDelivered("photo2", orderNow.Add(35*time.Minute))

	err := order.Cancel("user", "want refund", orderNow.Add(40*time.Minute))
	if !errors.Is(err, ErrOrderAlreadyDelivered) {
		t.Errorf("error = %v, want ErrOrderAlreadyDelivered", err)
	}
}

func TestOrder_Cancel_AlreadyCancelled(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	_ = order.Cancel("user", "changed mind", orderNow.Add(1*time.Minute))

	err := order.Cancel("user", "again", orderNow.Add(2*time.Minute))
	if !errors.Is(err, ErrOrderAlreadyCancelled) {
		t.Errorf("error = %v, want ErrOrderAlreadyCancelled", err)
	}
}

func TestOrder_Cancel_EmptyReason(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	err := order.Cancel("user", "", orderNow.Add(1*time.Minute))
	if !errors.Is(err, ErrCancelReasonRequired) {
		t.Errorf("error = %v, want ErrCancelReasonRequired", err)
	}
}

// ===== Dispatch Info Tests =====

func TestOrder_SetDispatchDistance(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	order.SetDispatchDistance(800, orderNow.Add(1*time.Minute))

	if order.Dispatch().DispatchDistance() != 800 {
		t.Errorf("DispatchDistance = %d, want 800", order.Dispatch().DispatchDistance())
	}
}

func TestOrder_DispatchInfo_Initially(t *testing.T) {
	order, _ := NewOrder(validOrderParams())
	if order.HasDriver() {
		t.Error("new order should not have a driver")
	}
	if order.DriverID() != "" {
		t.Errorf("DriverID = %q, should be empty", order.DriverID())
	}
	if order.Dispatch().ZoneID() != "zone-nasr" {
		t.Errorf("ZoneID = %q", order.Dispatch().ZoneID())
	}
}

// ===== ReconstructOrder Test =====

func TestReconstructOrder(t *testing.T) {
	price, _ := NewMoney(1299, "EGP")
	item, _ := NewOrderItem("item-001", "Burger", "برجر", price, 2)
	subtotal, _ := NewMoney(2598, "EGP")
	total, _ := NewMoney(2997, "EGP")
	delivery, _ := NewDeliveryInfo(30.05, 31.36, "Cairo", "")

	rec := OrderRecord{
		ID:            "order-rec",
		OrderNumber:   "AVEX-001",
		UserID:        "user-1",
		RestaurantID:  "rest-1",
		CustomerName:  "Test",
		CustomerPhone: "01012345678",
		DeliveryInfo:  delivery,
		Items:         []OrderItem{item},
		Subtotal:      subtotal,
		Total:         total,
		PaymentMethod: PaymentCash,
		Status:        StatusDelivered,
		CreatedAt:     orderNow,
		UpdatedAt:     orderNow,
	}
	order := ReconstructOrder(rec)
	if order.ID() != "order-rec" {
		t.Errorf("ID = %q", order.ID())
	}
	if !order.IsDelivered() {
		t.Error("should be delivered")
	}
}
