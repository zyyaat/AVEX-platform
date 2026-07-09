// Package http is the identity module's HTTP transport layer.
//
// It exposes the ServicePort via HTTP endpoints. The transport layer:
//   - Parses and validates HTTP requests (request structs in dto.go)
//   - Calls ServicePort methods (no business logic here)
//   - Maps domain errors to HTTP responses (errors.go)
//   - Extracts actor context from JWT (middleware.go)
//
// Design rules:
//   - No domain imports (only port types + DTOs)
//   - No repository imports
//   - No SQL
//   - No business rules
//   - Handlers are thin: parse → validate → call service → write response
package http

// ===== Request DTOs =====
//
// Request structs are transport-specific — they map 1:1 to HTTP request
// bodies. Validation happens in validation.go before the service is called.
// The service receives port.*Input structs (constructed in handlers.go).

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
        Name     string `json:"name"`
        Phone    string `json:"phone"`
        Password string `json:"password"`
        Email    string `json:"email,omitempty"`
        Locale   string `json:"locale,omitempty"`
}

// LoginRequest is the body for POST /auth/login and POST /auth/driver/login.
type LoginRequest struct {
        Phone    string `json:"phone"`
        Password string `json:"password"`
}

// DriverRegisterRequest is the body for POST /auth/driver/register.
type DriverRegisterRequest struct {
        Name          string `json:"name"`
        Phone         string `json:"phone"`
        Password      string `json:"password"`
        VehicleType   string `json:"vehicle_type"`
        LicenseNumber string `json:"license_number"`
        NationalID    string `json:"national_id"`
        AutoVerify    bool   `json:"auto_verify"`
}

// ChangePasswordRequest is the body for POST /auth/change-password.
type ChangePasswordRequest struct {
        OldPassword string `json:"old_password"`
        NewPassword string `json:"new_password"`
}

// UpdateDriverStatusRequest is the body for PATCH /drivers/status.
type UpdateDriverStatusRequest struct {
        Status string  `json:"status"` // "online" | "offline"
        Lat    float64 `json:"lat"`
        Lng    float64 `json:"lng"`
}

// SuspendDriverRequest is the body for POST /drivers/suspend.
type SuspendDriverRequest struct {
        DriverID string `json:"driver_id"`
        Reason   string `json:"reason"`
}

// ===== Response Envelope =====
//
// All responses use a standard envelope for consistency.
// Success: {"data": <payload>}
// Error:   {"error": {"message": "...", "code": "..."}} (see errors.go)

// SuccessResponse wraps a successful response payload.
type SuccessResponse struct {
        Data any `json:"data"`
}
