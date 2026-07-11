// Package service auth: authentication use cases.
//
// Contains: RegisterUser, LoginUser, LoginDriver, Logout.
//
// Each use case runs within a transaction (via TxRunner) and publishes
// events via EventPublisher atomically. If any step fails, the transaction
// rolls back and no events are published.
package service

import (
        "context"
        "errors"
        "fmt"

        "avex-backend/internal/modules/identity/domain"
        "avex-backend/internal/modules/identity/port"
)

// minPasswordLength is the minimum password length enforced by the service.
// The domain does not enforce this (it only checks non-empty hash); the
// service enforces plaintext password rules before hashing.
const minPasswordLength = 6

// RegisterUser creates a new user account.
//
// Flow:
//  1. Validate input (name, phone, password length).
//  2. Hash password.
//  3. Create User entity.
//  4. Within transaction:
//     a. Persist user.
//     b. Create session.
//     c. Persist session.
//     d. Issue JWT.
//     e. Publish UserRegistered event to outbox.
//  5. Return AuthResult (token + user DTO).
func (s *Service) RegisterUser(ctx context.Context, input port.RegisterUserInput) (*port.AuthResult, error) {
        // --- Pre-transaction validation (no DB access) ---
        if len(input.Name) < 2 {
                return nil, domain.ErrNameTooShort
        }
        if len(input.Password) < minPasswordLength {
                return nil, domain.ErrPasswordTooShort
        }

        // Hash password (CPU-bound, do outside transaction).
        hash, err := s.deps.PasswordHasher.Hash(input.Password)
        if err != nil {
                return nil, fmt.Errorf("hash password: %w", err)
        }

        // Create User entity (validates phone, email, etc.).
        userID := s.deps.IDGenerator.New()
        now := s.deps.Clock.Now()
        user, err := domain.NewUser(domain.UserParams{
                ID:           userID,
                Name:         input.Name,
                Phone:        input.Phone,
                Email:        input.Email,
                PasswordHash: hash,
                Locale:       input.Locale,
                Now:          now,
        })
        if err != nil {
                return nil, err
        }

        // Event context for this operation.
        ec := s.eventContext(ctx, port.ActorContext{
                Type:      "system", // registration is self-initiated
                ID:        userID,
                IP:        "",
                UserAgent: "",
        })

        // --- Transaction boundary ---
        var token string
        err = s.deps.TxRunner.RunInTx(ctx, func(ctx context.Context, exec port.Executor) error {
                // Persist user.
                if err := s.deps.Repos.Users.Create(ctx, exec, user); err != nil {
                        return err
                }

                // Create session + issue JWT.
                _, t, err := s.createSessionAndToken(ctx, exec, userID, domain.RoleUser, "", "")
                if err != nil {
                        return err
                }
                token = t

                // Publish UserRegistered event to outbox (same transaction).
                err = s.deps.EventPublisher.PublishUserRegistered(ctx, exec, port.UserRegisteredPayload{
                        UserID:      userID,
                        PhoneMasked: user.Phone().Masked(),
                        Name:        user.Name(),
                        Locale:      user.Locale(),
                }, ec)
                if err != nil {
                        return fmt.Errorf("publish user registered: %w", err)
                }
                return nil
        })
        if err != nil {
                return nil, err
        }

        return &port.AuthResult{
                Token: token,
                User:  ptr(toUserDTO(user)),
        }, nil
}

// LoginUser authenticates a user by phone + password.
//
// Returns ErrInvalidCredentials on wrong phone or password (does not
// distinguish to prevent user enumeration).
func (s *Service) LoginUser(ctx context.Context, input port.LoginInput) (*port.AuthResult, error) {
        // Normalize phone for lookup.
        phone, err := domain.NewPhone(input.Phone)
        if err != nil {
                return nil, domain.ErrInvalidCredentials
        }

        // Fetch user by phone (non-transactional read).
        user, err := s.deps.Repos.Users.GetByPhone(ctx, s.pool, phone)
        if err != nil {
                if errors.Is(err, domain.ErrUserNotFound) {
                        return nil, domain.ErrInvalidCredentials
                }
                return nil, err
        }

        // Verify password.
        if err := s.deps.PasswordHasher.Compare(user.PasswordHash(), input.Password); err != nil {
                return nil, domain.ErrInvalidCredentials
        }

        // Check account is active.
        if !user.IsActive() {
                return nil, domain.ErrUserDeactivated
        }

        // Event context.
        ec := s.eventContext(ctx, port.ActorContext{
                Type:      "user",
                ID:        user.ID(),
                IP:        input.IP,
                UserAgent: input.Agent,
        })

        // --- Transaction boundary ---
        var token string
        var sessionID string
        err = s.deps.TxRunner.RunInTx(ctx, func(ctx context.Context, exec port.Executor) error {
                // Determine role: admin users get "admin" role, others get "user".
                role := domain.RoleUser
                if user.IsAdmin() {
                        role = domain.RoleAdmin
                }
                // Create session + issue JWT.
                sid, t, err := s.createSessionAndToken(ctx, exec, user.ID(), role, input.IP, input.Agent)
                if err != nil {
                        return err
                }
                token = t
                sessionID = sid

                // Publish UserLoggedIn event.
                return s.deps.EventPublisher.PublishUserLoggedIn(ctx, exec, port.UserLoggedInPayload{
                        UserID:    user.ID(),
                        SessionID: sessionID,
                        IP:        input.IP,
                        UserAgent: input.Agent,
                }, ec)
        })
        if err != nil {
                return nil, err
        }

        return &port.AuthResult{
                Token:              token,
                User:               ptr(toUserDTO(*user)),
                MustChangePassword: false,
        }, nil
}

// RegisterDriver creates a new driver account in identity.drivers.
// If AutoVerify is true, the driver is created as verified + active
// (used for seeding / admin-created accounts).
func (s *Service) RegisterDriver(ctx context.Context, input port.RegisterDriverInput) (*port.AuthResult, error) {
        if len(input.Name) < 2 {
                return nil, domain.ErrNameTooShort
        }
        if len(input.Password) < minPasswordLength {
                return nil, domain.ErrPasswordTooShort
        }

        hash, err := s.deps.PasswordHasher.Hash(input.Password)
        if err != nil {
                return nil, fmt.Errorf("hash password: %w", err)
        }

        vt := domain.VehicleType(input.VehicleType)
        if !vt.IsValid() {
                vt = domain.VehicleTypeMotorcycle
        }

        driverID := s.deps.IDGenerator.New()
        now := s.deps.Clock.Now()

        driver, err := domain.NewDriver(domain.DriverParams{
                ID:            driverID,
                Name:          input.Name,
                Phone:         input.Phone,
                PasswordHash:  hash,
                VehicleType:   vt,
                LicenseNumber: input.LicenseNumber,
                NationalID:    input.NationalID,
                Locale:        input.Locale,
                Now:           now,
        })
        if err != nil {
                return nil, err
        }

        // Auto-verify + clear must-change-password if requested (for seeding)
        if input.AutoVerify {
                if err := driver.Verify(now); err != nil {
                        return nil, err
                }
                driver.ClearMustChangePassword()
        }

        ec := s.eventContext(ctx, port.ActorContext{
                Type: "system",
                ID:   driverID,
        })

        var token string
        err = s.deps.TxRunner.RunInTx(ctx, func(ctx context.Context, exec port.Executor) error {
                if err := s.deps.Repos.Drivers.Create(ctx, exec, driver); err != nil {
                        return err
                }
                sid, t, err := s.createSessionAndToken(ctx, exec, driverID, domain.RoleDriver, "", "")
                if err != nil {
                        return err
                }
                token = t
                _ = sid

                return s.deps.EventPublisher.PublishDriverRegistered(ctx, exec, port.DriverRegisteredPayload{
                        DriverID: driverID,
                        Name:     driver.Name(),
                }, ec)
        })
        if err != nil {
                return nil, err
        }

        return &port.AuthResult{
                Token:              token,
                Driver:             ptr(toDriverProfileDTO(driver)),
                MustChangePassword: driver.MustChangePassword(),
        }, nil
}

// LoginDriver authenticates a driver by phone + password.
//
// Returns ErrInvalidCredentials on wrong phone or password.
// Returns ErrDriverNotActive if the driver account is inactive.
// Returns ErrDriverNotVerified if the driver has not been verified.
func (s *Service) LoginDriver(ctx context.Context, input port.LoginInput) (*port.AuthResult, error) {
        phone, err := domain.NewPhone(input.Phone)
        if err != nil {
                return nil, domain.ErrInvalidCredentials
        }

        driver, err := s.deps.Repos.Drivers.GetByPhone(ctx, s.pool, phone)
        if err != nil {
                if errors.Is(err, domain.ErrDriverNotFound) {
                        return nil, domain.ErrInvalidCredentials
                }
                return nil, err
        }

        if err := s.deps.PasswordHasher.Compare(driver.PasswordHash(), input.Password); err != nil {
                return nil, domain.ErrInvalidCredentials
        }

        if !driver.IsActive() {
                return nil, domain.ErrDriverNotActive
        }
        if !driver.IsVerified() {
                return nil, domain.ErrDriverNotVerified
        }

        ec := s.eventContext(ctx, port.ActorContext{
                Type:      "driver",
                ID:        driver.ID(),
                IP:        input.IP,
                UserAgent: input.Agent,
        })

        var token string
        var sessionID string
        err = s.deps.TxRunner.RunInTx(ctx, func(ctx context.Context, exec port.Executor) error {
                sid, t, err := s.createSessionAndToken(ctx, exec, driver.ID(), domain.RoleDriver, input.IP, input.Agent)
                if err != nil {
                        return err
                }
                token = t
                sessionID = sid

                // Publish UserLoggedIn event (reused for drivers — "user" here
                // means "identity subject", not specifically the user table).
                return s.deps.EventPublisher.PublishUserLoggedIn(ctx, exec, port.UserLoggedInPayload{
                        UserID:    driver.ID(),
                        SessionID: sessionID,
                        IP:        input.IP,
                        UserAgent: input.Agent,
                }, ec)
        })
        if err != nil {
                return nil, err
        }

        return &port.AuthResult{
                Token:              token,
                Driver:             ptr(toDriverProfileDTO(*driver)),
                MustChangePassword: driver.MustChangePassword(),
        }, nil
}

// Logout revokes the session associated with the given session ID.
// Idempotent — returns nil if already revoked.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
        now := s.deps.Clock.Now()

        // Fetch session to get subject info for the event.
        session, err := s.deps.Repos.Sessions.GetByID(ctx, s.pool, sessionID)
        if err != nil {
                if errors.Is(err, domain.ErrSessionNotFound) {
                        return nil // idempotent: already gone
                }
                return err
        }

        ec := s.eventContext(ctx, port.ActorContext{
                Type: session.SubjectType().String(),
                ID:   session.SubjectID(),
        })

        return s.deps.TxRunner.RunInTx(ctx, func(ctx context.Context, exec port.Executor) error {
                if err := s.deps.Repos.Sessions.Revoke(ctx, exec, sessionID, now); err != nil {
                        // Idempotent: if already revoked, return nil.
                        if errors.Is(err, domain.ErrSessionAlreadyRevoked) {
                                return nil
                        }
                        return err
                }

                // Publish UserLoggedOut event (works for any subject type).
                return s.deps.EventPublisher.PublishUserLoggedOut(ctx, exec, port.UserLoggedOutPayload{
                        UserID:    session.SubjectID(),
                        SessionID: sessionID,
                }, ec)
        })
}

// ptr returns a pointer to the given value. Used to take addresses of
// DTOs returned by value.
func ptr[T any](v T) *T {
        return &v
}
