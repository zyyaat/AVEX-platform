// Package domain restaurant: Restaurant aggregate root.
package domain

import (
	"fmt"
	"strings"
	"time"
)

type Restaurant struct {
	id              string
	name            string
	nameAr          string
	description     string
	descriptionAr   string
	imageURL        string
	coverURL        string
	cuisines        string
	lat             float64
	lng             float64
	zoneID          string // soft ref → financial.delivery_zones
	merchantID      string // soft ref → identity.merchants
	rating          float64
	ratingCount     int
	deliveryTimeMin int
	deliveryTimeMax int
	deliveryFee     float64
	minOrder        float64
	isActive        bool
	isPro           bool
	createdAt       time.Time
	updatedAt       time.Time
}

type RestaurantParams struct {
	ID, Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines string
	Lat, Lng                                                                   float64
	ZoneID, MerchantID                                                         string
	Rating                                                                     float64
	RatingCount, DeliveryTimeMin, DeliveryTimeMax                              int
	DeliveryFee, MinOrder                                                      float64
	IsActive, IsPro                                                            bool
	Now                                                                        time.Time
}

func NewRestaurant(p RestaurantParams) (Restaurant, error) {
	if p.ID == "" {
		return Restaurant{}, NewValidationError("id", ErrInvalidID)
	}
	if strings.TrimSpace(p.Name) == "" {
		return Restaurant{}, NewValidationError("name", ErrNameRequired)
	}
	if p.Lat < -90 || p.Lat > 90 {
		return Restaurant{}, NewValidationError("lat", ErrInvalidLatitude)
	}
	if p.Lng < -180 || p.Lng > 180 {
		return Restaurant{}, NewValidationError("lng", ErrInvalidLongitude)
	}

	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return Restaurant{
		id: p.ID, name: strings.TrimSpace(p.Name), nameAr: p.NameAr,
		description: p.Description, descriptionAr: p.DescriptionAr,
		imageURL: p.ImageURL, coverURL: p.CoverURL, cuisines: p.Cuisines,
		lat: p.Lat, lng: p.Lng, zoneID: p.ZoneID, merchantID: p.MerchantID,
		rating: p.Rating, ratingCount: p.RatingCount,
		deliveryTimeMin: p.DeliveryTimeMin, deliveryTimeMax: p.DeliveryTimeMax,
		deliveryFee: p.DeliveryFee, minOrder: p.MinOrder,
		isActive: p.IsActive, isPro: p.IsPro,
		createdAt: now, updatedAt: now,
	}, nil
}

type RestaurantRecord struct {
	ID, Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines string
	Lat, Lng                                                                   float64
	ZoneID, MerchantID                                                         string
	Rating                                                                     float64
	RatingCount, DeliveryTimeMin, DeliveryTimeMax                              int
	DeliveryFee, MinOrder                                                      float64
	IsActive, IsPro                                                            bool
	CreatedAt, UpdatedAt                                                       time.Time
}

func ReconstructRestaurant(r RestaurantRecord) Restaurant {
	return Restaurant{
		id: r.ID, name: r.Name, nameAr: r.NameAr, description: r.Description, descriptionAr: r.DescriptionAr,
		imageURL: r.ImageURL, coverURL: r.CoverURL, cuisines: r.Cuisines,
		lat: r.Lat, lng: r.Lng, zoneID: r.ZoneID, merchantID: r.MerchantID,
		rating: r.Rating, ratingCount: r.RatingCount,
		deliveryTimeMin: r.DeliveryTimeMin, deliveryTimeMax: r.DeliveryTimeMax,
		deliveryFee: r.DeliveryFee, minOrder: r.MinOrder,
		isActive: r.IsActive, isPro: r.IsPro, createdAt: r.CreatedAt, updatedAt: r.UpdatedAt,
	}
}

func (r Restaurant) ID() string            { return r.id }
func (r Restaurant) Name() string          { return r.name }
func (r Restaurant) NameAr() string        { return r.nameAr }
func (r Restaurant) Description() string   { return r.description }
func (r Restaurant) DescriptionAr() string { return r.descriptionAr }
func (r Restaurant) ImageURL() string      { return r.imageURL }
func (r Restaurant) CoverURL() string      { return r.coverURL }
func (r Restaurant) Cuisines() string      { return r.cuisines }
func (r Restaurant) Lat() float64          { return r.lat }
func (r Restaurant) Lng() float64          { return r.lng }
func (r Restaurant) ZoneID() string        { return r.zoneID }
func (r Restaurant) MerchantID() string    { return r.merchantID }
func (r Restaurant) Rating() float64       { return r.rating }
func (r Restaurant) RatingCount() int      { return r.ratingCount }
func (r Restaurant) DeliveryTimeMin() int  { return r.deliveryTimeMin }
func (r Restaurant) DeliveryTimeMax() int  { return r.deliveryTimeMax }
func (r Restaurant) DeliveryFee() float64  { return r.deliveryFee }
func (r Restaurant) MinOrder() float64     { return r.minOrder }
func (r Restaurant) IsActive() bool        { return r.isActive }
func (r Restaurant) IsPro() bool           { return r.isPro }
func (r Restaurant) CreatedAt() time.Time  { return r.createdAt }
func (r Restaurant) UpdatedAt() time.Time  { return r.updatedAt }

func (r *Restaurant) Activate(now time.Time)   { r.isActive = true; r.updatedAt = now }
func (r *Restaurant) Deactivate(now time.Time) { r.isActive = false; r.updatedAt = now }
func (r *Restaurant) UpdateRating(rating float64, count int, now time.Time) {
	r.rating = rating
	r.ratingCount = count
	r.updatedAt = now
}

func (r Restaurant) String() string {
	return fmt.Sprintf("Restaurant{id=%s, name=%s, active=%v, zone=%s}", r.id, r.name, r.isActive, r.zoneID)
}
