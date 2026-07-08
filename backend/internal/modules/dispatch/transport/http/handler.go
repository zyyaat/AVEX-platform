// Package http: dispatch HTTP transport.
package http

import (
        "encoding/json"
        "log/slog"
        "net/http"
        "strconv"
        "time"

        "avex-backend/internal/modules/dispatch/domain"
        "avex-backend/internal/modules/dispatch/port"
        idp "avex-backend/internal/modules/identity/port"
        idhttp "avex-backend/internal/modules/identity/transport/http"
)

// Handler implements all dispatch HTTP endpoints.
type Handler struct {
        svc    port.ServicePort
        logger *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(svc port.ServicePort, logger *slog.Logger) *Handler {
        return &Handler{svc: svc, logger: logger}
}

// ===== Helpers =====

func writeJSON(w http.ResponseWriter, status int, v any) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeErr(w http.ResponseWriter, logger *slog.Logger, err error) {
        status := http.StatusInternalServerError
        switch {
        case err == domain.ErrDriverNotFound ||
                err == domain.ErrLocationNotFound ||
                err == domain.ErrOfferNotFound:
                status = http.StatusNotFound
        case err == domain.ErrDriverAlreadyExists ||
                err == domain.ErrOfferAlreadyExists:
                status = http.StatusConflict
        case err == domain.ErrDriverOffline ||
                err == domain.ErrDriverBusy ||
                err == domain.ErrDriverSuspended ||
                err == domain.ErrDriverOnDuty ||
                err == domain.ErrOfferExpired ||
                err == domain.ErrOfferAlreadyAccepted ||
                err == domain.ErrOfferAlreadyRejected ||
                err == domain.ErrOfferAlreadyCancelled ||
                err == domain.ErrOfferNotPending ||
                err == domain.ErrDriverNotEligible ||
                err == domain.ErrNoDriversAvailable ||
                err == domain.ErrMaxAttemptsReached:
                status = http.StatusUnprocessableEntity
        case err == domain.ErrInvalidID ||
                err == domain.ErrInvalidInput ||
                err == domain.ErrInvalidDriverStatus ||
                err == domain.ErrInvalidVehicleType ||
                err == domain.ErrInvalidLatitude ||
                err == domain.ErrInvalidLongitude ||
                err == domain.ErrInvalidBearing ||
                err == domain.ErrInvalidRadius ||
                err == domain.ErrInvalidLimit:
                status = http.StatusBadRequest
        }
        if status >= 500 && logger != nil {
                logger.Error("internal error", "error", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func parsePage(r *http.Request) port.PageQuery {
        limit, offset := 50, 0
        if l := r.URL.Query().Get("limit"); l != "" {
                if n, e := strconv.Atoi(l); e == nil {
                        limit = n
                }
        }
        if o := r.URL.Query().Get("offset"); o != "" {
                if n, e := strconv.Atoi(o); e == nil {
                        offset = n
                }
        }
        return port.PageQuery{Limit: limit, Offset: offset}
}

// ===== Driver Endpoints =====

// POST /api/v1/admin/drivers — register a new driver
func (h *Handler) RegisterDriver(w http.ResponseWriter, r *http.Request) {
        var req struct {
                UserID       string `json:"user_id"`
                VehicleType  string `json:"vehicle_type"`
                LicensePlate string `json:"license_plate"`
                ZoneIDs      []string `json:"zone_ids"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.RegisterDriver(r.Context(), port.RegisterDriverInput{
                UserID:       req.UserID,
                VehicleType:  req.VehicleType,
                LicensePlate: req.LicensePlate,
                ZoneIDs:      req.ZoneIDs,
        })
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/drivers/{id}
func (h *Handler) GetDriver(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.GetDriver(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/drivers?user_id=...
func (h *Handler) GetDriverByUserID(w http.ResponseWriter, r *http.Request) {
        userID := r.URL.Query().Get("user_id")
        if userID == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.GetDriverByUserID(r.Context(), userID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/drivers/{id}/online
func (h *Handler) GoOnline(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.GoOnline(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/drivers/{id}/offline
func (h *Handler) GoOffline(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.GoOffline(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/drivers/{id}/suspend
func (h *Handler) SuspendDriver(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        var req struct{ Reason string }
        _ = json.NewDecoder(r.Body).Decode(&req)
        result, err := h.svc.SuspendDriver(r.Context(), id, req.Reason)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/drivers/{id}/unsuspend
func (h *Handler) UnsuspendDriver(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.UnsuspendDriver(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/admin/drivers
func (h *Handler) ListDrivers(w http.ResponseWriter, r *http.Request) {
        page := parsePage(r)
        result, err := h.svc.ListDrivers(r.Context(), page)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/drivers/online?zone_id=...
func (h *Handler) ListOnlineDrivers(w http.ResponseWriter, r *http.Request) {
        zoneID := r.URL.Query().Get("zone_id")
        drivers, err := h.svc.ListOnlineDrivers(r.Context(), zoneID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, drivers)
}

// ===== Location Endpoints =====

// POST /api/v1/drivers/{id}/location
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
        driverID := r.PathValue("id")
        var req struct {
                Lat        float64
                Lng        float64
                Bearing    float64
                Speed      float64
                Accuracy   float64
                CapturedAt string // RFC3339
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        var capturedAt time.Time
        if req.CapturedAt != "" {
                t, err := time.Parse(time.RFC3339, req.CapturedAt)
                if err != nil {
                        writeErr(w, h.logger, domain.ErrInvalidInput)
                        return
                }
                capturedAt = t
        }
        result, err := h.svc.UpdateLocation(r.Context(), port.UpdateLocationInput{
                DriverID:   driverID,
                Lat:        req.Lat,
                Lng:        req.Lng,
                Bearing:    req.Bearing,
                Speed:      req.Speed,
                Accuracy:   req.Accuracy,
                CapturedAt: capturedAt,
        })
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/drivers/{id}/location
func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
        driverID := r.PathValue("id")
        result, err := h.svc.GetLocation(r.Context(), driverID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/drivers/nearby?lat=...&lng=...&radius=...&limit=...
func (h *Handler) FindNearestDrivers(w http.ResponseWriter, r *http.Request) {
        latStr := r.URL.Query().Get("lat")
        lngStr := r.URL.Query().Get("lng")
        if latStr == "" || lngStr == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        lat, err := strconv.ParseFloat(latStr, 64)
        if err != nil {
                writeErr(w, h.logger, domain.ErrInvalidLatitude)
                return
        }
        lng, err := strconv.ParseFloat(lngStr, 64)
        if err != nil {
                writeErr(w, h.logger, domain.ErrInvalidLongitude)
                return
        }
        radius := 0
        limit := 10
        if r := r.URL.Query().Get("radius"); r != "" {
                if n, e := strconv.Atoi(r); e == nil {
                        radius = n
                }
        }
        if l := r.URL.Query().Get("limit"); l != "" {
                if n, e := strconv.Atoi(l); e == nil {
                        limit = n
                }
        }
        drivers, err := h.svc.FindNearestDrivers(r.Context(), lat, lng, radius, limit)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, drivers)
}

// ===== Offer Endpoints =====

// POST /api/v1/admin/dispatch/offers — manually create an offer
func (h *Handler) CreateOffer(w http.ResponseWriter, r *http.Request) {
        var req struct {
                OrderID     string
                ZoneID      string
                PickupLat   float64
                PickupLng   float64
                DeliveryLat float64
                DeliveryLng float64
                Currency    string
                DriverID    string // optional — for manual dispatch
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.CreateOffer(r.Context(), port.CreateOfferInput{
                OrderID:     req.OrderID,
                ZoneID:      req.ZoneID,
                PickupLat:   req.PickupLat,
                PickupLng:   req.PickupLng,
                DeliveryLat: req.DeliveryLat,
                DeliveryLng: req.DeliveryLng,
                Currency:    req.Currency,
                DriverID:    req.DriverID,
        })
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/dispatch/offers/{id}
func (h *Handler) GetOffer(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.GetOffer(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/dispatch/offers/{id}/accept
// Body: { "driver_id": "..." }
func (h *Handler) AcceptOffer(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        var req struct{ DriverID string }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.AcceptOffer(r.Context(), id, req.DriverID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/dispatch/offers/{id}/reject
// Body: { "driver_id": "...", "reason": "..." }
func (h *Handler) RejectOffer(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        var req struct {
                DriverID string
                Reason   string
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.RejectOffer(r.Context(), id, req.DriverID, req.Reason)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/dispatch/offers/{id}/expire
func (h *Handler) ExpireOffer(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.ExpireOffer(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/dispatch/offers/{id}/cancel
func (h *Handler) CancelOffer(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.CancelOffer(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/dispatch/offers?driver_id=...
func (h *Handler) ListOffersByDriver(w http.ResponseWriter, r *http.Request) {
        driverID := r.URL.Query().Get("driver_id")
        if driverID == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        page := parsePage(r)
        result, err := h.svc.ListOffersByDriver(r.Context(), driverID, page)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/dispatch/offers?order_id=...
func (h *Handler) ListOffersByOrder(w http.ResponseWriter, r *http.Request) {
        orderID := r.URL.Query().Get("order_id")
        if orderID == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        offers, err := h.svc.ListOffersByOrder(r.Context(), orderID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, offers)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
        h := NewHandler(svc, logger)
        authMW := idhttp.Auth(jwtIssuer, logger)

        // Authenticated (Bearer)
        mux.Handle("GET /api/v1/drivers/{id}", authMW(http.HandlerFunc(h.GetDriver)))
        mux.Handle("GET /api/v1/drivers", authMW(http.HandlerFunc(h.GetDriverByUserID)))
        mux.Handle("POST /api/v1/drivers/{id}/online", authMW(http.HandlerFunc(h.GoOnline)))
        mux.Handle("POST /api/v1/drivers/{id}/offline", authMW(http.HandlerFunc(h.GoOffline)))
        mux.Handle("GET /api/v1/drivers/online", authMW(http.HandlerFunc(h.ListOnlineDrivers)))
        mux.Handle("POST /api/v1/drivers/{id}/location", authMW(http.HandlerFunc(h.UpdateLocation)))
        mux.Handle("GET /api/v1/drivers/{id}/location", authMW(http.HandlerFunc(h.GetLocation)))
        mux.Handle("GET /api/v1/drivers/nearby", authMW(http.HandlerFunc(h.FindNearestDrivers)))
        mux.Handle("GET /api/v1/dispatch/offers/{id}", authMW(http.HandlerFunc(h.GetOffer)))
        mux.Handle("POST /api/v1/dispatch/offers/{id}/accept", authMW(http.HandlerFunc(h.AcceptOffer)))
        mux.Handle("POST /api/v1/dispatch/offers/{id}/reject", authMW(http.HandlerFunc(h.RejectOffer)))
        mux.Handle("GET /api/v1/dispatch/offers", authMW(http.HandlerFunc(h.listOffers))) // dispatches by query param

        // Admin (Bearer)
        mux.Handle("POST /api/v1/admin/drivers", authMW(http.HandlerFunc(h.RegisterDriver)))
        mux.Handle("GET /api/v1/admin/drivers", authMW(http.HandlerFunc(h.ListDrivers)))
        mux.Handle("POST /api/v1/admin/drivers/{id}/suspend", authMW(http.HandlerFunc(h.SuspendDriver)))
        mux.Handle("POST /api/v1/admin/drivers/{id}/unsuspend", authMW(http.HandlerFunc(h.UnsuspendDriver)))
        mux.Handle("POST /api/v1/admin/dispatch/offers", authMW(http.HandlerFunc(h.CreateOffer)))
        mux.Handle("POST /api/v1/admin/dispatch/offers/{id}/expire", authMW(http.HandlerFunc(h.ExpireOffer)))
        mux.Handle("POST /api/v1/admin/dispatch/offers/{id}/cancel", authMW(http.HandlerFunc(h.CancelOffer)))
}

// listOffers dispatches to ListOffersByDriver or ListOffersByOrder based on query params.
func (h *Handler) listOffers(w http.ResponseWriter, r *http.Request) {
        if orderID := r.URL.Query().Get("order_id"); orderID != "" {
                h.ListOffersByOrder(w, r)
                return
        }
        h.ListOffersByDriver(w, r)
}
