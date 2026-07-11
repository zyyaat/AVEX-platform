// Package postgres implements catalog repository using pgx/v5.
package postgres

import (
        "context"
        "errors"
        "fmt"
        "time"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgconn"
        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/catalog/domain"
        "avex-backend/internal/modules/catalog/port"
        "avex-backend/internal/platform/database"
)

func toDBTX(exec port.Executor) database.DBTX {
        dbtx, ok := exec.(database.DBTX)
        if !ok {
                panic("postgres: port.Executor does not satisfy database.DBTX")
        }
        return dbtx
}

// ===== Repositories struct =====

type Repositories struct {
        restaurants *RestaurantRepository
        menuItems   *MenuItemRepository
        categories  *CategoryRepository
        storeHours  *StoreHoursRepository
}

func NewRepositories() *Repositories {
        return &Repositories{
                restaurants: &RestaurantRepository{},
                menuItems:   &MenuItemRepository{},
                categories:  &CategoryRepository{},
                storeHours:  &StoreHoursRepository{},
        }
}

func (r *Repositories) RepositorySet() port.RepositorySet {
        return port.RepositorySet{
                Restaurants: r.restaurants,
                MenuItems:   r.menuItems,
                Categories:  r.categories,
                StoreHours:  r.storeHours,
        }
}

// ===== RestaurantRepository =====

const restaurantColumns = `id, name, name_ar, description, description_ar, image_url, cover_url, cuisines, lat, lng, zone_id, merchant_id, rating, rating_count, delivery_time_min, delivery_time_max, delivery_fee, min_order, is_active, is_pro, created_at, updated_at`

type RestaurantRepository struct{}

func (r *RestaurantRepository) Create(ctx context.Context, exec port.Executor, rest domain.Restaurant) error {
        dbtx := toDBTX(exec)
        // merchant_id is UUID (nullable). Empty string causes PostgreSQL error.
        // Use NULL when merchant_id is empty.
        var merchantID interface{}
        if rest.MerchantID() != "" {
                merchantID = rest.MerchantID()
        }
        _, err := dbtx.Exec(ctx, `INSERT INTO catalog.restaurants (id,name,name_ar,description,description_ar,image_url,cover_url,cuisines,lat,lng,zone_id,merchant_id,rating,rating_count,delivery_time_min,delivery_time_max,delivery_fee,min_order,is_active,is_pro,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
                rest.ID(), rest.Name(), rest.NameAr(), rest.Description(), rest.DescriptionAr(), rest.ImageURL(), rest.CoverURL(), rest.Cuisines(), rest.Lat(), rest.Lng(), rest.ZoneID(), merchantID, rest.Rating(), rest.RatingCount(), rest.DeliveryTimeMin(), rest.DeliveryTimeMax(), rest.DeliveryFee(), rest.MinOrder(), rest.IsActive(), rest.IsPro(), rest.CreatedAt(), rest.UpdatedAt())
        if err != nil {
                var pgErr *pgconn.PgError
                if errors.As(err, &pgErr) && pgErr.Code == "23505" {
                        return domain.ErrRestaurantAlreadyExists
                }
                return fmt.Errorf("create restaurant: %w", err)
        }
        return nil
}

func (r *RestaurantRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Restaurant, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT `+restaurantColumns+` FROM catalog.restaurants WHERE id=$1`, id)
        rest, err := scanRestaurant(row)
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return nil, domain.ErrRestaurantNotFound
                }
                return nil, fmt.Errorf("get restaurant: %w", err)
        }
        return &rest, nil
}

func (r *RestaurantRepository) Update(ctx context.Context, exec port.Executor, rest domain.Restaurant) error {
        dbtx := toDBTX(exec)
        ct, err := dbtx.Exec(ctx, `UPDATE catalog.restaurants SET name=$1,name_ar=$2,description=$3,description_ar=$4,image_url=$5,cover_url=$6,cuisines=$7,delivery_time_min=$8,delivery_time_max=$9,delivery_fee=$10,min_order=$11,is_active=$12,is_pro=$13,updated_at=$14 WHERE id=$15`,
                rest.Name(), rest.NameAr(), rest.Description(), rest.DescriptionAr(), rest.ImageURL(), rest.CoverURL(), rest.Cuisines(), rest.DeliveryTimeMin(), rest.DeliveryTimeMax(), rest.DeliveryFee(), rest.MinOrder(), rest.IsActive(), rest.IsPro(), rest.UpdatedAt(), rest.ID())
        if err != nil {
                return fmt.Errorf("update restaurant: %w", err)
        }
        if ct.RowsAffected() == 0 {
                return domain.ErrRestaurantNotFound
        }
        return nil
}

func (r *RestaurantRepository) List(ctx context.Context, exec port.Executor, activeOnly bool, page port.PageQuery) (port.Page[domain.Restaurant], error) {
        page = page.Normalize()
        dbtx := toDBTX(exec)
        where := ""
        if activeOnly {
                where = "WHERE is_active = TRUE"
        }
        var total int64
        _ = dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM catalog.restaurants `+where).Scan(&total)
        rows, err := dbtx.Query(ctx, `SELECT `+restaurantColumns+` FROM catalog.restaurants `+where+` ORDER BY created_at DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
        if err != nil {
                return port.Page[domain.Restaurant]{}, fmt.Errorf("list restaurants: %w", err)
        }
        defer rows.Close()
        var items []domain.Restaurant
        for rows.Next() {
                rest, err := scanRestaurant(rows)
                if err != nil {
                        return port.Page[domain.Restaurant]{}, err
                }
                items = append(items, rest)
        }
        return port.Page[domain.Restaurant]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (r *RestaurantRepository) ListByZone(ctx context.Context, exec port.Executor, zoneID string, page port.PageQuery) (port.Page[domain.Restaurant], error) {
        page = page.Normalize()
        dbtx := toDBTX(exec)
        var total int64
        _ = dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM catalog.restaurants WHERE zone_id=$1 AND is_active=TRUE`, zoneID).Scan(&total)
        rows, err := dbtx.Query(ctx, `SELECT `+restaurantColumns+` FROM catalog.restaurants WHERE zone_id=$1 AND is_active=TRUE ORDER BY rating DESC LIMIT $2 OFFSET $3`, zoneID, page.Limit, page.Offset)
        if err != nil {
                return port.Page[domain.Restaurant]{}, err
        }
        defer rows.Close()
        var items []domain.Restaurant
        for rows.Next() {
                rest, err := scanRestaurant(rows)
                if err != nil {
                        return port.Page[domain.Restaurant]{}, err
                }
                items = append(items, rest)
        }
        return port.Page[domain.Restaurant]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func scanRestaurant(row pgx.Row) (domain.Restaurant, error) {
        var r domain.RestaurantRecord
        err := row.Scan(&r.ID, &r.Name, &r.NameAr, &r.Description, &r.DescriptionAr, &r.ImageURL, &r.CoverURL, &r.Cuisines, &r.Lat, &r.Lng, &r.ZoneID, &r.MerchantID, &r.Rating, &r.RatingCount, &r.DeliveryTimeMin, &r.DeliveryTimeMax, &r.DeliveryFee, &r.MinOrder, &r.IsActive, &r.IsPro, &r.CreatedAt, &r.UpdatedAt)
        if err != nil {
                return domain.Restaurant{}, err
        }
        return domain.ReconstructRestaurant(r), nil
}

// ===== MenuItemRepository =====

const menuItemColumns = `id, restaurant_id, category_id, name, name_ar, description, description_ar, price, image, image_url, is_popular, is_available, rating, rating_count, prep_time, calories, created_at, updated_at`

type MenuItemRepository struct{}

func (r *MenuItemRepository) Create(ctx context.Context, exec port.Executor, m domain.MenuItem) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO catalog.menu_items (id,restaurant_id,category_id,name,name_ar,description,description_ar,price,image,image_url,is_popular,is_available,rating,rating_count,prep_time,calories,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
                m.ID(), m.RestaurantID(), m.CategoryID(), m.Name(), m.NameAr(), m.Description(), m.DescriptionAr(), m.Price(), m.Image(), m.ImageURL(), m.IsPopular(), m.IsAvailable(), m.Rating(), m.RatingCount(), m.PrepTime(), m.Calories(), m.CreatedAt(), m.UpdatedAt())
        if err != nil {
                var pgErr *pgconn.PgError
                if errors.As(err, &pgErr) && pgErr.Code == "23505" {
                        return domain.ErrMenuItemAlreadyExists
                }
                return fmt.Errorf("create menu item: %w", err)
        }
        return nil
}

func (r *MenuItemRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.MenuItem, error) {
        dbtx := toDBTX(exec)
        row := dbtx.QueryRow(ctx, `SELECT `+menuItemColumns+` FROM catalog.menu_items WHERE id=$1`, id)
        m, err := scanMenuItem(row)
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return nil, domain.ErrMenuItemNotFound
                }
                return nil, err
        }
        return &m, nil
}

func (r *MenuItemRepository) Update(ctx context.Context, exec port.Executor, m domain.MenuItem) error {
        dbtx := toDBTX(exec)
        ct, err := dbtx.Exec(ctx, `UPDATE catalog.menu_items SET name=$1,name_ar=$2,description=$3,description_ar=$4,price=$5,image=$6,image_url=$7,is_popular=$8,is_available=$9,prep_time=$10,calories=$11,updated_at=$12 WHERE id=$13`,
                m.Name(), m.NameAr(), m.Description(), m.DescriptionAr(), m.Price(), m.Image(), m.ImageURL(), m.IsPopular(), m.IsAvailable(), m.PrepTime(), m.Calories(), m.UpdatedAt(), m.ID())
        if err != nil {
                return err
        }
        if ct.RowsAffected() == 0 {
                return domain.ErrMenuItemNotFound
        }
        return nil
}

func (r *MenuItemRepository) Delete(ctx context.Context, exec port.Executor, id string) error {
        dbtx := toDBTX(exec)
        ct, err := dbtx.Exec(ctx, `DELETE FROM catalog.menu_items WHERE id=$1`, id)
        if err != nil {
                return err
        }
        if ct.RowsAffected() == 0 {
                return domain.ErrMenuItemNotFound
        }
        return nil
}

func (r *MenuItemRepository) ListByRestaurant(ctx context.Context, exec port.Executor, restaurantID string) ([]domain.MenuItem, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT `+menuItemColumns+` FROM catalog.menu_items WHERE restaurant_id=$1 ORDER BY name`, restaurantID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        var items []domain.MenuItem
        for rows.Next() {
                m, err := scanMenuItem(rows)
                if err != nil {
                        return nil, err
                }
                items = append(items, m)
        }
        return items, nil
}

func (r *MenuItemRepository) ListPopular(ctx context.Context, exec port.Executor, limit int) ([]domain.MenuItem, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT `+menuItemColumns+` FROM catalog.menu_items WHERE is_popular=TRUE AND is_available=TRUE ORDER BY rating DESC LIMIT $1`, limit)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        var items []domain.MenuItem
        for rows.Next() {
                m, err := scanMenuItem(rows)
                if err != nil {
                        return nil, err
                }
                items = append(items, m)
        }
        return items, nil
}

func scanMenuItem(row pgx.Row) (domain.MenuItem, error) {
        var m domain.MenuItemRecord
        err := row.Scan(&m.ID, &m.RestaurantID, &m.CategoryID, &m.Name, &m.NameAr, &m.Description, &m.DescriptionAr, &m.Price, &m.Image, &m.ImageURL, &m.IsPopular, &m.IsAvailable, &m.Rating, &m.RatingCount, &m.PrepTime, &m.Calories, &m.CreatedAt, &m.UpdatedAt)
        if err != nil {
                return domain.MenuItem{}, err
        }
        return domain.ReconstructMenuItem(m), nil
}

// ===== CategoryRepository =====

type CategoryRepository struct{}

func (r *CategoryRepository) Create(ctx context.Context, exec port.Executor, c domain.Category) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO catalog.categories (id,name,name_ar,icon,image_url,sort_order,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
                c.ID(), c.Name(), c.NameAr(), c.Icon(), c.ImageURL(), c.SortOrder(), c.CreatedAt())
        if err != nil {
                var pgErr *pgconn.PgError
                if errors.As(err, &pgErr) && pgErr.Code == "23505" {
                        return domain.ErrCategoryAlreadyExists
                }
                return err
        }
        return nil
}

func (r *CategoryRepository) List(ctx context.Context, exec port.Executor) ([]domain.Category, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT id,name,name_ar,icon,image_url,sort_order,created_at FROM catalog.categories ORDER BY sort_order`)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        var cats []domain.Category
        for rows.Next() {
                var c domain.CategoryRecord
                if err := rows.Scan(&c.ID, &c.Name, &c.NameAr, &c.Icon, &c.ImageURL, &c.SortOrder, &c.CreatedAt); err != nil {
                        return nil, err
                }
                cats = append(cats, domain.ReconstructCategory(c))
        }
        return cats, nil
}

// ===== StoreHoursRepository =====

type StoreHoursRepository struct{}

func (r *StoreHoursRepository) Upsert(ctx context.Context, exec port.Executor, sh domain.StoreHours) error {
        dbtx := toDBTX(exec)
        _, err := dbtx.Exec(ctx, `INSERT INTO catalog.store_hours (id,restaurant_id,day_of_week,open_time,close_time,is_open) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (restaurant_id,day_of_week) DO UPDATE SET open_time=$4,close_time=$5,is_open=$6`,
                sh.ID(), sh.RestaurantID(), sh.DayOfWeek(), sh.OpenTime(), sh.CloseTime(), sh.IsOpen())
        return err
}

func (r *StoreHoursRepository) ListByRestaurant(ctx context.Context, exec port.Executor, restaurantID string) ([]domain.StoreHours, error) {
        dbtx := toDBTX(exec)
        rows, err := dbtx.Query(ctx, `SELECT id,restaurant_id,day_of_week,open_time,close_time,is_open FROM catalog.store_hours WHERE restaurant_id=$1 ORDER BY day_of_week`, restaurantID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        var hours []domain.StoreHours
        for rows.Next() {
                var sh domain.StoreHoursRecord
                if err := rows.Scan(&sh.ID, &sh.RestaurantID, &sh.DayOfWeek, &sh.OpenTime, &sh.CloseTime, &sh.IsOpen); err != nil {
                        return nil, err
                }
                hours = append(hours, domain.ReconstructStoreHours(sh))
        }
        return hours, nil
}

// suppress unused
var _ = time.Now
var _ = pgxpool.Pool{}
