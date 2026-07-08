// Package port service: ServicePort + DTOs.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/catalog/domain"
)

// ===== Input DTOs =====

type CreateRestaurantInput struct {
	Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines string
	Lat, Lng                                                               float64
	ZoneID, MerchantID                                                     string
	DeliveryTimeMin, DeliveryTimeMax                                       int
	DeliveryFee, MinOrder                                                  float64
	IsPro                                                                  bool
}

type UpdateRestaurantInput struct {
	Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines string
	DeliveryTimeMin, DeliveryTimeMax                                       int
	DeliveryFee, MinOrder                                                  float64
}

type CreateMenuItemInput struct {
	RestaurantID, CategoryID, Name, NameAr, Description, DescriptionAr, Image, ImageURL string
	Price                                                                               float64
	IsPopular, IsAvailable                                                              bool
	PrepTime, Calories                                                                  int
}

type UpdateMenuItemInput struct {
	Name, NameAr, Description, DescriptionAr, Image, ImageURL string
	Price                                                     float64
	IsPopular, IsAvailable                                    bool
	PrepTime, Calories                                        int
}

type CreateCategoryInput struct {
	Name, NameAr, Icon, ImageURL string
	SortOrder                    int
}

// ===== Output DTOs =====

type RestaurantDTO struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	NameAr          string    `json:"name_ar,omitempty"`
	Description     string    `json:"description,omitempty"`
	DescriptionAr   string    `json:"description_ar,omitempty"`
	ImageURL        string    `json:"image_url,omitempty"`
	CoverURL        string    `json:"cover_url,omitempty"`
	Cuisines        string    `json:"cuisines,omitempty"`
	Lat             float64   `json:"lat"`
	Lng             float64   `json:"lng"`
	ZoneID          string    `json:"zone_id,omitempty"`
	Rating          float64   `json:"rating"`
	RatingCount     int       `json:"rating_count"`
	DeliveryTimeMin int       `json:"delivery_time_min"`
	DeliveryTimeMax int       `json:"delivery_time_max"`
	DeliveryFee     float64   `json:"delivery_fee"`
	MinOrder        float64   `json:"min_order"`
	IsActive        bool      `json:"is_active"`
	IsPro           bool      `json:"is_pro"`
	CreatedAt       time.Time `json:"created_at"`
}

type MenuItemDTO struct {
	ID            string  `json:"id"`
	RestaurantID  string  `json:"restaurant_id"`
	CategoryID    string  `json:"category_id,omitempty"`
	Name          string  `json:"name"`
	NameAr        string  `json:"name_ar,omitempty"`
	Description   string  `json:"description,omitempty"`
	DescriptionAr string  `json:"description_ar,omitempty"`
	Price         float64 `json:"price"`
	ImageURL      string  `json:"image_url,omitempty"`
	IsPopular     bool    `json:"is_popular"`
	IsAvailable   bool    `json:"is_available"`
	Rating        float64 `json:"rating"`
	PrepTime      int     `json:"prep_time"`
	Calories      int     `json:"calories"`
}

type CategoryDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NameAr    string `json:"name_ar,omitempty"`
	Icon      string `json:"icon"`
	ImageURL  string `json:"image_url,omitempty"`
	SortOrder int    `json:"sort_order"`
}

// ===== ServicePort =====

type ServicePort interface {
	// Restaurant
	CreateRestaurant(ctx context.Context, input CreateRestaurantInput) (*RestaurantDTO, error)
	GetRestaurant(ctx context.Context, id string) (*RestaurantDTO, error)
	UpdateRestaurant(ctx context.Context, id string, input UpdateRestaurantInput) (*RestaurantDTO, error)
	ActivateRestaurant(ctx context.Context, id string) error
	DeactivateRestaurant(ctx context.Context, id string) error
	ListRestaurants(ctx context.Context, activeOnly bool, page PageQuery) (Page[RestaurantDTO], error)

	// Menu Items
	CreateMenuItem(ctx context.Context, input CreateMenuItemInput) (*MenuItemDTO, error)
	GetMenuItem(ctx context.Context, id string) (*MenuItemDTO, error)
	UpdateMenuItem(ctx context.Context, id string, input UpdateMenuItemInput) (*MenuItemDTO, error)
	DeleteMenuItem(ctx context.Context, id string) error
	ListMenuItems(ctx context.Context, restaurantID string) ([]MenuItemDTO, error)
	ListPopularItems(ctx context.Context, limit int) ([]MenuItemDTO, error)

	// Categories
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*CategoryDTO, error)
	ListCategories(ctx context.Context) ([]CategoryDTO, error)

	// Menu (grouped by category)
	GetMenu(ctx context.Context, restaurantID string) (map[string][]MenuItemDTO, error)
}

// Helper: domain → DTO
func ToRestaurantDTO(r domain.Restaurant) RestaurantDTO {
	return RestaurantDTO{
		ID: r.ID(), Name: r.Name(), NameAr: r.NameAr(),
		Description: r.Description(), DescriptionAr: r.DescriptionAr(),
		ImageURL: r.ImageURL(), CoverURL: r.CoverURL(), Cuisines: r.Cuisines(),
		Lat: r.Lat(), Lng: r.Lng(), ZoneID: r.ZoneID(),
		Rating: r.Rating(), RatingCount: r.RatingCount(),
		DeliveryTimeMin: r.DeliveryTimeMin(), DeliveryTimeMax: r.DeliveryTimeMax(),
		DeliveryFee: r.DeliveryFee(), MinOrder: r.MinOrder(),
		IsActive: r.IsActive(), IsPro: r.IsPro(), CreatedAt: r.CreatedAt(),
	}
}

func ToMenuItemDTO(m domain.MenuItem) MenuItemDTO {
	return MenuItemDTO{
		ID: m.ID(), RestaurantID: m.RestaurantID(), CategoryID: m.CategoryID(),
		Name: m.Name(), NameAr: m.NameAr(), Description: m.Description(), DescriptionAr: m.DescriptionAr(),
		Price: m.Price(), ImageURL: m.ImageURL(), IsPopular: m.IsPopular(), IsAvailable: m.IsAvailable(),
		Rating: m.Rating(), PrepTime: m.PrepTime(), Calories: m.Calories(),
	}
}

func ToCategoryDTO(c domain.Category) CategoryDTO {
	return CategoryDTO{ID: c.ID(), Name: c.Name(), NameAr: c.NameAr(), Icon: c.Icon(), ImageURL: c.ImageURL(), SortOrder: c.SortOrder()}
}
