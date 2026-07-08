// Package port repository: persistence interfaces for catalog module.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/catalog/domain"
)

type Executor interface{}
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}

type PageQuery struct{ Limit, Offset int }

func (p PageQuery) Normalize() PageQuery {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

type Page[T any] struct {
	Items         []T
	Total         int64
	Limit, Offset int
}

type RestaurantRepository interface {
	Create(ctx context.Context, exec Executor, r domain.Restaurant) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Restaurant, error)
	Update(ctx context.Context, exec Executor, r domain.Restaurant) error
	List(ctx context.Context, exec Executor, activeOnly bool, page PageQuery) (Page[domain.Restaurant], error)
	ListByZone(ctx context.Context, exec Executor, zoneID string, page PageQuery) (Page[domain.Restaurant], error)
}

type MenuItemRepository interface {
	Create(ctx context.Context, exec Executor, m domain.MenuItem) error
	GetByID(ctx context.Context, exec Executor, id string) (*domain.MenuItem, error)
	Update(ctx context.Context, exec Executor, m domain.MenuItem) error
	Delete(ctx context.Context, exec Executor, id string) error
	ListByRestaurant(ctx context.Context, exec Executor, restaurantID string) ([]domain.MenuItem, error)
	ListPopular(ctx context.Context, exec Executor, limit int) ([]domain.MenuItem, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, exec Executor, c domain.Category) error
	List(ctx context.Context, exec Executor) ([]domain.Category, error)
}

type StoreHoursRepository interface {
	Upsert(ctx context.Context, exec Executor, sh domain.StoreHours) error
	ListByRestaurant(ctx context.Context, exec Executor, restaurantID string) ([]domain.StoreHours, error)
}

type RepositorySet struct {
	Restaurants RestaurantRepository
	MenuItems   MenuItemRepository
	Categories  CategoryRepository
	StoreHours  StoreHoursRepository
}

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Logger interface {
	Debug(string, ...any)
	Info(string, ...any)
	Warn(string, ...any)
	Error(string, ...any)
}

type Deps struct {
	Clock       Clock
	IDGenerator IDGenerator
	Logger      Logger
	TxRunner    TxRunner
	Repos       RepositorySet
}
