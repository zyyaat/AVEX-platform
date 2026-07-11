// Package http routes: route registration for orders endpoints.
package http

import (
        "log/slog"
        "net/http"

        idp "avex-backend/internal/modules/identity/port"
        idhttp "avex-backend/internal/modules/identity/transport/http"
        "avex-backend/internal/modules/orders/port"
)

// RegisterRoutes registers all orders HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
        h := NewHandler(svc, logger)

        authMW := idhttp.Auth(jwtIssuer, logger)

        // Public
        mux.HandleFunc("GET /api/v1/orders/track/{orderNumber}", h.TrackOrder)

        // Customer (auth)
        mux.Handle("POST /api/v1/orders", authMW(http.HandlerFunc(h.CreateOrder)))
        mux.Handle("GET /api/v1/orders/my", authMW(http.HandlerFunc(h.ListMyOrders)))
        mux.Handle("GET /api/v1/orders/{id}", authMW(http.HandlerFunc(h.GetOrder)))
        mux.Handle("POST /api/v1/orders/{id}/cancel", authMW(http.HandlerFunc(h.CancelOrder)))

        // Merchant (auth)
        mux.Handle("POST /api/v1/orders/{id}/confirm", authMW(http.HandlerFunc(h.ConfirmOrder)))
        mux.Handle("POST /api/v1/orders/{id}/prepare", authMW(http.HandlerFunc(h.StartPreparing)))
        mux.Handle("POST /api/v1/orders/{id}/ready", authMW(http.HandlerFunc(h.MarkReadyForPickup)))
        mux.Handle("GET /api/v1/orders/restaurant/{restaurantID}", authMW(http.HandlerFunc(h.ListRestaurantOrders)))

        // System (no auth — called by dispatch module)
        mux.HandleFunc("POST /api/v1/orders/{id}/dispatch", h.StartDispatch)
        mux.HandleFunc("POST /api/v1/orders/{id}/assign", h.AssignDriver)

        // Driver (auth)
        mux.Handle("POST /api/v1/orders/{id}/pickup", authMW(http.HandlerFunc(h.MarkPickedUp)))
        mux.Handle("POST /api/v1/orders/{id}/deliver", authMW(http.HandlerFunc(h.MarkDelivered)))
        mux.Handle("GET /api/v1/orders/driver/{driverID}", authMW(http.HandlerFunc(h.ListDriverOrders)))
        // NEW: driver can list their own orders without knowing dispatch ID
        mux.Handle("GET /api/v1/orders/driver/me", authMW(http.HandlerFunc(h.ListMyDriverOrders)))

        // Admin (auth)
        mux.Handle("GET /api/v1/orders", authMW(http.HandlerFunc(h.ListOrdersByStatus)))
}
