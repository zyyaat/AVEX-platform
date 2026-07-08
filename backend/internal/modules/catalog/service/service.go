// Package service: catalog service layer.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/catalog/domain"
	"avex-backend/internal/modules/catalog/port"
)

type Service struct {
	deps port.Deps
	pool *pgxpool.Pool
}

func New(deps port.Deps, pool *pgxpool.Pool) *Service {
	return &Service{deps: deps, pool: pool}
}

var _ port.ServicePort = (*Service)(nil)

// ===== Restaurant =====

func (s *Service) CreateRestaurant(ctx context.Context, input port.CreateRestaurantInput) (*port.RestaurantDTO, error) {
	id := s.deps.IDGenerator.NewID()
	now := s.deps.Clock.Now()
	rest, err := domain.NewRestaurant(domain.RestaurantParams{
		ID: id, Name: input.Name, NameAr: input.NameAr, Description: input.Description, DescriptionAr: input.DescriptionAr,
		ImageURL: input.ImageURL, CoverURL: input.CoverURL, Cuisines: input.Cuisines,
		Lat: input.Lat, Lng: input.Lng, ZoneID: input.ZoneID, MerchantID: input.MerchantID,
		DeliveryTimeMin: input.DeliveryTimeMin, DeliveryTimeMax: input.DeliveryTimeMax,
		DeliveryFee: input.DeliveryFee, MinOrder: input.MinOrder,
		IsActive: true, IsPro: input.IsPro, Now: now,
	})
	if err != nil {
		return nil, err
	}
	if err := s.deps.Repos.Restaurants.Create(ctx, s.pool, rest); err != nil {
		return nil, err
	}
	dto := port.ToRestaurantDTO(rest)
	return &dto, nil
}

func (s *Service) GetRestaurant(ctx context.Context, id string) (*port.RestaurantDTO, error) {
	rest, err := s.deps.Repos.Restaurants.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	dto := port.ToRestaurantDTO(*rest)
	return &dto, nil
}

func (s *Service) UpdateRestaurant(ctx context.Context, id string, input port.UpdateRestaurantInput) (*port.RestaurantDTO, error) {
	rest, err := s.deps.Repos.Restaurants.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	now := s.deps.Clock.Now()
	updated, err := domain.NewRestaurant(domain.RestaurantParams{
		ID: rest.ID(), Name: input.Name, NameAr: input.NameAr, Description: input.Description, DescriptionAr: input.DescriptionAr,
		ImageURL: input.ImageURL, CoverURL: input.CoverURL, Cuisines: input.Cuisines,
		Lat: rest.Lat(), Lng: rest.Lng(), ZoneID: rest.ZoneID(), MerchantID: rest.MerchantID(),
		Rating: rest.Rating(), RatingCount: rest.RatingCount(),
		DeliveryTimeMin: input.DeliveryTimeMin, DeliveryTimeMax: input.DeliveryTimeMax,
		DeliveryFee: input.DeliveryFee, MinOrder: input.MinOrder,
		IsActive: rest.IsActive(), IsPro: rest.IsPro(), Now: now,
	})
	if err != nil {
		return nil, err
	}
	if err := s.deps.Repos.Restaurants.Update(ctx, s.pool, updated); err != nil {
		return nil, err
	}
	dto := port.ToRestaurantDTO(updated)
	return &dto, nil
}

func (s *Service) ActivateRestaurant(ctx context.Context, id string) error {
	rest, err := s.deps.Repos.Restaurants.GetByID(ctx, s.pool, id)
	if err != nil {
		return err
	}
	rest.Activate(s.deps.Clock.Now())
	return s.deps.Repos.Restaurants.Update(ctx, s.pool, *rest)
}

func (s *Service) DeactivateRestaurant(ctx context.Context, id string) error {
	rest, err := s.deps.Repos.Restaurants.GetByID(ctx, s.pool, id)
	if err != nil {
		return err
	}
	rest.Deactivate(s.deps.Clock.Now())
	return s.deps.Repos.Restaurants.Update(ctx, s.pool, *rest)
}

func (s *Service) ListRestaurants(ctx context.Context, activeOnly bool, page port.PageQuery) (port.Page[port.RestaurantDTO], error) {
	restPage, err := s.deps.Repos.Restaurants.List(ctx, s.pool, activeOnly, page)
	if err != nil {
		return port.Page[port.RestaurantDTO]{}, err
	}
	dtos := make([]port.RestaurantDTO, 0, len(restPage.Items))
	for _, r := range restPage.Items {
		dtos = append(dtos, port.ToRestaurantDTO(r))
	}
	return port.Page[port.RestaurantDTO]{Items: dtos, Total: restPage.Total, Limit: restPage.Limit, Offset: restPage.Offset}, nil
}

// ===== Menu Items =====

func (s *Service) CreateMenuItem(ctx context.Context, input port.CreateMenuItemInput) (*port.MenuItemDTO, error) {
	id := s.deps.IDGenerator.NewID()
	now := s.deps.Clock.Now()
	m, err := domain.NewMenuItem(domain.MenuItemParams{
		ID: id, RestaurantID: input.RestaurantID, CategoryID: input.CategoryID,
		Name: input.Name, NameAr: input.NameAr, Description: input.Description, DescriptionAr: input.DescriptionAr,
		Price: input.Price, Image: input.Image, ImageURL: input.ImageURL,
		IsPopular: input.IsPopular, IsAvailable: input.IsAvailable,
		PrepTime: input.PrepTime, Calories: input.Calories, Now: now,
	})
	if err != nil {
		return nil, err
	}
	if err := s.deps.Repos.MenuItems.Create(ctx, s.pool, m); err != nil {
		return nil, err
	}
	dto := port.ToMenuItemDTO(m)
	return &dto, nil
}

func (s *Service) GetMenuItem(ctx context.Context, id string) (*port.MenuItemDTO, error) {
	m, err := s.deps.Repos.MenuItems.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	dto := port.ToMenuItemDTO(*m)
	return &dto, nil
}

func (s *Service) UpdateMenuItem(ctx context.Context, id string, input port.UpdateMenuItemInput) (*port.MenuItemDTO, error) {
	m, err := s.deps.Repos.MenuItems.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	now := s.deps.Clock.Now()
	if input.Name != "" {
	}
	updated, _ := domain.NewMenuItem(domain.MenuItemParams{
		ID: m.ID(), RestaurantID: m.RestaurantID(), CategoryID: m.CategoryID(),
		Name: input.Name, NameAr: input.NameAr, Description: input.Description, DescriptionAr: input.DescriptionAr,
		Price: input.Price, Image: input.Image, ImageURL: input.ImageURL,
		IsPopular: input.IsPopular, IsAvailable: input.IsAvailable,
		PrepTime: input.PrepTime, Calories: input.Calories, Now: now,
	})
	if err := s.deps.Repos.MenuItems.Update(ctx, s.pool, updated); err != nil {
		return nil, err
	}
	dto := port.ToMenuItemDTO(updated)
	return &dto, nil
}

func (s *Service) DeleteMenuItem(ctx context.Context, id string) error {
	return s.deps.Repos.MenuItems.Delete(ctx, s.pool, id)
}

func (s *Service) ListMenuItems(ctx context.Context, restaurantID string) ([]port.MenuItemDTO, error) {
	items, err := s.deps.Repos.MenuItems.ListByRestaurant(ctx, s.pool, restaurantID)
	if err != nil {
		return nil, err
	}
	dtos := make([]port.MenuItemDTO, 0, len(items))
	for _, m := range items {
		dtos = append(dtos, port.ToMenuItemDTO(m))
	}
	return dtos, nil
}

func (s *Service) ListPopularItems(ctx context.Context, limit int) ([]port.MenuItemDTO, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := s.deps.Repos.MenuItems.ListPopular(ctx, s.pool, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]port.MenuItemDTO, 0, len(items))
	for _, m := range items {
		dtos = append(dtos, port.ToMenuItemDTO(m))
	}
	return dtos, nil
}

// ===== Categories =====

func (s *Service) CreateCategory(ctx context.Context, input port.CreateCategoryInput) (*port.CategoryDTO, error) {
	id := s.deps.IDGenerator.NewID()
	c, err := domain.NewCategory(domain.CategoryParams{
		ID: id, Name: input.Name, NameAr: input.NameAr, Icon: input.Icon, ImageURL: input.ImageURL, SortOrder: input.SortOrder, Now: s.deps.Clock.Now(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.deps.Repos.Categories.Create(ctx, s.pool, c); err != nil {
		return nil, err
	}
	dto := port.ToCategoryDTO(c)
	return &dto, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]port.CategoryDTO, error) {
	cats, err := s.deps.Repos.Categories.List(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	dtos := make([]port.CategoryDTO, 0, len(cats))
	for _, c := range cats {
		dtos = append(dtos, port.ToCategoryDTO(c))
	}
	return dtos, nil
}

// ===== Menu (grouped) =====

func (s *Service) GetMenu(ctx context.Context, restaurantID string) (map[string][]port.MenuItemDTO, error) {
	items, err := s.deps.Repos.MenuItems.ListByRestaurant(ctx, s.pool, restaurantID)
	if err != nil {
		return nil, err
	}
	menu := make(map[string][]port.MenuItemDTO)
	for _, m := range items {
		cat := m.CategoryID()
		if cat == "" {
			cat = "uncategorized"
		}
		menu[cat] = append(menu[cat], port.ToMenuItemDTO(m))
	}
	return menu, nil
}

// suppress unused
var _ = fmt.Sprintf
var _ = slog.Default
var _ = time.Now
