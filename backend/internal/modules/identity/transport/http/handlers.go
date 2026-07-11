// Package http handlers: HTTP handlers for identity endpoints.
//
// Each handler:
//  1. Parses the request body (readJSON)
//  2. Validates the request (validate*)
//  3. Constructs a port.*Input struct
//  4. Calls the appropriate ServicePort method
//  5. Writes a JSON response (writeJSON) or error (writeError)
//
// Handlers are thin — no business logic. The service layer does all
// the work. Handlers only translate between HTTP and the service interface.
package http

import (
        "context"
        "log/slog"
        "net/http"

        "github.com/google/uuid"
        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/identity/port"
)

// Handler holds dependencies for HTTP handlers.
// The service is the only business dependency — handlers do NOT access
// repositories or domain directly.
type Handler struct {
        svc    port.ServicePort
        logger *slog.Logger
        pool   *pgxpool.Pool // for direct DB access (admin create driver → dispatch)
}

// NewHandler creates a new Handler.
func NewHandler(svc port.ServicePort, logger *slog.Logger) *Handler {
        return &Handler{svc: svc, logger: logger}
}

// ===== Auth Endpoints =====

// Register handles POST /auth/register.
// Public endpoint (no auth required).
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
        var req RegisterRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateRegister(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        result, err := h.svc.RegisterUser(r.Context(), port.RegisterUserInput{
                Name:     req.Name,
                Phone:    req.Phone,
                Password: req.Password,
                Email:    req.Email,
                Locale:   req.Locale,
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, result)
}

// DriverRegister handles POST /auth/driver/register.
// Public endpoint — creates a new driver account in identity.drivers.
func (h *Handler) DriverRegister(w http.ResponseWriter, r *http.Request) {
        var req DriverRegisterRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if req.Name == "" || req.Phone == "" || req.Password == "" {
                writeError(w, h.logger, newValidationError(map[string]string{
                        "name": "name is required", "phone": "phone is required", "password": "password is required",
                }))
                return
        }

        result, err := h.svc.RegisterDriver(r.Context(), port.RegisterDriverInput{
                Name:          req.Name,
                Phone:         req.Phone,
                Password:      req.Password,
                VehicleType:   req.VehicleType,
                LicenseNumber: req.LicenseNumber,
                NationalID:    req.NationalID,
                AutoVerify:    req.AutoVerify,
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, result)
}

// Login handles POST /auth/login.
// Public endpoint.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
        var req LoginRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateLogin(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        result, err := h.svc.LoginUser(r.Context(), port.LoginInput{
                Phone:    req.Phone,
                Password: req.Password,
                IP:       clientIP(r),
                Agent:    r.UserAgent(),
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// DriverLogin handles POST /auth/driver/login.
// Public endpoint.
func (h *Handler) DriverLogin(w http.ResponseWriter, r *http.Request) {
        var req LoginRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateLogin(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        result, err := h.svc.LoginDriver(r.Context(), port.LoginInput{
                Phone:    req.Phone,
                Password: req.Password,
                IP:       clientIP(r),
                Agent:    r.UserAgent(),
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// Logout handles POST /auth/logout.
// Requires authentication. The session ID comes from the JWT (via Auth middleware).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        err := h.svc.Logout(r.Context(), actor.SessionID)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// ChangePassword handles POST /auth/change-password.
// Requires authentication. The subject ID comes from the JWT.
// Both users and drivers use this endpoint; the role in the JWT determines
// which service method is called.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        var req ChangePasswordRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateChangePassword(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        input := port.ChangePasswordInput{
                SubjectID:   actor.Subject,
                OldPassword: req.OldPassword,
                NewPassword: req.NewPassword,
        }

        var err error
        if actor.Role == "driver" {
                err = h.svc.ChangeDriverPassword(r.Context(), input)
        } else {
                err = h.svc.ChangePassword(r.Context(), input)
        }
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

// ===== User Endpoints =====

// GetMe handles GET /users/me.
// Requires authentication (user role).
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        user, err := h.svc.GetUser(r.Context(), actor.Subject)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, user)
}

// ===== Driver Endpoints =====

// GetDriverMe handles GET /drivers/me.
// Requires authentication (driver role).
func (h *Handler) GetDriverMe(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        driver, err := h.svc.GetDriverProfile(r.Context(), actor.Subject)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, driver)
}

// GetMerchantMe handles GET /merchants/me.
// Requires authentication (merchant role).
// Returns the merchant's profile (linked restaurant, settings, etc.).
func (h *Handler) GetMerchantMe(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        merchant, err := h.svc.GetMerchantProfile(r.Context(), actor.Subject)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, merchant)
}

// GetAgentMe handles GET /agents/me.
// Requires authentication (agent role).
// Returns the support agent's profile.
// NOTE: Agents are stored as users with role='agent' in identity.users.
// This endpoint returns the user profile (same as /users/me but for agents).
func (h *Handler) GetAgentMe(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        // Agents are users with role='agent'. Return the user profile.
        user, err := h.svc.GetUser(r.Context(), actor.Subject)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, user)
}

// UpdateDriverStatus handles PATCH /drivers/status.
// Requires authentication (driver role).
func (h *Handler) UpdateDriverStatus(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        var req UpdateDriverStatusRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateUpdateDriverStatus(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        result, err := h.svc.UpdateDriverStatus(r.Context(), port.UpdateDriverStatusInput{
                DriverID: actor.Subject,
                Status:   req.Status,
                Lat:      req.Lat,
                Lng:      req.Lng,
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// SuspendDriver handles POST /drivers/suspend.
// Requires authentication (admin role only).
func (h *Handler) SuspendDriver(w http.ResponseWriter, r *http.Request) {
        actor := actorFromContext(r.Context())
        if actor == nil {
                writeAuthError(w, "authentication required")
                return
        }

        var req SuspendDriverRequest
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }
        if verr := validateSuspendDriver(&req); verr != nil {
                writeError(w, h.logger, verr)
                return
        }

        err := h.svc.SuspendDriver(r.Context(), port.SuspendDriverInput{
                DriverID:    req.DriverID,
                Reason:      req.Reason,
                SuspendedBy: actor.Subject,
        })
        if err != nil {
                writeError(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "driver suspended"})
}

// ===== Health Check =====

// Health handles GET /healthz.
// Public endpoint. Returns 200 if the service is up.
// A deeper readiness check (DB ping) can be added later.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
        writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ===== Helpers =====

// clientIP extracts the client IP from the request.
// Checks X-Forwarded-For first (for proxies), falls back to RemoteAddr.
func clientIP(r *http.Request) string {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
                // X-Forwarded-For may contain multiple IPs; take the first.
                if idx := indexOf(xff, ','); idx > 0 {
                        return trim(xff[:idx])
                }
                return trim(xff)
        }
        // RemoteAddr is "host:port" — strip the port.
        addr := r.RemoteAddr
        if idx := lastIndexOf(addr, ':'); idx > 0 {
                return addr[:idx]
        }
        return addr
}

// indexOf returns the index of the first occurrence of b in s, or -1.
func indexOf(s string, b byte) int {
        for i := 0; i < len(s); i++ {
                if s[i] == b {
                        return i
                }
        }
        return -1
}

// lastIndexOf returns the index of the last occurrence of b in s, or -1.
func lastIndexOf(s string, b byte) int {
        for i := len(s) - 1; i >= 0; i-- {
                if s[i] == b {
                        return i
                }
        }
        return -1
}

// trim removes leading and trailing whitespace from s.
func trim(s string) string {
        start := 0
        end := len(s)
        for start < end && (s[start] == ' ' || s[start] == '\t') {
                start++
        }
        for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
                end--
        }
        return s[start:end]
}

// Ensure context import is used (actorFromContext uses it).
var _ = context.Background

// ===== Setup Endpoints (for initial system setup only) =====
// These are NOT auth-protected. They should be disabled in production.
// They exist because there's no way to create the first admin or verify
// the first driver without them.

// PromoteToAdmin handles POST /api/v1/setup/promote-admin
// Body: {"phone": "01000000000"}
// Promotes a user to admin (is_admin = true).
func (h *Handler) PromoteToAdmin(w http.ResponseWriter, r *http.Request) {
        var req struct {
                Phone string `json:"phone"`
        }
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }

        user, err := h.svc.GetUserByPhone(r.Context(), req.Phone)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }

        // Promote to admin
        if err := h.svc.PromoteUserToAdmin(r.Context(), user.ID); err != nil {
                writeError(w, h.logger, err)
                return
        }

        writeJSON(w, http.StatusOK, map[string]string{"status": "promoted to admin"})
}

// VerifyDriver handles POST /api/v1/setup/verify-driver
// Body: {"phone": "01012345678"}
// Verifies a driver (is_verified = true, is_active = true).
func (h *Handler) VerifyDriver(w http.ResponseWriter, r *http.Request) {
        var req struct {
                Phone string `json:"phone"`
        }
        if err := readJSON(r, &req); err != nil {
                writeError(w, h.logger, err)
                return
        }

        // Get driver by phone
        driver, err := h.svc.GetDriverByPhone(r.Context(), req.Phone)
        if err != nil {
                writeError(w, h.logger, err)
                return
        }

        // Verify driver
        if err := h.svc.VerifyDriverAccount(r.Context(), driver.ID); err != nil {
                writeError(w, h.logger, err)
                return
        }

        writeJSON(w, http.StatusOK, map[string]string{"status": "driver verified"})
}

// AdminCreateDriverHandler handles POST /api/v1/admin/drivers/create
// Creates a complete driver in ONE call:
//   1. identity.drivers (verified + active)
//   2. dispatch.drivers (for delivery operations)
func (h *Handler) AdminCreateDriverHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string   `json:"name"`
		Phone         string   `json:"phone"`
		Password      string   `json:"password"`
		VehicleType   string   `json:"vehicle_type"`
		LicenseNumber string   `json:"license_number"`
		NationalID    string   `json:"national_id"`
		LicensePlate  string   `json:"license_plate"`
		ZoneIDs       []string `json:"zone_ids"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, h.logger, err)
		return
	}

	// Step 1: Create driver in identity.drivers (verified + active)
	driverID, err := h.svc.AdminCreateDriver(r.Context(), port.AdminCreateDriverInput{
		Name:          req.Name,
		Phone:         req.Phone,
		Password:      req.Password,
		VehicleType:   req.VehicleType,
		LicenseNumber: req.LicenseNumber,
		NationalID:    req.NationalID,
		LicensePlate:  req.LicensePlate,
		ZoneIDs:       req.ZoneIDs,
	})
	if err != nil {
		writeError(w, h.logger, err)
		return
	}

	// Step 2: Register in dispatch.drivers via DIRECT SQL INSERT
	// Map identity vehicle types to dispatch vehicle types (dispatch only allows bike|scooter|car).
	dispatchVehicleType := "bike"
	switch req.VehicleType {
	case "scooter":
		dispatchVehicleType = "scooter"
	case "car":
		dispatchVehicleType = "car"
	}

	dispatchID := uuid.New().String()
	licensePlate := req.LicensePlate
	if licensePlate == "" {
		licensePlate = "N/A"
	}

	_, dispatchErr := h.pool.Exec(r.Context(), `
		INSERT INTO dispatch.drivers (
			id, user_id, vehicle_type, license_plate,
			status, rating, rating_count, acceptance_rate, completion_rate, total_deliveries,
			zone_ids, current_order_id, go_online_at, go_offline_at, suspended_reason,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4,
			'offline', 5.0, 0, 100, 100, 0,
			$5, NULL, NULL, NULL, '',
			NOW(), NOW(), 1
		)
		ON CONFLICT (user_id) DO NOTHING
	`,
		dispatchID, driverID, dispatchVehicleType, licensePlate,
		req.ZoneIDs,
	)

	if dispatchErr != nil {
		h.logger.Error("dispatch.drivers insert failed", "error", dispatchErr, "driver_id", driverID)
		writeJSON(w, http.StatusCreated, map[string]any{
			"driver_id":   driverID,
			"status":      "partial",
			"message":     "Driver created in identity, dispatch registration failed",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"driver_id":   driverID,
		"dispatch_id": dispatchID,
		"status":      "created",
		"message":     "Driver fully created",
	})
}
