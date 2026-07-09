// Package http routes: route registration for identity endpoints.
//
// Routes are registered on a *http.ServeMux (Go 1.22+ method routing).
// The mux is returned to the caller (module.go) which combines it with
// platform-level middleware (requestid, logging, recovery, CORS).
//
// Route groups:
//   - Public:   /healthz, /auth/register, /auth/login, /auth/driver/login
//   - User:     /users/me, /auth/logout, /auth/change-password (any authenticated)
//   - Driver:   /drivers/me, /drivers/status (driver role)
//   - Admin:    /drivers/suspend (admin role)
package http

import (
        "log/slog"
        "net/http"

        "avex-backend/internal/modules/identity/port"
)

// RoutesConfig holds middleware dependencies needed for route registration.
type RoutesConfig struct {
        JWTIssuer      port.JWTIssuer
        Logger         *slog.Logger
        AllowedOrigins []string
}

// RegisterRoutes registers all identity HTTP routes on the given mux.
// The mux should already have the platform-level middleware (requestid,
// logging, recovery, CORS) applied at the server level — this function
// only registers route-specific handlers and per-route auth middleware.
func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, cfg RoutesConfig) {
        h := NewHandler(svc, cfg.Logger)

        // ----- Public routes (no auth) -----
        mux.HandleFunc("GET /healthz", h.Health)
        mux.HandleFunc("POST /api/v1/auth/register", h.Register)
        mux.HandleFunc("POST /api/v1/auth/driver/register", h.DriverRegister)
        mux.HandleFunc("POST /api/v1/auth/login", h.Login)
        mux.HandleFunc("POST /api/v1/auth/driver/login", h.DriverLogin)

        // ----- Authenticated routes (any role) -----
        // These use the Auth middleware to require a valid JWT.
        authMW := Auth(cfg.JWTIssuer, cfg.Logger)

        mux.Handle("POST /api/v1/auth/logout", authMW(http.HandlerFunc(h.Logout)))
        mux.Handle("POST /api/v1/auth/change-password", authMW(http.HandlerFunc(h.ChangePassword)))
        mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(h.GetMe)))

        // ----- Driver routes (driver role) -----
        driverAuth := Auth(cfg.JWTIssuer, cfg.Logger)
        mux.Handle("GET /api/v1/drivers/me", driverAuth(http.HandlerFunc(h.GetDriverMe)))
        mux.Handle("PATCH /api/v1/drivers/status", driverAuth(http.HandlerFunc(h.UpdateDriverStatus)))

        // ----- Admin routes (admin role) -----
        adminAuth := RequireRole(cfg.JWTIssuer, cfg.Logger, "admin")
        mux.Handle("POST /api/v1/admin/drivers/suspend", adminAuth(http.HandlerFunc(h.SuspendDriver)))
}
