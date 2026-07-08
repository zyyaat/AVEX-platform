// Package domain dispatch_info: DispatchInfo value object.
//
// Encapsulates all dispatch-related metadata for an order.
// This value object separates dispatch concerns from the Order entity,
// preventing Order from becoming a God Object.
//
// In the future, this can be extracted to a separate DispatchAssignment
// entity in the Dispatch module without breaking the Order entity.
//
// Imports stdlib only.
package domain

// DispatchInfo holds dispatch-related metadata for an order.
// All fields are optional except zoneID (set at order creation).
type DispatchInfo struct {
	driverID         *string // nil until a driver is assigned
	zoneID           string  // soft ref → financial.delivery_zones.id
	dispatchDistance *int    // meters: driver → restaurant (set when driver accepts)
	deliveryDistance *int    // meters: restaurant → customer (set at order creation)
	pickupPhotoURL   *string // photo proof of pickup
	deliveryPhotoURL *string // photo proof of delivery
}

// NewDispatchInfo creates a DispatchInfo with the given zone and delivery distance.
// driverID, dispatchDistance, and photos start as nil (set later in the lifecycle).
func NewDispatchInfo(zoneID string, deliveryDistance *int) DispatchInfo {
	return DispatchInfo{
		zoneID:           zoneID,
		deliveryDistance: deliveryDistance,
	}
}

// DriverID returns the assigned driver ID, or empty string if none assigned.
func (d DispatchInfo) DriverID() string {
	if d.driverID == nil {
		return ""
	}
	return *d.driverID
}

// HasDriver reports whether a driver has been assigned.
func (d DispatchInfo) HasDriver() bool {
	return d.driverID != nil && *d.driverID != ""
}

// ZoneID returns the delivery zone ID.
func (d DispatchInfo) ZoneID() string {
	return d.zoneID
}

// DispatchDistance returns the dispatch distance in meters, or 0 if not set.
func (d DispatchInfo) DispatchDistance() int {
	if d.dispatchDistance == nil {
		return 0
	}
	return *d.dispatchDistance
}

// DeliveryDistance returns the delivery distance in meters, or 0 if not set.
func (d DispatchInfo) DeliveryDistance() int {
	if d.deliveryDistance == nil {
		return 0
	}
	return *d.deliveryDistance
}

// PickupPhotoURL returns the pickup photo URL, or empty string if not set.
func (d DispatchInfo) PickupPhotoURL() string {
	if d.pickupPhotoURL == nil {
		return ""
	}
	return *d.pickupPhotoURL
}

// DeliveryPhotoURL returns the delivery photo URL, or empty string if not set.
func (d DispatchInfo) DeliveryPhotoURL() string {
	if d.deliveryPhotoURL == nil {
		return ""
	}
	return *d.deliveryPhotoURL
}

// IsZero reports whether the dispatch info is completely unset.
func (d DispatchInfo) IsZero() bool {
	return d.driverID == nil && d.zoneID == "" && d.dispatchDistance == nil &&
		d.deliveryDistance == nil && d.pickupPhotoURL == nil && d.deliveryPhotoURL == nil
}
