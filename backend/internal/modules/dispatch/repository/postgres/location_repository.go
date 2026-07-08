// Package postgres location_repository: DriverLocationRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/dispatch/domain"
	"avex-backend/internal/modules/dispatch/port"
)

// DriverLocationRepository implements port.DriverLocationRepository.
type DriverLocationRepository struct{}

var _ port.DriverLocationRepository = (*DriverLocationRepository)(nil)

// Upsert inserts or updates the current location for a driver.
func (r *DriverLocationRepository) Upsert(ctx context.Context, exec port.Executor, loc domain.DriverLocation) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO dispatch.driver_locations (
			driver_id, lat, lng, bearing, speed, accuracy, captured_at, received_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (driver_id) DO UPDATE SET
			lat = EXCLUDED.lat,
			lng = EXCLUDED.lng,
			bearing = EXCLUDED.bearing,
			speed = EXCLUDED.speed,
			accuracy = EXCLUDED.accuracy,
			captured_at = EXCLUDED.captured_at,
			received_at = EXCLUDED.received_at
	`,
		loc.DriverID(),
		loc.Lat(),
		loc.Lng(),
		loc.Bearing(),
		loc.Speed(),
		loc.Accuracy(),
		loc.CapturedAt(),
		loc.ReceivedAt(),
	)
	if err != nil {
		return fmt.Errorf("upsert location: %w", err)
	}
	return nil
}

// GetByDriver retrieves the current location for a driver.
func (r *DriverLocationRepository) GetByDriver(ctx context.Context, exec port.Executor, driverID string) (*domain.DriverLocation, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `
		SELECT driver_id, lat, lng, bearing, speed, accuracy, captured_at, received_at
		FROM dispatch.driver_locations
		WHERE driver_id = $1
	`, driverID)
	loc, err := scanLocation(row)
	if err != nil {
		return nil, mapLocationReadError(err)
	}
	return &loc, nil
}

// FindNearestDrivers returns up to `limit` drivers within `radiusM` meters
// of the given (lat, lng), ordered by distance ascending.
//
// Implementation: bounding-box filter in SQL + Haversine re-ranking in Go.
// For production scale, this should use PostGIS KNN (`<-|->` operator on gist index).
// The bounding-box approach is a good approximation for ~10K drivers.
func (r *DriverLocationRepository) FindNearestDrivers(ctx context.Context, exec port.Executor, lat, lng float64, radiusM int, maxAge time.Duration, limit int) ([]port.NearbyDriver, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: %d", domain.ErrInvalidLimit, limit)
	}
	if radiusM <= 0 {
		return nil, fmt.Errorf("%w: %d", domain.ErrInvalidRadius, radiusM)
	}

	dbtx := toDBTX(exec)

	// Convert radius (meters) to degree offsets.
	// 1 degree of latitude ≈ 111,000 meters.
	// 1 degree of longitude ≈ 111,000 * cos(lat) meters.
	const metersPerDegLat = 111000.0
	degLat := float64(radiusM) / metersPerDegLat
	degLng := float64(radiusM) / (metersPerDegLat * cosDeg(lat))

	minLat := lat - degLat
	maxLat := lat + degLat
	minLng := lng - degLng
	maxLng := lng + degLng

	cutoff := time.Now().UTC().Add(-maxAge)
	rows, err := dbtx.Query(ctx, `
		SELECT
			dl.driver_id, dl.lat, dl.lng, dl.bearing, dl.speed, dl.captured_at
		FROM dispatch.driver_locations dl
		JOIN dispatch.drivers d ON d.id = dl.driver_id
		WHERE d.status = 'online'
		  AND dl.captured_at >= $1
		  AND dl.lat BETWEEN $2 AND $3
		  AND dl.lng BETWEEN $4 AND $5
	`, cutoff, minLat, maxLat, minLng, maxLng)
	if err != nil {
		return nil, fmt.Errorf("find nearest: %w", err)
	}
	defer rows.Close()

	type rawLoc struct {
		driverID  string
		lat, lng  float64
		bearing   float64
		speed     float64
		captured  time.Time
	}
	var candidates []rawLoc
	for rows.Next() {
		var rl rawLoc
		if err := rows.Scan(&rl.driverID, &rl.lat, &rl.lng, &rl.bearing, &rl.speed, &rl.captured); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		candidates = append(candidates, rl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	// Haversine re-ranking + filter
	nearby := make([]port.NearbyDriver, 0, len(candidates))
	for _, c := range candidates {
		dist := haversineMeters(c.lat, c.lng, lat, lng)
		if int(dist) > radiusM {
			continue // outside the radius (bounding box was a square)
		}
		nearby = append(nearby, port.NearbyDriver{
			DriverID:    c.driverID,
			Lat:         c.lat,
			Lng:         c.lng,
			DistanceM:   int(dist),
			Bearing:     c.bearing,
			Speed:       c.speed,
			LocationAge: time.Since(c.captured),
			CapturedAt:  c.captured,
		})
	}

	// Sort by distance ascending (insertion sort — small N)
	for i := 1; i < len(nearby); i++ {
		for j := i; j > 0 && nearby[j].DistanceM < nearby[j-1].DistanceM; j-- {
			nearby[j], nearby[j-1] = nearby[j-1], nearby[j]
		}
	}

	// Apply limit
	if len(nearby) > limit {
		nearby = nearby[:limit]
	}

	return nearby, nil
}

// DeleteByDriver removes the location for a driver.
func (r *DriverLocationRepository) DeleteByDriver(ctx context.Context, exec port.Executor, driverID string) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `DELETE FROM dispatch.driver_locations WHERE driver_id = $1`, driverID)
	if err != nil {
		return fmt.Errorf("delete location: %w", err)
	}
	return nil
}

// ===== Helpers =====

func scanLocation(s scanner) (domain.DriverLocation, error) {
	var (
		driverID                                     string
		lat, lng, bearing, speed, accuracy           float64
		capturedAt, receivedAt                       time.Time
	)
	if err := s.Scan(&driverID, &lat, &lng, &bearing, &speed, &accuracy, &capturedAt, &receivedAt); err != nil {
		return domain.DriverLocation{}, err
	}
	// Reconstruct without an ID field — driver_locations table uses driver_id as PK.
	return domain.RehydrateDriverLocation("", driverID, lat, lng, bearing, speed, accuracy, capturedAt, receivedAt), nil
}

func mapLocationReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrLocationNotFound
	}
	return fmt.Errorf("location read: %w", err)
}

// cosDeg returns cos of degrees (degrees → radians → cos).
func cosDeg(deg float64) float64 {
	rad := deg * 3.14159265358979323846 / 180.0
	return mathCos(rad)
}
