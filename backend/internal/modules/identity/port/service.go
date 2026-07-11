// Package port service: ServicePort interface (what identity exposes to the
// world) and the DTOs used as input/output.
//
// ServicePort is the SINGLE entry point to the identity module. Both the
// HTTP transport layer (within identity) and other modules (via their own
// services) call methods on ServicePort.
//
// Design rules:
//   - Methods return DTOs, NOT domain entities. This prevents other modules
//     from importing identity/domain (which is forbidden by the architecture
//     rules). DTOs are immutable snapshots of the data callers need.
//   - Input DTOs use primitive types (string, int) — callers don't need to
//     construct domain value objects (e.g. Phone) before calling.
//   - Errors are domain sentinel errors (from domain/errors.go). The httperr
//     package maps them to HTTP status codes.
//   - Methods that modify state (Register, Login, ChangePassword, etc.)
//     run within a transaction (via TxRunner) and publish events via
//     EventPublisher atomically.
//   - The service does NOT validate JWT tokens — that's the middleware's
//     job. Methods that require authentication receive the actor's identity
//     via context (set by auth middleware).
//
// Imports: stdlib + domain only. DTOs use only primitive types.
package port

import (
        "context"
        "time"
)

// ===== Input DTOs =====

// RegisterUserInput holds the parameters for user registration.
type RegisterUserInput struct {
        Name     string
        Phone    string // raw phone, normalized by the service
        Password string
        Email    string // optional
        Locale   string // optional, defaults to "ar"
}

// RegisterDriverInput holds the parameters for driver registration.
type RegisterDriverInput struct {
        Name          string
        Phone         string
        Password      string
        VehicleType   string // motorcycle | car | scooter | bike
        LicenseNumber string
        NationalID    string
        Locale        string
        // AutoVerify — if true, the driver is created as verified + active
        // (used for seeding / admin-created accounts). Default: false.
        AutoVerify bool
}

// LoginInput holds the parameters for login (works for all identity types).
type LoginInput struct {
        Phone    string
        Password string
        IP       string // for audit/session tracking
        Agent    string // user-agent for session tracking
}

// ChangePasswordInput holds the parameters for a password change.
type ChangePasswordInput struct {
        SubjectID   string // user/driver/merchant/agent ID
        OldPassword string
        NewPassword string
}

// UpdateDriverStatusInput holds the parameters for a driver status update.
type UpdateDriverStatusInput struct {
        DriverID string
        Status   string // "online" | "offline"
        Lat      float64
        Lng      float64
}

// SuspendDriverInput holds the parameters for suspending a driver.
type SuspendDriverInput struct {
        DriverID    string
        Reason      string
        SuspendedBy string // admin user ID
}

// ===== Output DTOs =====

// UserDTO is the output representation of a User.
// Returned by GetUser, RegisterUser, LoginUser.
// Phone is the FULL phone number (not masked) — this DTO is only returned
// to the user themselves or to admins, never to other modules.
type UserDTO struct {
        ID            string    `json:"id"`
        Name          string    `json:"name"`
        Phone         string    `json:"phone"`
        Email         string    `json:"email"`
        LoyaltyPoints int       `json:"loyalty_points"`
        IsAdmin       bool      `json:"is_admin"`
        Locale        string    `json:"locale"`
        Timezone      string    `json:"timezone"`
        CreatedAt     time.Time `json:"created_at"`
}

// DriverProfileDTO is the output representation of a Driver.
// Returned by GetDriverProfile, driver login.
// Phone is masked — this DTO is returned to other modules (dispatch,
// support) that should not see the full phone number.
type DriverProfileDTO struct {
        ID                 string     `json:"id"`
        Name               string     `json:"name"`
        PhoneMasked        string     `json:"phone_masked"`
        VehicleType        string     `json:"vehicle_type"`
        TierID             string     `json:"tier_id,omitempty"`
        Status             string     `json:"status"`
        IsOnline           bool       `json:"is_online"`
        IsVerified         bool       `json:"is_verified"`
        IsActive           bool       `json:"is_active"`
        Lat                float64    `json:"lat,omitempty"`
        Lng                float64    `json:"lng,omitempty"`
        LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
        MustChangePassword bool       `json:"must_change_password"`
}

// MerchantProfileDTO is the output representation of a Merchant.
// Returned by GetMerchantProfile.
type MerchantProfileDTO struct {
        ID           string `json:"id"`
        RestaurantID string `json:"restaurant_id"`
        Name         string `json:"name"`
        IsActive     bool   `json:"is_active"`
}

// AuthResult holds the result of a successful authentication (login/register).
type AuthResult struct {
        Token string `json:"token"`
        // One of the following will be set, depending on the identity type:
        User   *UserDTO          `json:"user,omitempty"`
        Driver *DriverProfileDTO `json:"driver,omitempty"`
        // MustChangePassword indicates the user/driver must change password
        // before accessing other endpoints.
        MustChangePassword bool `json:"must_change_password"`
}

// ===== ServicePort Interface =====

// ServicePort is what the identity module exposes to the outside world.
// Other modules and the HTTP transport layer call methods on this interface.
//
// Methods are grouped by identity type for readability, but all live on
// a single interface — the identity service implements all of them.
//
// Concurrency: all methods are safe for concurrent use. The service does
// not hold locks; it relies on database transactions for isolation.
type ServicePort interface {

        // ----- User Authentication -----

        // RegisterUser creates a new user account.
        // Returns AuthResult with token + user DTO on success.
        // Returns ErrUserAlreadyExists if the phone is registered.
        // Returns ErrInvalidPhone, ErrNameTooShort, ErrPasswordTooShort for
        // validation failures.
        RegisterUser(ctx context.Context, input RegisterUserInput) (*AuthResult, error)

        // RegisterDriver creates a new driver account in identity.drivers.
        // If AutoVerify is true, the driver is created as verified + active.
        RegisterDriver(ctx context.Context, input RegisterDriverInput) (*AuthResult, error)

        // LoginUser authenticates a user by phone + password.
        // Returns AuthResult with token + user DTO on success.
        // Returns ErrInvalidCredentials on wrong phone or password (intentionally
        // does not distinguish to prevent user enumeration).
        LoginUser(ctx context.Context, input LoginInput) (*AuthResult, error)

        // Logout revokes the session associated with the given session ID.
        // Idempotent — returns nil if already revoked.
        Logout(ctx context.Context, sessionID string) error

        // ChangePassword changes a user's password.
        // Returns ErrPasswordMismatch if oldPassword is wrong.
        // Revokes all other sessions for the user after a successful change.
        ChangePassword(ctx context.Context, input ChangePasswordInput) error

        // GetUser retrieves a user by ID.
        // Returns ErrUserNotFound if not found.
        GetUser(ctx context.Context, userID string) (*UserDTO, error)

        // ----- Driver Authentication & Management -----

        // LoginDriver authenticates a driver by phone + password.
        // Returns AuthResult with token + driver DTO on success.
        // Returns ErrInvalidCredentials, ErrDriverNotActive, ErrDriverNotVerified
        // for respective failures.
        LoginDriver(ctx context.Context, input LoginInput) (*AuthResult, error)

        // ChangeDriverPassword changes a driver's password.
        ChangeDriverPassword(ctx context.Context, input ChangePasswordInput) error

        // GetDriverProfile retrieves a driver's profile by ID.
        // Returns ErrDriverNotFound if not found.
        GetDriverProfile(ctx context.Context, driverID string) (*DriverProfileDTO, error)

        // UpdateDriverStatus transitions a driver's status (online/offline).
        // Returns ErrInvalidDriverStatus for invalid transitions.
        // Returns ErrDriverSuspended, ErrDriverNotVerified, ErrDriverNotActive
        // for respective conditions when going online.
        UpdateDriverStatus(ctx context.Context, input UpdateDriverStatusInput) (*DriverProfileDTO, error)

        // SuspendDriver suspends a driver (admin operation).
        // Records who suspended, when, and why. Revokes all active sessions.
        // Idempotent — returns nil if already suspended.
        SuspendDriver(ctx context.Context, input SuspendDriverInput) error

        // ----- Merchant -----

        // GetMerchantProfile retrieves a merchant's profile by ID.
        // Returns ErrMerchantNotFound if not found.
        GetMerchantProfile(ctx context.Context, merchantID string) (*MerchantProfileDTO, error)

        // ----- Cross-Module Verification (used by other modules) -----

        // VerifyUserExists checks if a user with the given ID exists and is active.
        // Returns true if the user exists and is active, false otherwise.
        // Does NOT return an error for "not found" — only for infrastructure issues.
        VerifyUserExists(ctx context.Context, userID string) (bool, error)

        // VerifyDriverExists checks if a driver with the given ID exists and is active.
        // Returns true if the driver exists and is active, false otherwise.
        VerifyDriverExists(ctx context.Context, driverID string) (bool, error)

        // ----- Utility -----

        // HashPassword hashes a plaintext password. Exposed for use cases where
        // a hash is needed without going through the full register flow (e.g.
        // seeding, admin-created accounts).
        HashPassword(ctx context.Context, password string) (string, error)

        // ----- Setup (for initial system setup) -----

        // GetUserByPhone retrieves a user by phone number (string).
        GetUserByPhone(ctx context.Context, phone string) (*UserDTO, error)

        // GetDriverByPhone retrieves a driver by phone number (string).
        GetDriverByPhone(ctx context.Context, phone string) (*DriverProfileDTO, error)

        // PromoteUserToAdmin sets is_admin=true for the given user.
        PromoteUserToAdmin(ctx context.Context, userID string) error

        // VerifyDriverAccount sets is_verified=true and is_active=true for the given driver.
        VerifyDriverAccount(ctx context.Context, driverID string) error


        // AdminCreateDriver creates a driver in identity.drivers (verified + active)
        // AND returns the identity driver ID so the caller can register in dispatch.drivers.
        AdminCreateDriver(ctx context.Context, input AdminCreateDriverInput) (driverID string, err error)
}

// AdminCreateDriverInput holds all fields needed to create a complete driver.
type AdminCreateDriverInput struct {
        Name          string
        Phone         string
        Password      string
        VehicleType   string // motorcycle | car | scooter | bike
        LicenseNumber string
        NationalID    string
        LicensePlate  string
        ZoneIDs       []string
}
