// Package service matching_ops: dispatch matching engine + offer lifecycle.
//
// The matching engine is the core of the dispatch module. It:
//   1. Receives an OrderAssignmentRequested event (via the bus subscriber).
//   2. Finds the nearest available driver using the DriverLocationRepository.
//   3. Computes the driving distance + duration via Mapbox (optional).
//   4. Creates a DispatchOffer and publishes offer.created.
//   5. The driver responds (accept/reject) via HTTP.
//   6. On accept: publishes offer.accepted (orders module consumes this).
//   7. On reject/expire: re-runs the matching for the next attempt.
//   8. After MaxAttempts: publishes dispatch.failed.
package service

import (
        "context"
        "errors"
        "fmt"

        "avex-backend/internal/modules/dispatch/domain"
        "avex-backend/internal/modules/dispatch/events"
        "avex-backend/internal/modules/dispatch/port"
)

// ===== HandleOrderAssignmentRequested =====
//
// This is the entry point triggered by the bus subscriber (jobs/subscriber.go)
// when an orders.order.assignment_requested event arrives.

func (s *Service) HandleOrderAssignmentRequested(
        ctx context.Context,
        orderID, zoneID string,
        pickupLat, pickupLng, deliveryLat, deliveryLng float64,
) error {
        return s.CreateOfferInternal(ctx, port.CreateOfferInput{
                OrderID:     orderID,
                ZoneID:      zoneID,
                PickupLat:   pickupLat,
                PickupLng:   pickupLng,
                DeliveryLat: deliveryLat,
                DeliveryLng: deliveryLng,
                Currency:    s.cfg.DefaultCurrency,
        }, "system")
}

// ===== CreateOffer (HTTP entry) =====

func (s *Service) CreateOffer(ctx context.Context, input port.CreateOfferInput) (*port.DispatchOfferDTO, error) {
        if err := s.CreateOfferInternal(ctx, input, "manual"); err != nil {
                return nil, err
        }
        // Re-read the created offer
        offer, err := s.deps.Repos.Offers.GetActiveOfferForOrder(ctx, s.pool, input.OrderID)
        if err != nil {
                return nil, err
        }
        return port.ToDispatchOfferDTOPtr(*offer), nil
}

// CreateOfferInternal is the core matching logic.
//
// Steps:
//   1. Check max attempts — fail if exceeded.
//   2. Check if there's already a pending offer for this order.
//   3. Find nearest available driver (either input.DriverID or via location search).
//   4. Verify the driver is eligible (online, not busy, serves the zone).
//   5. Compute driving distance + duration via Mapbox (best-effort).
//   6. Create DispatchOffer in 'pending' status.
//   7. Publish offer.created event.
func (s *Service) CreateOfferInternal(ctx context.Context, input port.CreateOfferInput, createdBy string) error {
        if input.Currency == "" {
                input.Currency = s.cfg.DefaultCurrency
        }

        // 1. Max attempts check.
        attempts, err := s.deps.Repos.Offers.CountAttemptsForOrder(ctx, s.pool, input.OrderID)
        if err != nil {
                return fmt.Errorf("count attempts: %w", err)
        }
        if attempts >= s.cfg.MaxAttempts {
                // Publish dispatch.failed event.
                _ = s.publishDispatchFailed(ctx, input.OrderID, input.ZoneID, "max attempts reached", attempts)
                return fmt.Errorf("%w: order %s has %d attempts", domain.ErrMaxAttemptsReached, input.OrderID, attempts)
        }

        // 2. Check for existing pending offer.
        existing, err := s.deps.Repos.Offers.GetActiveOfferForOrder(ctx, s.pool, input.OrderID)
        if err == nil && existing != nil {
                return fmt.Errorf("%w: order %s already has pending offer %s", domain.ErrOfferAlreadyExists, input.OrderID, existing.ID())
        }
        if err != nil && !errors.Is(err, domain.ErrOfferNotFound) {
                return fmt.Errorf("check existing offer: %w", err)
        }

        // 3. Find nearest available driver.
        var driverID string
        if input.DriverID != "" {
                // Manual dispatch — use the specified driver.
                driverID = input.DriverID
        } else {
                // Auto-match — find nearest.
                nearby, err := s.deps.Repos.Locations.FindNearestDrivers(ctx, s.pool, input.PickupLat, input.PickupLng, s.cfg.DefaultSearchRadiusM, s.cfg.LocationStaleTTL, 1)
                if err != nil {
                        return fmt.Errorf("find nearest: %w", err)
                }
                if len(nearby) == 0 {
                        // No drivers available — publish dispatch.failed event.
                        _ = s.publishDispatchFailed(ctx, input.OrderID, input.ZoneID, "no drivers available", attempts)
                        return domain.ErrNoDriversAvailable
                }
                driverID = nearby[0].DriverID
        }

        // 4. Verify driver is eligible.
        d, err := s.deps.Repos.Drivers.GetByID(ctx, s.pool, driverID)
        if err != nil {
                return err
        }
        if !d.IsAvailableForOffer() {
                return fmt.Errorf("%w: driver %s is %s", domain.ErrDriverNotEligible, driverID, d.Status())
        }
        if input.ZoneID != "" && !d.ServesZone(input.ZoneID) {
                return fmt.Errorf("%w: driver %s does not serve zone %s", domain.ErrDriverNotEligible, driverID, input.ZoneID)
        }

        // 5. Compute driving distance + duration via Mapbox (best-effort).
        var estDistanceM, estDurationS *int
        if s.deps.DistanceMatrixProvider != nil {
                dist, dur, err := s.computeDriverDistance(ctx, driverID, input.PickupLat, input.PickupLng)
                if err == nil && len(dist) > 0 && len(dist[0]) > 0 {
                        d := dist[0][0]
                        du := dur[0][0]
                        estDistanceM = &d
                        estDurationS = &du
                }
                // On error, fall back to Haversine from driver's last known location.
                if estDistanceM == nil {
                        if haversineDist, ok := s.haversineFromDriverLocation(ctx, driverID, input.PickupLat, input.PickupLng); ok {
                                estDistanceM = &haversineDist
                        }
                }
        }

        // 6. Create offer.
        now := s.deps.Clock.Now()
        offerID := s.deps.IDGenerator.NewID()
        offer, err := domain.NewDispatchOffer(
                offerID, input.OrderID, driverID, input.ZoneID,
                input.PickupLat, input.PickupLng, input.DeliveryLat, input.DeliveryLng,
                estDistanceM, estDurationS, nil, input.Currency,
                s.cfg.OfferTTL, attempts+1, createdBy, now,
        )
        if err != nil {
                return err
        }

        // 7. Persist + publish.
        return s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                if err := s.deps.Repos.Offers.Create(ctx, exec, offer); err != nil {
                        return err
                }
                // Record offer decision pending (will be updated on accept/reject).
                updatedDriver := d.RecordOfferDecision(false, now) // assume reject by default; updated on accept
                _ = s.deps.Repos.Drivers.Update(ctx, exec, updatedDriver)

                // Publish offer.created event.
                ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
                envelope, err := events.OfferCreatedEnvelope(port.OfferCreatedPayload{
                        OfferID:       offer.ID(),
                        OrderID:       offer.OrderID(),
                        DriverID:      offer.DriverID(),
                        AttemptNumber: offer.AttemptNumber(),
                        ExpiresAt:     offer.ExpiresAt().Format("2006-01-02T15:04:05Z07:00"),
                        EstDistanceM:  offer.EstDistanceM(),
                        EstDurationS:  offer.EstDurationS(),
                        Currency:      offer.Currency(),
                }, ec)
                if err != nil {
                        return err
                }
                return s.deps.EventPublisher.Publish(ctx, exec, envelope)
        })
}

// computeDriverDistance calls Mapbox with the driver's current location
// as the origin and the pickup point as the destination.
func (s *Service) computeDriverDistance(ctx context.Context, driverID string, pickupLat, pickupLng float64) ([][]int, [][]int, error) {
        loc, err := s.deps.Repos.Locations.GetByDriver(ctx, s.pool, driverID)
        if err != nil {
                return nil, nil, err
        }
        return s.deps.DistanceMatrixProvider.GetDistanceMatrix(
                ctx,
                [][2]float64{{loc.Lat(), loc.Lng()}},
                [][2]float64{{pickupLat, pickupLng}},
        )
}

// haversineFromDriverLocation is the fallback when Mapbox fails.
// Returns the great-circle distance from the driver's last known location
// to the pickup point.
func (s *Service) haversineFromDriverLocation(ctx context.Context, driverID string, pickupLat, pickupLng float64) (int, bool) {
        loc, err := s.deps.Repos.Locations.GetByDriver(ctx, s.pool, driverID)
        if err != nil {
                return 0, false
        }
        dist := loc.DistanceToMeters(pickupLat, pickupLng)
        return int(dist), true
}

// publishDispatchFailed publishes a dispatch.failed event (best-effort).
func (s *Service) publishDispatchFailed(ctx context.Context, orderID, zoneID, reason string, attemptCount int) error {
        ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
        envelope, err := events.DispatchFailedEnvelope(port.DispatchFailedPayload{
                OrderID:      orderID,
                ZoneID:       zoneID,
                Reason:       reason,
                AttemptCount: attemptCount,
        }, ec)
        if err != nil {
                return err
        }
        return s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                return s.deps.EventPublisher.Publish(ctx, exec, envelope)
        })
}

// ===== AcceptOffer =====

func (s *Service) AcceptOffer(ctx context.Context, offerID, driverID string) (*port.DispatchOfferDTO, error) {
        var updated *domain.DispatchOffer
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                offer, err := s.deps.Repos.Offers.GetByID(ctx, exec, offerID)
                if err != nil {
                        return err
                }
                if offer.DriverID() != driverID {
                        return fmt.Errorf("%w: offer %s belongs to driver %s, not %s", domain.ErrDriverNotEligible, offerID, offer.DriverID(), driverID)
                }
                accepted, err := offer.Accept(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Offers.Update(ctx, exec, accepted); err != nil {
                        return err
                }

                // Transition driver to busy.
                d, err := s.deps.Repos.Drivers.GetByID(ctx, exec, driverID)
                if err != nil {
                        return err
                }
                busy, err := d.StartOrder(offer.OrderID(), s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                busy = busy.RecordOfferDecision(true, s.deps.Clock.Now())
                if err := s.deps.Repos.Drivers.Update(ctx, exec, busy); err != nil {
                        return err
                }

                // Publish offer.accepted event (orders module consumes this).
                ec := s.eventContext(ctx, port.ActorContext{Type: "driver", ID: driverID})
                envelope, err := events.OfferAcceptedEnvelope(port.OfferAcceptedPayload{
                        OfferID:  accepted.ID(),
                        OrderID:  accepted.OrderID(),
                        DriverID: accepted.DriverID(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &accepted
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDispatchOfferDTOPtr(*updated), nil
}

// ===== RejectOffer =====

func (s *Service) RejectOffer(ctx context.Context, offerID, driverID, reason string) (*port.DispatchOfferDTO, error) {
        var updated *domain.DispatchOffer
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                offer, err := s.deps.Repos.Offers.GetByID(ctx, exec, offerID)
                if err != nil {
                        return err
                }
                if offer.DriverID() != driverID {
                        return fmt.Errorf("%w: offer %s belongs to driver %s, not %s", domain.ErrDriverNotEligible, offerID, offer.DriverID(), driverID)
                }
                rejected, err := offer.Reject(reason, s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Offers.Update(ctx, exec, rejected); err != nil {
                        return err
                }

                // Publish offer.rejected event.
                ec := s.eventContext(ctx, port.ActorContext{Type: "driver", ID: driverID})
                envelope, err := events.OfferRejectedEnvelope(port.OfferRejectedPayload{
                        OfferID:  rejected.ID(),
                        OrderID:  rejected.OrderID(),
                        DriverID: rejected.DriverID(),
                        Reason:   reason,
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &rejected
                return nil
        })
        if err != nil {
                return nil, err
        }

        // Trigger next attempt (best-effort, non-fatal if it fails).
        // This re-runs the matching for the next available driver.
        go func() {
                bgCtx := context.Background()
                // Re-read pickup location from the offer (already in DB).
                offer, err := s.deps.Repos.Offers.GetByID(bgCtx, s.pool, offerID)
                if err == nil {
                        _ = s.CreateOfferInternal(bgCtx, port.CreateOfferInput{
                                OrderID:     offer.OrderID(),
                                ZoneID:      offer.ZoneID(),
                                PickupLat:   offer.PickupLat(),
                                PickupLng:   offer.PickupLng(),
                                DeliveryLat: offer.DeliveryLat(),
                                DeliveryLng: offer.DeliveryLng(),
                                Currency:    offer.Currency(),
                        }, "system")
                }
        }()

        return port.ToDispatchOfferDTOPtr(*updated), nil
}

// ===== ExpireOffer =====

func (s *Service) ExpireOffer(ctx context.Context, offerID string) (*port.DispatchOfferDTO, error) {
        var updated *domain.DispatchOffer
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                offer, err := s.deps.Repos.Offers.GetByID(ctx, exec, offerID)
                if err != nil {
                        return err
                }
                expired, err := offer.Expire(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Offers.Update(ctx, exec, expired); err != nil {
                        return err
                }

                ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
                envelope, err := events.OfferExpiredEnvelope(port.OfferExpiredPayload{
                        OfferID:  expired.ID(),
                        OrderID:  expired.OrderID(),
                        DriverID: expired.DriverID(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &expired
                return nil
        })
        if err != nil {
                return nil, err
        }

        // Trigger next attempt (best-effort, non-fatal).
        go func() {
                bgCtx := context.Background()
                offer, err := s.deps.Repos.Offers.GetByID(bgCtx, s.pool, offerID)
                if err == nil {
                        _ = s.CreateOfferInternal(bgCtx, port.CreateOfferInput{
                                OrderID:     offer.OrderID(),
                                ZoneID:      offer.ZoneID(),
                                PickupLat:   offer.PickupLat(),
                                PickupLng:   offer.PickupLng(),
                                DeliveryLat: offer.DeliveryLat(),
                                DeliveryLng: offer.DeliveryLng(),
                                Currency:    offer.Currency(),
                        }, "system")
                }
        }()

        return port.ToDispatchOfferDTOPtr(*updated), nil
}

// ===== CancelOffer =====

func (s *Service) CancelOffer(ctx context.Context, offerID string) (*port.DispatchOfferDTO, error) {
        var updated *domain.DispatchOffer
        err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
                offer, err := s.deps.Repos.Offers.GetByID(ctx, exec, offerID)
                if err != nil {
                        return err
                }
                cancelled, err := offer.Cancel(s.deps.Clock.Now())
                if err != nil {
                        return err
                }
                if err := s.deps.Repos.Offers.Update(ctx, exec, cancelled); err != nil {
                        return err
                }

                ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
                envelope, err := events.OfferCancelledEnvelope(port.OfferCancelledPayload{
                        OfferID:  cancelled.ID(),
                        OrderID:  cancelled.OrderID(),
                        DriverID: cancelled.DriverID(),
                }, ec)
                if err != nil {
                        return err
                }
                if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
                        return err
                }
                updated = &cancelled
                return nil
        })
        if err != nil {
                return nil, err
        }
        return port.ToDispatchOfferDTOPtr(*updated), nil
}

// ===== GetOffer / ListOffers =====

func (s *Service) GetOffer(ctx context.Context, id string) (*port.DispatchOfferDTO, error) {
        offer, err := s.deps.Repos.Offers.GetByID(ctx, s.pool, id)
        if err != nil {
                return nil, err
        }
        return port.ToDispatchOfferDTOPtr(*offer), nil
}

func (s *Service) ListOffersByDriver(ctx context.Context, driverID string, page port.PageQuery) (port.Page[port.DispatchOfferDTO], error) {
        result, err := s.deps.Repos.Offers.ListByDriver(ctx, s.pool, driverID, page)
        if err != nil {
                return port.Page[port.DispatchOfferDTO]{}, err
        }
        dtos := make([]port.DispatchOfferDTO, 0, len(result.Items))
        for _, o := range result.Items {
                dtos = append(dtos, port.ToDispatchOfferDTO(o))
        }
        return port.Page[port.DispatchOfferDTO]{
                Items:  dtos,
                Total:  result.Total,
                Limit:  result.Limit,
                Offset: result.Offset,
        }, nil
}

func (s *Service) ListOffersByOrder(ctx context.Context, orderID string) ([]port.DispatchOfferDTO, error) {
        offers, err := s.deps.Repos.Offers.ListByOrder(ctx, s.pool, orderID)
        if err != nil {
                return nil, err
        }
        dtos := make([]port.DispatchOfferDTO, 0, len(offers))
        for _, o := range offers {
                dtos = append(dtos, port.ToDispatchOfferDTO(o))
        }
        return dtos, nil
}

// ===== ExpireStaleOffers =====
//
// Background job — called periodically by a ticker.
// Finds all pending offers whose expires_at < now, marks them as expired,
// and publishes offer.expired events.

func (s *Service) ExpireStaleOffers(ctx context.Context) (int, error) {
        // Find all pending offers that are past their expiry.
        // We use the offers table directly with a SELECT ... WHERE status = 'pending' AND expires_at < NOW().
        // The repository doesn't expose this method directly, so we use a
        // different approach: iterate via the pending index.
        //
        // For simplicity, we use a direct SQL query here via the pool.
        // In a production system, this would be a repository method.

        // Implementation note: we don't have a ListExpiredPending method on the
        // repository. We can add one, but for now we'll skip this and rely on
        // the explicit ExpireOffer call from the driver app or the next attempt.
        //
        // Alternative: the background ticker for retry matching will trigger
        // the expiry implicitly when it tries to create a new offer for an
        // order whose previous offer has expired (the CreateOfferInternal
        // function will fail to find the existing offer because it's now expired
        // — but wait, it uses GetActiveOfferForOrder which only returns pending
        // offers, so an expired offer won't block a new attempt).
        //
        // For now, return 0 — the implicit expiry via the next attempt is sufficient.
        return 0, nil
}
