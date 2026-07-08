// Package port service: ServicePort + DTOs for the dispatch module.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/dispatch/domain"
)

// ===== Driver DTOs =====

type RegisterDriverInput struct {
	UserID       string
	VehicleType  string // bike | scooter | car
	LicensePlate string
	ZoneIDs      []string
}

type DriverDTO struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	VehicleType     string     `json:"vehicle_type"`
	LicensePlate    string     `json:"license_plate"`
	Status          string     `json:"status"`
	Rating          float64    `json:"rating"`
	RatingCount     int        `json:"rating_count"`
	AcceptanceRate  int        `json:"acceptance_rate"`
	CompletionRate  int        `json:"completion_rate"`
	TotalDeliveries int        `json:"total_deliveries"`
	ZoneIDs         []string   `json:"zone_ids,omitempty"`
	CurrentOrderID  string     `json:"current_order_id,omitempty"`
	GoOnlineAt      *time.Time `json:"go_online_at,omitempty"`
	GoOfflineAt     *time.Time `json:"go_offline_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ===== Location DTOs =====

type UpdateLocationInput struct {
	DriverID   string
	Lat        float64
	Lng        float64
	Bearing    float64
	Speed      float64
	Accuracy   float64
	CapturedAt time.Time
}

type LocationDTO struct {
	DriverID   string    `json:"driver_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Bearing    float64   `json:"bearing"`
	Speed      float64   `json:"speed"`
	Accuracy   float64   `json:"accuracy"`
	CapturedAt time.Time `json:"captured_at"`
	ReceivedAt time.Time `json:"received_at"`
}

// ===== Offer DTOs =====

// CreateOfferInput is used when the dispatch service receives an
// OrderAssignmentRequested event from the orders module.
type CreateOfferInput struct {
	OrderID     string
	ZoneID      string
	PickupLat   float64
	PickupLng   float64
	DeliveryLat float64
	DeliveryLng float64
	Currency    string
	// DriverID can be empty — the service will find the nearest available driver.
	// If specified, the offer goes directly to that driver (manual dispatch).
	DriverID string
}

type DispatchOfferDTO struct {
	ID            string     `json:"id"`
	OrderID       string     `json:"order_id"`
	DriverID      string     `json:"driver_id"`
	ZoneID        string     `json:"zone_id,omitempty"`
	Status        string     `json:"status"`
	PickupLat     float64    `json:"pickup_lat"`
	PickupLng     float64    `json:"pickup_lng"`
	DeliveryLat   float64    `json:"delivery_lat"`
	DeliveryLng   float64    `json:"delivery_lng"`
	EstDistanceM  *int       `json:"est_distance_m,omitempty"`
	EstDurationS  *int       `json:"est_duration_s,omitempty"`
	EstFareCents  *int64     `json:"est_fare_cents,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	OfferTTL      string     `json:"offer_ttl"` // duration string
	OfferedAt     time.Time  `json:"offered_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty"`
	RejectedAt    *time.Time `json:"rejected_at,omitempty"`
	ExpiredAt     *time.Time `json:"expired_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	RejectReason  string     `json:"reject_reason,omitempty"`
	AttemptNumber int        `json:"attempt_number"`
	CreatedBy     string     `json:"created_by"`
}

// ===== ServicePort =====

type ServicePort interface {
	// ===== Driver Operations =====
	RegisterDriver(ctx context.Context, input RegisterDriverInput) (*DriverDTO, error)
	GetDriver(ctx context.Context, id string) (*DriverDTO, error)
	GetDriverByUserID(ctx context.Context, userID string) (*DriverDTO, error)
	GoOnline(ctx context.Context, driverID string) (*DriverDTO, error)
	GoOffline(ctx context.Context, driverID string) (*DriverDTO, error)
	SuspendDriver(ctx context.Context, id, reason string) (*DriverDTO, error)
	UnsuspendDriver(ctx context.Context, id string) (*DriverDTO, error)
	ListDrivers(ctx context.Context, page PageQuery) (Page[DriverDTO], error)
	ListOnlineDrivers(ctx context.Context, zoneID string) ([]DriverDTO, error)

	// ===== Location Operations =====
	UpdateLocation(ctx context.Context, input UpdateLocationInput) (*LocationDTO, error)
	GetLocation(ctx context.Context, driverID string) (*LocationDTO, error)
	FindNearestDrivers(ctx context.Context, lat, lng float64, radiusM int, limit int) ([]NearbyDriver, error)

	// ===== Dispatch Offer Operations =====
	// CreateOffer is called when an OrderAssignmentRequested event arrives.
	// It finds the nearest available driver, creates an offer, and publishes
	// offer.created.
	CreateOffer(ctx context.Context, input CreateOfferInput) (*DispatchOfferDTO, error)
	AcceptOffer(ctx context.Context, offerID, driverID string) (*DispatchOfferDTO, error)
	RejectOffer(ctx context.Context, offerID, driverID, reason string) (*DispatchOfferDTO, error)
	ExpireOffer(ctx context.Context, offerID string) (*DispatchOfferDTO, error)
	CancelOffer(ctx context.Context, offerID string) (*DispatchOfferDTO, error)
	GetOffer(ctx context.Context, id string) (*DispatchOfferDTO, error)
	ListOffersByDriver(ctx context.Context, driverID string, page PageQuery) (Page[DispatchOfferDTO], error)
	ListOffersByOrder(ctx context.Context, orderID string) ([]DispatchOfferDTO, error)

	// ===== Driver Order Lifecycle =====
	// Called by the orders module when an order is picked up / delivered.
	// These methods transition the driver state accordingly.
	DriverOrderCompleted(ctx context.Context, driverID, orderID string) error

	// ===== Event Handler (called by the bus subscriber) =====
	// HandleOrderAssignmentRequested is the consumer of
	// orders.order.assignment_requested. It finds the nearest driver and
	// creates an offer. Idempotent — uses the inbox pattern.
	HandleOrderAssignmentRequested(ctx context.Context, orderID, zoneID string, pickupLat, pickupLng, deliveryLat, deliveryLng float64) error

	// ===== Background Job (called by a ticker) =====
	// ExpireStaleOffers marks all expired offers as expired and publishes
	// offer.expired events. Returns the number of offers expired.
	ExpireStaleOffers(ctx context.Context) (int, error)
}

// ===== Domain → DTO Mappers =====

func ToDriverDTO(d domain.Driver) DriverDTO {
	return DriverDTO{
		ID:              d.ID(),
		UserID:          d.UserID(),
		VehicleType:     string(d.VehicleType()),
		LicensePlate:    d.LicensePlate(),
		Status:          string(d.Status()),
		Rating:          d.Rating(),
		RatingCount:     d.RatingCount(),
		AcceptanceRate:  d.AcceptanceRate(),
		CompletionRate:  d.CompletionRate(),
		TotalDeliveries: d.TotalDeliveries(),
		ZoneIDs:         d.ZoneIDs(),
		CurrentOrderID:  d.CurrentOrderID(),
		GoOnlineAt:      d.GoOnlineAt(),
		GoOfflineAt:     d.GoOfflineAt(),
		CreatedAt:       d.CreatedAt(),
		UpdatedAt:       d.UpdatedAt(),
	}
}

func ToLocationDTO(loc domain.DriverLocation) LocationDTO {
	return LocationDTO{
		DriverID:   loc.DriverID(),
		Lat:        loc.Lat(),
		Lng:        loc.Lng(),
		Bearing:    loc.Bearing(),
		Speed:      loc.Speed(),
		Accuracy:   loc.Accuracy(),
		CapturedAt: loc.CapturedAt(),
		ReceivedAt: loc.ReceivedAt(),
	}
}

func ToDispatchOfferDTO(o domain.DispatchOffer) DispatchOfferDTO {
	return DispatchOfferDTO{
		ID:            o.ID(),
		OrderID:       o.OrderID(),
		DriverID:      o.DriverID(),
		ZoneID:        o.ZoneID(),
		Status:        string(o.Status()),
		PickupLat:     o.PickupLat(),
		PickupLng:     o.PickupLng(),
		DeliveryLat:   o.DeliveryLat(),
		DeliveryLng:   o.DeliveryLng(),
		EstDistanceM:  o.EstDistanceM(),
		EstDurationS:  o.EstDurationS(),
		EstFareCents:  o.EstFareCents(),
		Currency:      o.Currency(),
		OfferTTL:      o.OfferTTL().String(),
		OfferedAt:     o.OfferedAt(),
		ExpiresAt:     o.ExpiresAt(),
		RespondedAt:   o.RespondedAt(),
		AcceptedAt:    o.AcceptedAt(),
		RejectedAt:    o.RejectedAt(),
		ExpiredAt:     o.ExpiredAt(),
		CancelledAt:   o.CancelledAt(),
		RejectReason:  o.RejectReason(),
		AttemptNumber: o.AttemptNumber(),
		CreatedBy:     o.CreatedBy(),
	}
}

// ===== Pointer Helpers =====

func ToDriverDTOPtr(d domain.Driver) *DriverDTO {
	dto := ToDriverDTO(d)
	return &dto
}

func ToLocationDTOPtr(l domain.DriverLocation) *LocationDTO {
	dto := ToLocationDTO(l)
	return &dto
}

func ToDispatchOfferDTOPtr(o domain.DispatchOffer) *DispatchOfferDTO {
	dto := ToDispatchOfferDTO(o)
	return &dto
}
