// Package http handlers: HTTP handlers for orders endpoints.
package http

import (
        "context"
        "encoding/json"
        "fmt"
        "log/slog"
        "net/http"
        "strconv"

        idhttp "avex-backend/internal/modules/identity/transport/http"
        "avex-backend/internal/modules/orders/port"
)

type Handler struct {
        svc    port.ServicePort
        logger *slog.Logger
}

func NewHandler(svc port.ServicePort, logger *slog.Logger) *Handler {
        return &Handler{svc: svc, logger: logger}
}

// ===== Create Order =====
// POST /api/v1/orders

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
        var req CreateOrderRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateCreateOrder(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        items := make([]port.CreateOrderItemInput, len(req.Items))
        for i, item := range req.Items {
                items[i] = port.CreateOrderItemInput{
                        MenuItemID: item.MenuItemID, Name: item.Name, NameAr: item.NameAr,
                        PriceCents: item.PriceCents, Quantity: item.Quantity,
                }
        }

        input := port.CreateOrderInput{
                UserID: req.UserID, RestaurantID: req.RestaurantID,
                CustomerName: req.CustomerName, CustomerPhone: req.CustomerPhone,
                DeliveryLat: req.DeliveryInfo.Lat, DeliveryLng: req.DeliveryInfo.Lng,
                DeliveryAddress: req.DeliveryInfo.Address, DeliveryNotes: req.DeliveryInfo.Notes,
                Items:         items,
                SubtotalCents: req.Subtotal, DeliveryFeeCents: req.DeliveryFee,
                DiscountCents: req.Discount, TaxCents: req.Tax, TotalCents: req.Total,
                Currency: req.Currency, PaymentMethod: req.PaymentMethod,
                CouponCode: req.CouponCode, ZoneID: req.ZoneID,
                IdempotencyKey: r.Header.Get("Idempotency-Key"),
        }

        result, err := h.svc.CreateOrder(r.Context(), input)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, result)
}

// ===== Confirm Order =====
// POST /api/v1/orders/{id}/confirm

func (h *Handler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        changedBy := r.Header.Get("X-Actor-ID")
        if changedBy == "" {
                changedBy = "system"
        }

        result, err := h.svc.ConfirmOrder(r.Context(), orderID, changedBy)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Start Preparing =====
// POST /api/v1/orders/{id}/prepare

func (h *Handler) StartPreparing(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        changedBy := r.Header.Get("X-Actor-ID")
        if changedBy == "" {
                changedBy = "system"
        }

        result, err := h.svc.StartPreparing(r.Context(), orderID, changedBy)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Mark Ready For Pickup =====
// POST /api/v1/orders/{id}/ready

func (h *Handler) MarkReadyForPickup(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        changedBy := r.Header.Get("X-Actor-ID")
        if changedBy == "" {
                changedBy = "system"
        }

        result, err := h.svc.MarkReadyForPickup(r.Context(), orderID, changedBy)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Start Dispatch =====
// POST /api/v1/orders/{id}/dispatch

func (h *Handler) StartDispatch(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")

        result, err := h.svc.StartDispatch(r.Context(), orderID)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Assign Driver =====
// POST /api/v1/orders/{id}/assign

func (h *Handler) AssignDriver(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        var req AssignDriverRequest
        _ = json.NewDecoder(r.Body).Decode(&req)

        input := port.AssignDriverInput{
                OrderID: orderID, DriverID: req.DriverID,
                AssignmentID: req.AssignmentID, DispatchDistM: req.DispatchDistM,
        }

        result, err := h.svc.AssignDriver(r.Context(), input)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Mark Picked Up =====
// POST /api/v1/orders/{id}/pickup
// Driver ID is taken from the JWT (via ActorFromContext), NOT from headers.

func (h *Handler) MarkPickedUp(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        // FIXED: use JWT actor instead of X-Driver-ID header (security fix)
        actor := idhttp.ActorFromContext(r.Context())
        if actor == nil {
                writeError(w, h.logger, fmt.Errorf("authentication required"))
                return
        }
        driverID := actor.Subject
        var req MarkPickedUpRequest
        _ = json.NewDecoder(r.Body).Decode(&req)

        input := port.MarkPickedUpInput{OrderID: orderID, DriverID: driverID, PickupPhotoURL: req.PickupPhotoURL}

        result, err := h.svc.MarkPickedUp(r.Context(), input)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Mark Delivered =====
// POST /api/v1/orders/{id}/deliver
// Driver ID is taken from the JWT (via ActorFromContext), NOT from headers.

func (h *Handler) MarkDelivered(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        // FIXED: use JWT actor instead of X-Driver-ID header (security fix)
        actor := idhttp.ActorFromContext(r.Context())
        if actor == nil {
                writeError(w, h.logger, fmt.Errorf("authentication required"))
                return
        }
        driverID := actor.Subject
        var req MarkDeliveredRequest
        _ = json.NewDecoder(r.Body).Decode(&req)

        input := port.MarkDeliveredInput{
                OrderID: orderID, DriverID: driverID,
                DeliveryPhotoURL: req.DeliveryPhotoURL, DeliveryDistanceM: req.DeliveryDistanceM,
        }

        result, err := h.svc.MarkDelivered(r.Context(), input)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Cancel Order =====
// POST /api/v1/orders/{id}/cancel

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")
        var req CancelOrderRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateCancelOrder(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        cancelledBy := r.Header.Get("X-Actor-Type")
        if cancelledBy == "" {
                cancelledBy = "user"
        }

        input := port.CancelOrderInput{OrderID: orderID, CancelledBy: cancelledBy, Reason: req.Reason}

        result, err := h.svc.CancelOrder(r.Context(), input)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Get Order =====
// GET /api/v1/orders/{id}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
        orderID := r.PathValue("id")

        result, err := h.svc.GetOrder(r.Context(), orderID)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Track Order =====
// GET /api/v1/orders/track/{orderNumber}

func (h *Handler) TrackOrder(w http.ResponseWriter, r *http.Request) {
        orderNumber := r.PathValue("orderNumber")

        result, err := h.svc.TrackOrder(r.Context(), orderNumber)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== List My Orders =====
// GET /api/v1/orders/my?limit=50&offset=0

func (h *Handler) ListMyOrders(w http.ResponseWriter, r *http.Request) {
        // FIXED: use JWT actor instead of X-User-ID header (security fix)
        userID := ""
        if actor := idhttp.ActorFromContext(r.Context()); actor != nil {
                userID = actor.Subject
        }
        if userID == "" {
                // Fall back to header for backwards compatibility
                userID = r.Header.Get("X-User-ID")
        }
        if userID == "" {
                writeError(w, h.logger, fmt.Errorf("authentication required"))
                return
        }
        page := parsePageQuery(r)

        result, err := h.svc.ListMyOrders(r.Context(), userID, page)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== List Restaurant Orders =====
// GET /api/v1/orders/restaurant/{restaurantID}?limit=50&offset=0

func (h *Handler) ListRestaurantOrders(w http.ResponseWriter, r *http.Request) {
        restaurantID := r.PathValue("restaurantID")
        page := parsePageQuery(r)

        result, err := h.svc.ListRestaurantOrders(r.Context(), restaurantID, page)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== List Driver Orders =====
// GET /api/v1/orders/driver/{driverID}?limit=50&offset=0

func (h *Handler) ListDriverOrders(w http.ResponseWriter, r *http.Request) {
        // FIXED: use JWT actor if available, fall back to path param for backwards compat
        driverID := r.PathValue("driverID")
        if actor := idhttp.ActorFromContext(r.Context()); actor != nil && driverID == "" {
                driverID = actor.Subject
        }
        page := parsePageQuery(r)

        result, err := h.svc.ListDriverOrders(r.Context(), driverID, page)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== List Orders By Status (admin) =====
// GET /api/v1/orders?status=pending&limit=50&offset=0

func (h *Handler) ListOrdersByStatus(w http.ResponseWriter, r *http.Request) {
        status := r.URL.Query().Get("status")
        if status == "" {
                status = "pending"
        }
        page := parsePageQuery(r)

        result, err := h.svc.ListOrdersByStatus(r.Context(), status, page)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Helper: parse pagination from query params =====

func parsePageQuery(r *http.Request) port.PageQuery {
        limit := 50
        offset := 0
        if l := r.URL.Query().Get("limit"); l != "" {
                if n, err := strconv.Atoi(l); err == nil && n > 0 {
                        limit = n
                }
        }
        if o := r.URL.Query().Get("offset"); o != "" {
                if n, err := strconv.Atoi(o); err == nil && n >= 0 {
                        offset = n
                }
        }
        return port.PageQuery{Limit: limit, Offset: offset}
}

// suppress unused import
var _ = context.Background

// ===== List My Driver Orders =====
// GET /api/v1/orders/driver/me?limit=50&offset=0
// Uses JWT actor.Subject as the driver ID — no path param needed.
// This is the preferred endpoint for the driver app.

func (h *Handler) ListMyDriverOrders(w http.ResponseWriter, r *http.Request) {
	actor := idhttp.ActorFromContext(r.Context())
	if actor == nil {
		writeError(w, h.logger, fmt.Errorf("authentication required"))
		return
	}
	driverID := actor.Subject
	page := parsePageQuery(r)

	result, err := h.svc.ListDriverOrders(r.Context(), driverID, page)
	if err != nil {
		writeError(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
