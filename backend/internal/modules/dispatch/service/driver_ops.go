// Package service driver_ops: driver registration + lifecycle + location.
package service

import (
        "context"
        "errors"
        "fmt"
        "time"

        "avex-backend/internal/modules/dispatch/domain"
        "avex-backend/internal/modules/dispatch/events"
        "avex-backend/internal/modules/dispatch/port"
)

// ===== RegisterDriver =====

func (s *Service) RegisterDriver(ctx context.Context, input port.RegisterDriverInput) (*port.DriverDTO, error) {
        now := s.deps.Clock.Now()
        id := s.deps.IDGenerator.NewID()

        d, err := domain.NewDriver(id, input.UserID, domain.VehicleType(input.VehicleType), input.LicensePlate, input.ZoneIDs, now)
        if err != nil {
                return nil, err
        }

        if err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                if err := s.deps.Repos.Drivers.Create(ctx, exec, d); err != nil {
                        return err
                }
                // Publish driver.registered event
                ec := s.eventContext(ctx, port.ActorContext{Type: "system", ID: input.UserID})
                envelope, err := events.DriverRegisteredEnvelope(port.DriverRegisteredPayload{
                        DriverID:    d.ID(),
                        UserID:      d.UserID(),
                        VehicleType: string(d.VehicleType()),
                }, ec)
                if err != nil {
                        return err
                }
                return s.deps.EventPublisher.Publish(ctx, exec, envelope)
        }); err != nil {
                return nil, err
        }

        return port.ToDriverDTOPtr(d), nil
}

// ===== GetDriver / GetDriverByUserID =====

func (s *Service) GetDriver(ctx context.Context, id string) (*port.DriverDTO, error) {
        d, err := s.deps.Repos.Drivers.GetByID(ctx, s.pool, id)
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*d), nil
}

func (s *Service) GetDriverByUserID(ctx context.Context, userID string) (*port.DriverDTO, error) {
        d, err := s.deps.Repos.Drivers.GetByUserID(ctx, s.pool, userID)
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*d), nil
}

// ===== GoOnline / GoOffline =====

func (s *Service) GoOnline(ctx context.Context, driverID string) (*port.DriverDTO, error) {
        var updated *domain.Driver
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, driverID)
                if err != nil {
                        return err
                }
                online, err := d.GoOnline(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Drivers.Update(ctx, exec, online); err != nil {
                        return err
                }
                // Publish driver.online event
                ec := s.eventContext(ctx, port.ActorContext{Type: "driver", ID: driverID})
                envelope, err := events.DriverWentOnlineEnvelope(port.DriverWentOnlinePayload{
                        DriverID: online.ID(),
                        ZoneIDs:  online.ZoneIDs(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &online
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*updated), nil
}

func (s *Service) GoOffline(ctx context.Context, driverID string) (*port.DriverDTO, error) {
        var updated *domain.Driver
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, driverID)
                if err != nil {
                        return err
                }
                offline, err := d.GoOffline(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Drivers.Update(ctx, exec, offline); err != nil {
                        return err
                }
                // Remove driver from locations table so they're not matched
                _ = s.deps.Repos.Locations.DeleteByDriver(ctx, exec, driverID)

                // Publish driver.offline event
                ec := s.eventContext(ctx, port.ActorContext{Type: "driver", ID: driverID})
                envelope, err := events.DriverWentOfflineEnvelope(port.DriverWentOfflinePayload{
                        DriverID: offline.ID(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &offline
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*updated), nil
}

// ===== Suspend / Unsuspend =====

func (s *Service) SuspendDriver(ctx context.Context, id, reason string) (*port.DriverDTO, error) {
        var updated *domain.Driver
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, id)
                if err != nil {
                        return err
                }
                suspended, err := d.Suspend(reason, s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Drivers.Update(ctx, exec, suspended); err != nil {
                        return err
                }
                // Remove from locations
                _ = s.deps.Repos.Locations.DeleteByDriver(ctx, exec, id)

                ec := s.eventContext(ctx, port.ActorContext{Type: "admin"})
                envelope, err := events.DriverSuspendedEnvelope(port.DriverSuspendedPayload{
                        DriverID: suspended.ID(),
                        Reason:   reason,
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &suspended
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*updated), nil
}

func (s *Service) UnsuspendDriver(ctx context.Context, id string) (*port.DriverDTO, error) {
        var updated *domain.Driver
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, id)
                if err != nil {
                        return err
                }
                offline, err := d.Unsuspend(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Drivers.Update(ctx, exec, offline); err != nil {
                        return err
                }
                updated = &offline
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDriverDTOPtr(*updated), nil
}

// ===== ListDrivers / ListOnlineDrivers =====

func (s *Service) ListDrivers(ctx context.Context, page port.PageQuery) (port.Page[port.DriverDTO], error) {
        result, err := s.deps.Repos.Drivers.ListAll(ctx, s.pool, page)
        if err != nil {
                return port.Page[port.DriverDTO]{}, err
        }
        dtos := make([]port.DriverDTO, 0, len(result.Items))
        for _, d := range result.Items {
                dtos = append(dtos, port.ToDriverDTO(d))
        }
        return port.Page[port.DriverDTO]{
                Items:  dtos,
                Total:  result.Total,
                Limit:  result.Limit,
                Offset: result.Offset,
        }, nil
}

func (s *Service) ListOnlineDrivers(ctx context.Context, zoneID string) ([]port.DriverDTO, error) {
        drivers, err := s.deps.Repos.Drivers.ListOnlineByZone(ctx, s.pool, zoneID)
        if err != nil {
                return nil, err
        }
        dtos := make([]port.DriverDTO, 0, len(drivers))
        for _, d := range drivers {
                dtos = append(dtos, port.ToDriverDTO(d))
        }
        return dtos, nil
}

// ===== UpdateLocation / GetLocation / FindNearestDrivers =====

func (s *Service) UpdateLocation(ctx context.Context, input port.UpdateLocationInput) (*port.LocationDTO, error) {
        now := s.deps.Clock.Now()
        capturedAt := input.CapturedAt
        if capturedAt.IsZero() {
                capturedAt = now
        }

        loc, err := domain.NewDriverLocation(
                s.deps.IDGenerator.NewID(), input.DriverID,
                input.Lat, input.Lng, input.Bearing, input.Speed, input.Accuracy,
                capturedAt, now,
        )
        if err != nil {
                return nil, err
        }

        // Verify the driver exists and is online (not offline).
        d, err := s.deps.Repos.Drivers.GetByID(ctx, s.pool, input.DriverID)
        if err != nil {
                return nil, err
        }
        if d.Status() == domain.DriverStatusOffline || d.Status() == domain.DriverStatusSuspended {
                return nil, domain.ErrDriverOffline
        }

        // Upsert location (no transaction needed — single statement).
        if err := s.deps.Repos.Locations.Upsert(ctx, s.pool, loc); err != nil {
                return nil, err
        }

        // Best-effort event publish (non-fatal if it fails).
        ec := s.eventContext(ctx, port.ActorContext{Type: "driver", ID: input.DriverID})
        envelope, err := events.DriverLocationUpdatedEnvelope(port.DriverLocationUpdatedPayload{
                DriverID:   input.DriverID,
                Lat:        input.Lat,
                Lng:        input.Lng,
                Bearing:    input.Bearing,
                Speed:      input.Speed,
                CapturedAt: capturedAt.Format(time.RFC3339),
        }, ec)
        if err == nil {
                _ = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                        return s.deps.EventPublisher.Publish(ctx, exec, envelope)
                })
        }

        return port.ToLocationDTOPtr(loc), nil
}

func (s *Service) GetLocation(ctx context.Context, driverID string) (*port.LocationDTO, error) {
        loc, err := s.deps.Repos.Locations.GetByDriver(ctx, s.pool, driverID)
        if err != nil {
                return nil, err
        }
        return port.ToLocationDTOPtr(*loc), nil
}

func (s *Service) FindNearestDrivers(ctx context.Context, lat, lng float64, radiusM int, limit int) ([]port.NearbyDriver, error) {
        if radiusM <= 0 {
                radiusM = s.cfg.DefaultSearchRadiusM
        }
        if limit <= 0 {
                limit = 10
        }
        return s.deps.Repos.Locations.FindNearestDrivers(ctx, s.pool, lat, lng, radiusM, s.cfg.LocationStaleTTL, limit)
}

// ===== DriverOrderCompleted =====

// DriverOrderCompleted transitions the driver from busy → online and
// updates the rating. Called by the orders module when an order is delivered.
func (s *Service) DriverOrderCompleted(ctx context.Context, driverID, orderID string) error {
        return s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, driverID)
                if err != nil {
                        return err
                }
                if d.CurrentOrderID() != orderID {
                        return fmt.Errorf("%w: driver %s is not on order %s (on %s)", domain.ErrDriverNotEligible, driverID, orderID, d.CurrentOrderID())
                }
                completed, err := d.CompleteOrder(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                return s.deps.Repos.Drivers.Update(ctx, exec, completed)
        })
}

// suppress unused imports
var _ = errors.Is
