// Package domain order: Order aggregate root.
//
// Order is the central entity of the orders module. It:
//   - Stores identity references (user, restaurant, driver — all soft refs)
//   - Stores a customer snapshot (name, phone, delivery location)
//   - Stores a financial snapshot (subtotal, fees, tax, total — received from Pricing, NOT calculated)
//   - Manages the order lifecycle via a state machine (9 states)
//   - Holds dispatch metadata via the DispatchInfo value object
//   - Supports idempotency for duplicate request protection
//
// Design rules:
//   - Order does NOT calculate prices. It receives them as a snapshot.
//   - Order does NOT manage dispatch logic. It stores dispatch metadata.
//   - Order enforces business invariants (non-empty items, valid transitions).
//   - All mutations go through domain methods (no public setters).
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// Order is the aggregate root for the orders module.
type Order struct {
	id           string
	orderNumber  string
	userID       string
	restaurantID string

	// Customer snapshot (denormalized at order time for audit)
	customerName  string
	customerPhone string
	deliveryInfo  DeliveryInfo

	// Financial snapshot (received from Pricing module, NOT calculated here)
	items         []OrderItem
	subtotal      Money
	deliveryFee   Money
	discount      Money
	tax           Money
	total         Money
	paymentMethod PaymentMethod

	// Lifecycle
	status     OrderStatus
	couponCode *string

	// Dispatch metadata (value object)
	dispatch DispatchInfo

	// Timestamps
	createdAt     time.Time
	updatedAt     time.Time
	confirmedAt   *time.Time
	preparingAt   *time.Time
	readyAt       *time.Time
	dispatchingAt *time.Time
	assignedAt    *time.Time
	pickedUpAt    *time.Time
	deliveredAt   *time.Time
	cancelledAt   *time.Time
	cancelReason  *string
	cancelledBy   *string

	// Idempotency
	idempotencyKey string
}

// ===== Constructor =====

// OrderParams holds the parameters for creating a new order.
// All financial values are snapshots received from the Pricing module.
type OrderParams struct {
	ID             string
	OrderNumber    string
	UserID         string
	RestaurantID   string
	CustomerName   string
	CustomerPhone  string
	DeliveryInfo   DeliveryInfo
	Items          []OrderItem
	Subtotal       Money
	DeliveryFee    Money
	Discount       Money
	Tax            Money
	Total          Money
	PaymentMethod  PaymentMethod
	CouponCode     string
	ZoneID         string
	DeliveryDistM  *int
	IdempotencyKey string
	Now            time.Time
}

// NewOrder creates a new Order in the "pending" status with full validation.
func NewOrder(params OrderParams) (Order, error) {
	if params.ID == "" {
		return Order{}, NewValidationError("id", ErrInvalidID)
	}
	if params.OrderNumber == "" {
		return Order{}, NewValidationError("order_number", ErrInvalidOrderNumber)
	}
	if params.UserID == "" {
		return Order{}, NewValidationError("user_id", ErrUserIDRequired)
	}
	if params.RestaurantID == "" {
		return Order{}, NewValidationError("restaurant_id", ErrRestaurantIDRequired)
	}
	if strings.TrimSpace(params.CustomerName) == "" {
		return Order{}, NewValidationError("customer_name", ErrCustomerNameRequired)
	}
	if params.CustomerPhone == "" {
		return Order{}, NewValidationError("customer_phone", ErrCustomerPhoneRequired)
	}
	if params.DeliveryInfo.IsZero() {
		return Order{}, NewValidationError("delivery_info", ErrDeliveryInfoRequired)
	}
	if len(params.Items) == 0 {
		return Order{}, NewValidationError("items", ErrEmptyOrderItems)
	}
	if !params.PaymentMethod.IsValid() {
		return Order{}, NewValidationError("payment_method", ErrInvalidPaymentMethod)
	}

	// Validate that all items have the same currency as the order totals.
	for i, item := range params.Items {
		if item.Price().Currency() != params.Subtotal.Currency() {
			return Order{}, NewValidationError("items",
				fmt.Errorf("%w: item %d has currency %s, order has %s",
					ErrCurrencyMismatch, i, item.Price().Currency(), params.Subtotal.Currency()))
		}
	}

	// Validate total is non-negative.
	if params.Total.Amount() < 0 {
		return Order{}, NewValidationError("total", ErrInvalidMoneyAmount)
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var coupon *string
	if params.CouponCode != "" {
		c := params.CouponCode
		coupon = &c
	}

	return Order{
		id:             params.ID,
		orderNumber:    params.OrderNumber,
		userID:         params.UserID,
		restaurantID:   params.RestaurantID,
		customerName:   strings.TrimSpace(params.CustomerName),
		customerPhone:  params.CustomerPhone,
		deliveryInfo:   params.DeliveryInfo,
		items:          params.Items,
		subtotal:       params.Subtotal,
		deliveryFee:    params.DeliveryFee,
		discount:       params.Discount,
		tax:            params.Tax,
		total:          params.Total,
		paymentMethod:  params.PaymentMethod,
		status:         StatusPending,
		couponCode:     coupon,
		dispatch:       NewDispatchInfo(params.ZoneID, params.DeliveryDistM),
		createdAt:      now,
		updatedAt:      now,
		idempotencyKey: params.IdempotencyKey,
	}, nil
}

// ===== Reconstruction =====

// OrderRecord holds all fields to rebuild an Order from persistence.
type OrderRecord struct {
	ID             string
	OrderNumber    string
	UserID         string
	RestaurantID   string
	CustomerName   string
	CustomerPhone  string
	DeliveryInfo   DeliveryInfo
	Items          []OrderItem
	Subtotal       Money
	DeliveryFee    Money
	Discount       Money
	Tax            Money
	Total          Money
	PaymentMethod  PaymentMethod
	Status         OrderStatus
	CouponCode     *string
	Dispatch       DispatchInfo
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ConfirmedAt    *time.Time
	PreparingAt    *time.Time
	ReadyAt        *time.Time
	DispatchingAt  *time.Time
	AssignedAt     *time.Time
	PickedUpAt     *time.Time
	DeliveredAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   *string
	CancelledBy    *string
	IdempotencyKey string
}

// ReconstructOrder rebuilds an Order from persistence (no validation).
func ReconstructOrder(rec OrderRecord) Order {
	return Order{
		id:             rec.ID,
		orderNumber:    rec.OrderNumber,
		userID:         rec.UserID,
		restaurantID:   rec.RestaurantID,
		customerName:   rec.CustomerName,
		customerPhone:  rec.CustomerPhone,
		deliveryInfo:   rec.DeliveryInfo,
		items:          rec.Items,
		subtotal:       rec.Subtotal,
		deliveryFee:    rec.DeliveryFee,
		discount:       rec.Discount,
		tax:            rec.Tax,
		total:          rec.Total,
		paymentMethod:  rec.PaymentMethod,
		status:         rec.Status,
		couponCode:     rec.CouponCode,
		dispatch:       rec.Dispatch,
		createdAt:      rec.CreatedAt,
		updatedAt:      rec.UpdatedAt,
		confirmedAt:    rec.ConfirmedAt,
		preparingAt:    rec.PreparingAt,
		readyAt:        rec.ReadyAt,
		dispatchingAt:  rec.DispatchingAt,
		assignedAt:     rec.AssignedAt,
		pickedUpAt:     rec.PickedUpAt,
		deliveredAt:    rec.DeliveredAt,
		cancelledAt:    rec.CancelledAt,
		cancelReason:   rec.CancelReason,
		cancelledBy:    rec.CancelledBy,
		idempotencyKey: rec.IdempotencyKey,
	}
}

// ===== Getters =====

func (o Order) ID() string                   { return o.id }
func (o Order) OrderNumber() string          { return o.orderNumber }
func (o Order) UserID() string               { return o.userID }
func (o Order) RestaurantID() string         { return o.restaurantID }
func (o Order) CustomerName() string         { return o.customerName }
func (o Order) CustomerPhone() string        { return o.customerPhone }
func (o Order) DeliveryInfo() DeliveryInfo   { return o.deliveryInfo }
func (o Order) Items() []OrderItem           { return o.items }
func (o Order) Subtotal() Money              { return o.subtotal }
func (o Order) DeliveryFee() Money           { return o.deliveryFee }
func (o Order) Discount() Money              { return o.discount }
func (o Order) Tax() Money                   { return o.tax }
func (o Order) Total() Money                 { return o.total }
func (o Order) PaymentMethod() PaymentMethod { return o.paymentMethod }
func (o Order) Status() OrderStatus          { return o.status }
func (o Order) CouponCode() string {
	if o.couponCode == nil {
		return ""
	}
	return *o.couponCode
}
func (o Order) HasCoupon() bool           { return o.couponCode != nil }
func (o Order) Dispatch() DispatchInfo    { return o.dispatch }
func (o Order) CreatedAt() time.Time      { return o.createdAt }
func (o Order) UpdatedAt() time.Time      { return o.updatedAt }
func (o Order) ConfirmedAt() *time.Time   { return o.confirmedAt }
func (o Order) PreparingAt() *time.Time   { return o.preparingAt }
func (o Order) ReadyAt() *time.Time       { return o.readyAt }
func (o Order) DispatchingAt() *time.Time { return o.dispatchingAt }
func (o Order) AssignedAt() *time.Time    { return o.assignedAt }
func (o Order) PickedUpAt() *time.Time    { return o.pickedUpAt }
func (o Order) DeliveredAt() *time.Time   { return o.deliveredAt }
func (o Order) CancelledAt() *time.Time   { return o.cancelledAt }
func (o Order) CancelReason() string {
	if o.cancelReason == nil {
		return ""
	}
	return *o.cancelReason
}
func (o Order) CancelledBy() string {
	if o.cancelledBy == nil {
		return ""
	}
	return *o.cancelledBy
}
func (o Order) IdempotencyKey() string { return o.idempotencyKey }

// Convenience queries
func (o Order) IsPending() bool        { return o.status == StatusPending }
func (o Order) IsConfirmed() bool      { return o.status == StatusConfirmed }
func (o Order) IsPreparing() bool      { return o.status == StatusPreparing }
func (o Order) IsReadyForPickup() bool { return o.status == StatusReadyForPickup }
func (o Order) IsDispatching() bool    { return o.status == StatusDispatching }
func (o Order) IsAssigned() bool       { return o.status == StatusAssigned }
func (o Order) IsPickedUp() bool       { return o.status == StatusPickedUp }
func (o Order) IsDelivered() bool      { return o.status == StatusDelivered }
func (o Order) IsCancelled() bool      { return o.status == StatusCancelled }
func (o Order) IsTerminal() bool       { return o.status.IsTerminal() }
func (o Order) IsActive() bool         { return o.status.IsActive() }
func (o Order) HasDriver() bool        { return o.dispatch.HasDriver() }
func (o Order) DriverID() string       { return o.dispatch.DriverID() }

// ItemsCount returns the total number of items (sum of quantities).
func (o Order) ItemsCount() int {
	total := 0
	for _, item := range o.items {
		total += item.Quantity()
	}
	return total
}

// ===== Lifecycle Behavior (state transitions) =====

// Confirm transitions the order from pending to confirmed.
// Called by the merchant when they accept the order.
func (o *Order) Confirm(now time.Time) error {
	newStatus, err := o.status.Transition(StatusConfirmed)
	if err != nil {
		return err
	}
	o.status = newStatus
	o.confirmedAt = &now
	o.updatedAt = now
	return nil
}

// StartPreparing transitions the order from confirmed to preparing.
// Called by the merchant when they start cooking.
func (o *Order) StartPreparing(now time.Time) error {
	newStatus, err := o.status.Transition(StatusPreparing)
	if err != nil {
		return err
	}
	o.status = newStatus
	o.preparingAt = &now
	o.updatedAt = now
	return nil
}

// MarkReadyForPickup transitions the order from preparing to ready_for_pickup.
// Called by the merchant when the food is ready.
func (o *Order) MarkReadyForPickup(now time.Time) error {
	newStatus, err := o.status.Transition(StatusReadyForPickup)
	if err != nil {
		return err
	}
	o.status = newStatus
	o.readyAt = &now
	o.updatedAt = now
	return nil
}

// StartDispatch transitions the order from ready_for_pickup to dispatching.
// Called by the system when the dispatch engine starts looking for a driver.
func (o *Order) StartDispatch(now time.Time) error {
	newStatus, err := o.status.Transition(StatusDispatching)
	if err != nil {
		return err
	}
	o.status = newStatus
	o.dispatchingAt = &now
	o.updatedAt = now
	return nil
}

// RetryDispatch transitions the order from dispatching back to ready_for_pickup.
// Called by the system when all driver offers expired/rejected and a retry is needed.
func (o *Order) RetryDispatch(now time.Time) error {
	newStatus, err := o.status.Transition(StatusReadyForPickup)
	if err != nil {
		return err
	}
	o.status = newStatus
	o.dispatchingAt = nil
	o.updatedAt = now
	return nil
}

// AssignDriver transitions the order from dispatching to assigned.
// Called by the system when a driver accepts the assignment.
func (o *Order) AssignDriver(driverID string, now time.Time) error {
	newStatus, err := o.status.Transition(StatusAssigned)
	if err != nil {
		return err
	}
	if driverID == "" {
		return NewValidationError("driver_id", ErrInvalidInput)
	}

	// Update dispatch info with driver ID.
	d := o.dispatch
	d.driverID = &driverID
	o.dispatch = d

	o.status = newStatus
	o.assignedAt = &now
	o.updatedAt = now
	return nil
}

// MarkPickedUp transitions the order from assigned to picked_up.
// Called by the driver when they pick up the order from the restaurant.
func (o *Order) MarkPickedUp(pickupPhotoURL string, now time.Time) error {
	newStatus, err := o.status.Transition(StatusPickedUp)
	if err != nil {
		return err
	}

	// Update dispatch info with pickup photo.
	if pickupPhotoURL != "" {
		d := o.dispatch
		d.pickupPhotoURL = &pickupPhotoURL
		o.dispatch = d
	}

	o.status = newStatus
	o.pickedUpAt = &now
	o.updatedAt = now
	return nil
}

// MarkDelivered transitions the order from picked_up to delivered (terminal).
// Called by the driver when they deliver the order to the customer.
func (o *Order) MarkDelivered(deliveryPhotoURL string, now time.Time) error {
	newStatus, err := o.status.Transition(StatusDelivered)
	if err != nil {
		return err
	}

	// Update dispatch info with delivery photo + dispatch distance if available.
	if deliveryPhotoURL != "" {
		d := o.dispatch
		d.deliveryPhotoURL = &deliveryPhotoURL
		o.dispatch = d
	}

	o.status = newStatus
	o.deliveredAt = &now
	o.updatedAt = now
	return nil
}

// Cancel transitions the order to cancelled.
// cancelledBy indicates who cancelled: "user", "merchant", "support", or "system".
// reason is required for all cancellations.
// Cancellation is allowed from all non-terminal states.
func (o *Order) Cancel(cancelledBy, reason string, now time.Time) error {
	if o.status.IsTerminal() {
		if o.status == StatusDelivered {
			return ErrOrderAlreadyDelivered
		}
		return ErrOrderAlreadyCancelled
	}

	if strings.TrimSpace(reason) == "" {
		return ErrCancelReasonRequired
	}

	// All non-terminal states can transition to cancelled.
	newStatus, err := o.status.Transition(StatusCancelled)
	if err != nil {
		// If the state machine doesn't allow direct transition to cancelled,
		// it means the status is terminal (but we checked above).
		// This should never happen — but handle it defensively.
		return fmt.Errorf("%w: cannot cancel from %s", ErrOrderCannotBeCancelled, o.status)
	}

	o.status = newStatus
	o.cancelledAt = &now
	o.cancelReason = &reason
	o.cancelledBy = &cancelledBy
	o.updatedAt = now
	return nil
}

// SetDispatchDistance sets the dispatch distance (driver → restaurant).
// Called by the dispatch module when a driver accepts.
func (o *Order) SetDispatchDistance(distanceM int, now time.Time) {
	d := o.dispatch
	d.dispatchDistance = &distanceM
	o.dispatch = d
	o.updatedAt = now
}

// ===== String =====

func (o Order) String() string {
	return fmt.Sprintf("Order{id=%s, number=%s, user=%s, restaurant=%s, status=%s, total=%s, items=%d}",
		o.id, o.orderNumber, o.userID, o.restaurantID, o.status, o.total.String(), o.ItemsCount())
}
