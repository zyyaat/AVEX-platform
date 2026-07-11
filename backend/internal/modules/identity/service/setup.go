// Package service setup: initial system setup helpers.
// These methods are used by the setup endpoints to promote users to admin
// and verify drivers. They should only be used during initial setup.
package service

import (
	"context"
	"fmt"
	"time"

	"avex-backend/internal/modules/identity/domain"
	"avex-backend/internal/modules/identity/port"
)

// GetUserByPhone retrieves a user by phone number.
func (s *Service) GetUserByPhone(ctx context.Context, phoneStr string) (*port.UserDTO, error) {
	phone, err := domain.NewPhone(phoneStr)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	user, err := s.deps.Repos.Users.GetByPhone(ctx, s.pool, phone)
	if err != nil {
		return nil, err
	}
	dto := toUserDTO(*user)
	return &dto, nil
}

// GetDriverByPhone retrieves a driver by phone number.
func (s *Service) GetDriverByPhone(ctx context.Context, phoneStr string) (*port.DriverProfileDTO, error) {
	phone, err := domain.NewPhone(phoneStr)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	driver, err := s.deps.Repos.Drivers.GetByPhone(ctx, s.pool, phone)
	if err != nil {
		return nil, err
	}
	dto := toDriverProfileDTO(*driver)
	return &dto, nil
}

// PromoteUserToAdmin sets is_admin=true for the given user.
func (s *Service) PromoteUserToAdmin(ctx context.Context, userID string) error {
	user, err := s.deps.Repos.Users.GetByID(ctx, s.pool, userID)
	if err != nil {
		return err
	}
	user.PromoteToAdmin(time.Now())
	return s.deps.Repos.Users.Update(ctx, s.pool, *user)
}

// VerifyDriverAccount sets is_verified=true and is_active=true for the given driver.
func (s *Service) VerifyDriverAccount(ctx context.Context, driverID string) error {
	driver, err := s.deps.Repos.Drivers.GetByID(ctx, s.pool, driverID)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := driver.Verify(now); err != nil {
		// Already verified — not an error
	}
	driver.ClearMustChangePassword()
	return s.deps.Repos.Drivers.Update(ctx, s.pool, *driver)
}

// suppress unused import
var _ = fmt.Errorf

// AdminCreateDriver creates a driver in identity.drivers (verified + active)
// and returns the identity driver ID. The caller (admin handler) then
// registers the driver in dispatch.drivers using the dispatch service.
func (s *Service) AdminCreateDriver(ctx context.Context, input port.AdminCreateDriverInput) (string, error) {
	// Register the driver in identity.drivers with auto-verify
	result, err := s.RegisterDriver(ctx, port.RegisterDriverInput{
		Name:          input.Name,
		Phone:         input.Phone,
		Password:      input.Password,
		VehicleType:   input.VehicleType,
		LicenseNumber: input.LicenseNumber,
		NationalID:    input.NationalID,
		AutoVerify:    true,
	})
	if err != nil {
		return "", err
	}
	if result.Driver == nil {
		return "", fmt.Errorf("driver creation succeeded but driver DTO is nil")
	}
	return result.Driver.ID, nil
}
