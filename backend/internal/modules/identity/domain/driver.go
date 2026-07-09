// Package domain driver: Driver entity with business invariants and
// state transitions.
//
// A Driver is a delivery driver who can accept dispatch offers and
// deliver orders. Drivers authenticate with phone + password and must
// be verified by an admin before going online.
//
// Invariants enforced by this entity:
//   - ID must be non-empty
//   - Name must be >= 2 chars
//   - Phone must be a valid Egyptian mobile (normalized)
//   - PasswordHash must be non-empty
//   - NationalID must be non-empty
//   - LicenseNumber must be non-empty
//   - VehicleType must be a recognized value (motorcycle, car, bicycle, scooter)
//   - Status transitions must follow the state machine (see status.go)
//   - A suspended driver cannot go online (must be unsuspended first)
//   - An unverified driver cannot go online
//   - An inactive driver cannot go online
//   - Going online requires a location (lat, lng)
//   - tierID is optional (starter tier assigned when null at the dispatch layer)
//
// Imports stdlib only.
package domain

import (
        "fmt"
        "strings"
        "time"
)

// VehicleType represents the type of vehicle a driver uses.
type VehicleType string

const (
        VehicleTypeMotorcycle VehicleType = "motorcycle"
        VehicleTypeCar        VehicleType = "car"
        VehicleTypeBicycle    VehicleType = "bicycle"
        VehicleTypeScooter    VehicleType = "scooter"
)

// IsValid reports whether the vehicle type is recognized.
func (v VehicleType) IsValid() bool {
        switch v {
        case VehicleTypeMotorcycle, VehicleTypeCar, VehicleTypeBicycle, VehicleTypeScooter:
                return true
        }
        return false
}

// String returns the string representation.
func (v VehicleType) String() string {
        return string(v)
}

// AllVehicleTypes returns all valid vehicle types.
func AllVehicleTypes() []VehicleType {
        return []VehicleType{
                VehicleTypeMotorcycle,
                VehicleTypeCar,
                VehicleTypeBicycle,
                VehicleTypeScooter,
        }
}

// Location holds a geographic coordinate.
type Location struct {
        Lat float64
        Lng float64
}

// IsZero reports whether the location is unset (0, 0).
func (l Location) IsZero() bool {
        return l.Lat == 0 && l.Lng == 0
}

// Driver represents a delivery driver.
type Driver struct {
        id                 string
        name               string
        phone              Phone
        passwordHash       string
        vehicleType        VehicleType
        licenseNumber      string
        nationalID         string
        tierID             string // soft ref to financial.driver_tiers.id, may be ""
        status             DriverStatus
        isActive           bool
        isVerified         bool
        mustChangePassword bool
        location           Location
        locationUpdatedAt  *time.Time
        lastSeenAt         *time.Time
        shiftStart         *time.Time
        autoAccept         bool
        suspendedAt        *time.Time
        suspendedReason    string
        suspendedBy        string // admin user ID
        locale             string
        timezone           string
        createdAt          time.Time
        updatedAt          time.Time
}

// ----- Constructor -----

// DriverParams holds required parameters for creating a new driver.
type DriverParams struct {
        ID            string
        Name          string
        Phone         string
        PasswordHash  string
        VehicleType   VehicleType
        LicenseNumber string
        NationalID    string
        Locale        string
        Timezone      string
        Now           time.Time
}

// NewDriver creates a new Driver with the given parameters.
// New drivers start in the Offline status, are not verified, and must
// change their password on first login.
func NewDriver(params DriverParams) (Driver, error) {
        if params.ID == "" {
                return Driver{}, NewValidationError("id", ErrInvalidID)
        }

        name := strings.TrimSpace(params.Name)
        if len(name) < 2 {
                return Driver{}, NewValidationError("name", ErrNameTooShort)
        }

        phone, err := NewPhone(params.Phone)
        if err != nil {
                return Driver{}, NewValidationError("phone", err)
        }

        if params.PasswordHash == "" {
                return Driver{}, NewValidationError("password_hash", ErrInvalidInput)
        }

        if !params.VehicleType.IsValid() {
                return Driver{}, NewValidationError("vehicle_type", ErrInvalidVehicleType)
        }

        license := strings.TrimSpace(params.LicenseNumber)
        if license == "" {
                return Driver{}, NewValidationError("license_number", ErrInvalidLicenseNumber)
        }

        nationalID := strings.TrimSpace(params.NationalID)
        if nationalID == "" {
                return Driver{}, NewValidationError("national_id", ErrInvalidNationalID)
        }

        locale := strings.TrimSpace(params.Locale)
        if locale == "" {
                locale = "ar"
        }

        timezone := strings.TrimSpace(params.Timezone)
        if timezone == "" {
                timezone = "Africa/Cairo"
        }

        now := params.Now
        if now.IsZero() {
                now = time.Now().UTC()
        }

        return Driver{
                id:                 params.ID,
                name:               name,
                phone:              phone,
                passwordHash:       params.PasswordHash,
                vehicleType:        params.VehicleType,
                licenseNumber:      license,
                nationalID:         nationalID,
                tierID:             "",
                status:             DriverStatusOffline,
                isActive:           true,
                isVerified:         false,
                mustChangePassword: true,
                location:           Location{},
                locale:             locale,
                timezone:           timezone,
                createdAt:          now,
                updatedAt:          now,
        }, nil
}

// ----- Reconstruction -----

// DriverRecord holds all fields to rebuild a Driver from persistence.
type DriverRecord struct {
        ID                 string
        Name               string
        Phone              string
        PasswordHash       string
        VehicleType        VehicleType
        LicenseNumber      string
        NationalID         string
        TierID             string
        Status             DriverStatus
        IsActive           bool
        IsVerified         bool
        MustChangePassword bool
        Location           Location
        LocationUpdatedAt  *time.Time
        LastSeenAt         *time.Time
        ShiftStart         *time.Time
        AutoAccept         bool
        SuspendedAt        *time.Time
        SuspendedReason    string
        SuspendedBy        string
        Locale             string
        Timezone           string
        CreatedAt          time.Time
        UpdatedAt          time.Time
}

// ReconstructDriver rebuilds a Driver from persistence (no validation).
func ReconstructDriver(rec DriverRecord) Driver {
        phone, _ := NewPhone(rec.Phone)
        return Driver{
                id:                 rec.ID,
                name:               rec.Name,
                phone:              phone,
                passwordHash:       rec.PasswordHash,
                vehicleType:        rec.VehicleType,
                licenseNumber:      rec.LicenseNumber,
                nationalID:         rec.NationalID,
                tierID:             rec.TierID,
                status:             rec.Status,
                isActive:           rec.IsActive,
                isVerified:         rec.IsVerified,
                mustChangePassword: rec.MustChangePassword,
                location:           rec.Location,
                locationUpdatedAt:  rec.LocationUpdatedAt,
                lastSeenAt:         rec.LastSeenAt,
                shiftStart:         rec.ShiftStart,
                autoAccept:         rec.AutoAccept,
                suspendedAt:        rec.SuspendedAt,
                suspendedReason:    rec.SuspendedReason,
                suspendedBy:        rec.SuspendedBy,
                locale:             rec.Locale,
                timezone:           rec.Timezone,
                createdAt:          rec.CreatedAt,
                updatedAt:          rec.UpdatedAt,
        }
}

// ----- Getters -----

func (d Driver) ID() string                    { return d.id }
func (d Driver) Name() string                  { return d.name }
func (d Driver) Phone() Phone                  { return d.phone }
func (d Driver) PasswordHash() string          { return d.passwordHash }
func (d Driver) VehicleType() VehicleType      { return d.vehicleType }
func (d Driver) LicenseNumber() string         { return d.licenseNumber }
func (d Driver) NationalID() string            { return d.nationalID }
func (d Driver) TierID() string                { return d.tierID }
func (d Driver) Status() DriverStatus          { return d.status }
func (d Driver) IsActive() bool                { return d.isActive }
func (d Driver) IsVerified() bool              { return d.isVerified }
func (d Driver) MustChangePassword() bool      { return d.mustChangePassword }
func (d Driver) Location() Location            { return d.location }
func (d Driver) LocationUpdatedAt() *time.Time { return d.locationUpdatedAt }
func (d Driver) LastSeenAt() *time.Time        { return d.lastSeenAt }
func (d Driver) ShiftStart() *time.Time        { return d.shiftStart }
func (d Driver) AutoAccept() bool              { return d.autoAccept }
func (d Driver) SuspendedAt() *time.Time       { return d.suspendedAt }
func (d Driver) SuspendedReason() string       { return d.suspendedReason }
func (d Driver) SuspendedBy() string           { return d.suspendedBy }
func (d Driver) Locale() string                { return d.locale }
func (d Driver) Timezone() string              { return d.timezone }
func (d Driver) CreatedAt() time.Time          { return d.createdAt }
func (d Driver) UpdatedAt() time.Time          { return d.updatedAt }

// IsOnline is a convenience method for the dispatch layer.
func (d Driver) IsOnline() bool { return d.status.IsOnline() }

// IsSuspended reports whether the driver is suspended.
func (d Driver) IsSuspended() bool { return d.status.IsSuspended() }

// IsAvailableForDispatch reports whether the driver can receive dispatch offers.
// A driver is available if: active AND verified AND online AND has a fresh location.
func (d Driver) IsAvailableForDispatch(now time.Time, staleSeconds int) bool {
        if !d.isActive || !d.isVerified || !d.IsOnline() {
                return false
        }
        if d.location.IsZero() || d.locationUpdatedAt == nil {
                return false
        }
        staleAfter := d.locationUpdatedAt.Add(time.Duration(staleSeconds) * time.Second)
        return now.Before(staleAfter)
}

// ----- Behavior (mutations) -----

// ChangePassword updates the password hash and clears the must-change flag.
func (d *Driver) ChangePassword(newHash string, now time.Time) error {
        if newHash == "" {
                return NewValidationError("password_hash", ErrInvalidInput)
        }
        d.passwordHash = newHash
        d.mustChangePassword = false
        d.updatedAt = now
        return nil
}

// ClearMustChangePassword clears the must-change-password flag without
// changing the password. Used for seeded/admin-created accounts.
func (d *Driver) ClearMustChangePassword() {
        d.mustChangePassword = false
}

// Verify marks the driver as verified by an admin.
// Returns an error if already verified.
func (d *Driver) Verify(now time.Time) error {
        if d.isVerified {
                return nil // idempotent
        }
        d.isVerified = true
        d.updatedAt = now
        return nil
}

// SetTier assigns the driver to a tier.
// The tierID is a soft reference — validation happens at the port layer.
func (d *Driver) SetTier(tierID string, now time.Time) {
        if tierID != "" {
                d.tierID = tierID
                d.updatedAt = now
        }
}

// GoOnline transitions the driver from Offline to Online.
// Returns an error if:
//   - The driver is suspended (must be unsuspended first)
//   - The driver is not verified
//   - The driver is not active
//   - No location is provided
//   - Already online
func (d *Driver) GoOnline(loc Location, now time.Time) error {
        if d.IsSuspended() {
                return ErrDriverSuspended
        }
        if !d.isVerified {
                return ErrDriverNotVerified
        }
        if !d.isActive {
                return ErrDriverNotActive
        }
        if loc.IsZero() {
                return ErrLocationRequired
        }

        newStatus, err := d.status.Transition(DriverStatusOnline)
        if err != nil {
                return err
        }

        d.status = newStatus
        d.location = loc
        d.locationUpdatedAt = &now
        d.lastSeenAt = &now
        d.shiftStart = &now
        d.updatedAt = now
        return nil
}

// GoOffline transitions the driver from Online to Offline.
func (d *Driver) GoOffline(now time.Time) error {
        newStatus, err := d.status.Transition(DriverStatusOffline)
        if err != nil {
                return err
        }
        d.status = newStatus
        d.shiftStart = nil
        d.updatedAt = now
        return nil
}

// UpdateLocation updates the driver's current location and last-seen timestamp.
// Only valid while online.
func (d *Driver) UpdateLocation(loc Location, now time.Time) error {
        if !d.IsOnline() {
                return ErrInvalidDriverStatus
        }
        if loc.IsZero() {
                return ErrLocationRequired
        }
        d.location = loc
        d.locationUpdatedAt = &now
        d.lastSeenAt = &now
        return nil
}

// UpdateLastSeen refreshes the last-seen timestamp without changing location.
func (d *Driver) UpdateLastSeen(now time.Time) {
        d.lastSeenAt = &now
}

// SetAutoAccept toggles auto-accept for dispatch offers.
func (d *Driver) SetAutoAccept(enabled bool, now time.Time) {
        d.autoAccept = enabled
        d.updatedAt = now
}

// Suspend transitions the driver to Suspended status.
// Records who suspended, when, and why.
func (d *Driver) Suspend(reason, suspendedBy string, now time.Time) error {
        if d.IsSuspended() {
                return nil // idempotent
        }

        newStatus, err := d.status.Transition(DriverStatusSuspended)
        if err != nil {
                return err
        }

        d.status = newStatus
        d.suspendedAt = &now
        d.suspendedReason = strings.TrimSpace(reason)
        d.suspendedBy = suspendedBy
        d.shiftStart = nil
        d.updatedAt = now
        return nil
}

// Unsuspend transitions the driver from Suspended to Offline.
// The driver must explicitly go online again to receive offers.
func (d *Driver) Unsuspend(now time.Time) error {
        if !d.IsSuspended() {
                return ErrInvalidDriverStatus
        }

        newStatus, err := d.status.Transition(DriverStatusOffline)
        if err != nil {
                return err
        }

        d.status = newStatus
        d.suspendedAt = nil
        d.suspendedReason = ""
        d.suspendedBy = ""
        d.updatedAt = now
        return nil
}

// Deactivate sets isActive = false. The driver cannot go online while inactive.
func (d *Driver) Deactivate(now time.Time) {
        d.isActive = false
        d.updatedAt = now
}

// Reactivate sets isActive = true.
func (d *Driver) Reactivate(now time.Time) {
        d.isActive = true
        d.updatedAt = now
}

// UpdateProfile updates name and locale.
func (d *Driver) UpdateProfile(name string, now time.Time) error {
        if name != "" {
                if len(strings.TrimSpace(name)) < 2 {
                        return NewValidationError("name", ErrNameTooShort)
                }
                d.name = strings.TrimSpace(name)
        }
        d.updatedAt = now
        return nil
}

// ----- String representation (no PII) -----

func (d Driver) String() string {
        return fmt.Sprintf("Driver{id=%s, name=%s, phone=%s, status=%s, verified=%v, active=%v, tier=%s}",
                d.id, d.name, d.phone.Masked(), d.status, d.isVerified, d.isActive, d.tierID)
}
