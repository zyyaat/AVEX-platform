// Package domain menu_item: MenuItem entity — snapshot of a dish.
package domain

import (
	"fmt"
	"strings"
	"time"
)

type MenuItem struct {
	id            string
	restaurantID  string
	categoryID    string
	name          string
	nameAr        string
	description   string
	descriptionAr string
	price         float64
	image         string
	imageURL      string
	isPopular     bool
	isAvailable   bool
	rating        float64
	ratingCount   int
	prepTime      int
	calories      int
	createdAt     time.Time
	updatedAt     time.Time
}

type MenuItemParams struct {
	ID, RestaurantID, CategoryID, Name, NameAr, Description, DescriptionAr, Image, ImageURL string
	Price                                                                                   float64
	IsPopular, IsAvailable                                                                  bool
	Rating                                                                                  float64
	RatingCount, PrepTime, Calories                                                         int
	Now                                                                                     time.Time
}

func NewMenuItem(p MenuItemParams) (MenuItem, error) {
	if p.ID == "" {
		return MenuItem{}, NewValidationError("id", ErrInvalidID)
	}
	if p.RestaurantID == "" {
		return MenuItem{}, NewValidationError("restaurant_id", ErrInvalidInput)
	}
	if strings.TrimSpace(p.Name) == "" {
		return MenuItem{}, NewValidationError("name", ErrNameRequired)
	}
	if p.Price < 0 {
		return MenuItem{}, NewValidationError("price", ErrInvalidPrice)
	}

	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	return MenuItem{
		id: p.ID, restaurantID: p.RestaurantID, categoryID: p.CategoryID,
		name: strings.TrimSpace(p.Name), nameAr: p.NameAr,
		description: p.Description, descriptionAr: p.DescriptionAr,
		price: p.Price, image: p.Image, imageURL: p.ImageURL,
		isPopular: p.IsPopular, isAvailable: p.IsAvailable,
		rating: p.Rating, ratingCount: p.RatingCount,
		prepTime: p.PrepTime, calories: p.Calories,
		createdAt: now, updatedAt: now,
	}, nil
}

type MenuItemRecord struct {
	ID, RestaurantID, CategoryID, Name, NameAr, Description, DescriptionAr, Image, ImageURL string
	Price                                                                                   float64
	IsPopular, IsAvailable                                                                  bool
	Rating                                                                                  float64
	RatingCount, PrepTime, Calories                                                         int
	CreatedAt, UpdatedAt                                                                    time.Time
}

func ReconstructMenuItem(r MenuItemRecord) MenuItem {
	return MenuItem{
		id: r.ID, restaurantID: r.RestaurantID, categoryID: r.CategoryID,
		name: r.Name, nameAr: r.NameAr, description: r.Description, descriptionAr: r.DescriptionAr,
		price: r.Price, image: r.Image, imageURL: r.ImageURL,
		isPopular: r.IsPopular, isAvailable: r.IsAvailable,
		rating: r.Rating, ratingCount: r.RatingCount,
		prepTime: r.PrepTime, calories: r.Calories,
		createdAt: r.CreatedAt, updatedAt: r.UpdatedAt,
	}
}

func (m MenuItem) ID() string            { return m.id }
func (m MenuItem) RestaurantID() string  { return m.restaurantID }
func (m MenuItem) CategoryID() string    { return m.categoryID }
func (m MenuItem) Name() string          { return m.name }
func (m MenuItem) NameAr() string        { return m.nameAr }
func (m MenuItem) Description() string   { return m.description }
func (m MenuItem) DescriptionAr() string { return m.descriptionAr }
func (m MenuItem) Price() float64        { return m.price }
func (m MenuItem) Image() string         { return m.image }
func (m MenuItem) ImageURL() string      { return m.imageURL }
func (m MenuItem) IsPopular() bool       { return m.isPopular }
func (m MenuItem) IsAvailable() bool     { return m.isAvailable }
func (m MenuItem) Rating() float64       { return m.rating }
func (m MenuItem) RatingCount() int      { return m.ratingCount }
func (m MenuItem) PrepTime() int         { return m.prepTime }
func (m MenuItem) Calories() int         { return m.calories }
func (m MenuItem) CreatedAt() time.Time  { return m.createdAt }
func (m MenuItem) UpdatedAt() time.Time  { return m.updatedAt }

func (m *MenuItem) SetAvailable(available bool, now time.Time) {
	m.isAvailable = available
	m.updatedAt = now
}
func (m *MenuItem) UpdatePrice(price float64, now time.Time) error {
	if price < 0 {
		return NewValidationError("price", ErrInvalidPrice)
	}
	m.price = price
	m.updatedAt = now
	return nil
}

func (m MenuItem) String() string {
	return fmt.Sprintf("MenuItem{id=%s, name=%s, price=%.2f, available=%v}", m.id, m.name, m.price, m.isAvailable)
}
