// Package http: catalog HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"avex-backend/internal/modules/catalog/domain"
	"avex-backend/internal/modules/catalog/port"
	idp "avex-backend/internal/modules/identity/port"
	idhttp "avex-backend/internal/modules/identity/transport/http"
)

type Handler struct {
	svc    port.ServicePort
	logger *slog.Logger
}

func NewHandler(svc port.ServicePort, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeErr(w http.ResponseWriter, logger *slog.Logger, err error) {
	status := http.StatusInternalServerError
	switch {
	case err == domain.ErrRestaurantNotFound || err == domain.ErrMenuItemNotFound || err == domain.ErrCategoryNotFound:
		status = http.StatusNotFound
	case err == domain.ErrRestaurantAlreadyExists || err == domain.ErrMenuItemAlreadyExists || err == domain.ErrCategoryAlreadyExists:
		status = http.StatusConflict
	case err == domain.ErrInvalidInput || err == domain.ErrNameRequired || err == domain.ErrInvalidPrice || err == domain.ErrInvalidID:
		status = http.StatusBadRequest
	}
	if status >= 500 && logger != nil {
		logger.Error("internal error", "error", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func parsePage(r *http.Request) port.PageQuery {
	limit, offset := 50, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, e := strconv.Atoi(l); e == nil {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, e := strconv.Atoi(o); e == nil {
			offset = n
		}
	}
	return port.PageQuery{Limit: limit, Offset: offset}
}

// ===== Public Endpoints =====

func (h *Handler) ListRestaurants(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("all") != "true"
	page := parsePage(r)
	result, err := h.svc.ListRestaurants(r.Context(), activeOnly, page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetRestaurant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.GetRestaurant(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetMenu(w http.ResponseWriter, r *http.Request) {
	restaurantID := r.PathValue("id")
	menu, err := h.svc.GetMenu(r.Context(), restaurantID)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) ListPopularItems(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, e := strconv.Atoi(l); e == nil {
			limit = n
		}
	}
	items, err := h.svc.ListPopularItems(r.Context(), limit)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ===== Admin Endpoints (auth required) =====

func (h *Handler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines, ZoneID, MerchantID string
		Lat, Lng, DeliveryFee, MinOrder                                                            float64
		DeliveryTimeMin, DeliveryTimeMax                                                           int
		IsPro                                                                                      bool
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateRestaurant(r.Context(), port.CreateRestaurantInput{
		Name: req.Name, NameAr: req.NameAr, Description: req.Description, DescriptionAr: req.DescriptionAr,
		ImageURL: req.ImageURL, CoverURL: req.CoverURL, Cuisines: req.Cuisines,
		Lat: req.Lat, Lng: req.Lng, ZoneID: req.ZoneID, MerchantID: req.MerchantID,
		DeliveryTimeMin: req.DeliveryTimeMin, DeliveryTimeMax: req.DeliveryTimeMax,
		DeliveryFee: req.DeliveryFee, MinOrder: req.MinOrder, IsPro: req.IsPro,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name, NameAr, Description, DescriptionAr, ImageURL, CoverURL, Cuisines string
		DeliveryTimeMin, DeliveryTimeMax                                       int
		DeliveryFee, MinOrder                                                  float64
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.UpdateRestaurant(r.Context(), id, port.UpdateRestaurantInput{
		Name: req.Name, NameAr: req.NameAr, Description: req.Description, DescriptionAr: req.DescriptionAr,
		ImageURL: req.ImageURL, CoverURL: req.CoverURL, Cuisines: req.Cuisines,
		DeliveryTimeMin: req.DeliveryTimeMin, DeliveryTimeMax: req.DeliveryTimeMax,
		DeliveryFee: req.DeliveryFee, MinOrder: req.MinOrder,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ActivateRestaurant(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ActivateRestaurant(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

func (h *Handler) DeactivateRestaurant(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeactivateRestaurant(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *Handler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RestaurantID, CategoryID, Name, NameAr, Description, DescriptionAr, Image, ImageURL string
		Price                                                                               float64
		IsPopular, IsAvailable                                                              bool
		PrepTime, Calories                                                                  int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateMenuItem(r.Context(), port.CreateMenuItemInput{
		RestaurantID: req.RestaurantID, CategoryID: req.CategoryID,
		Name: req.Name, NameAr: req.NameAr, Description: req.Description, DescriptionAr: req.DescriptionAr,
		Price: req.Price, Image: req.Image, ImageURL: req.ImageURL,
		IsPopular: req.IsPopular, IsAvailable: req.IsAvailable, PrepTime: req.PrepTime, Calories: req.Calories,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name, NameAr, Description, DescriptionAr, Image, ImageURL string
		Price                                                     float64
		IsPopular, IsAvailable                                    bool
		PrepTime, Calories                                        int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.UpdateMenuItem(r.Context(), id, port.UpdateMenuItemInput{
		Name: req.Name, NameAr: req.NameAr, Description: req.Description, DescriptionAr: req.DescriptionAr,
		Price: req.Price, Image: req.Image, ImageURL: req.ImageURL,
		IsPopular: req.IsPopular, IsAvailable: req.IsAvailable, PrepTime: req.PrepTime, Calories: req.Calories,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMenuItem(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name, NameAr, Icon, ImageURL string
		SortOrder                    int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateCategory(r.Context(), port.CreateCategoryInput{
		Name: req.Name, NameAr: req.NameAr, Icon: req.Icon, ImageURL: req.ImageURL, SortOrder: req.SortOrder,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	// Public
	mux.HandleFunc("GET /api/v1/restaurants", h.ListRestaurants)
	mux.HandleFunc("GET /api/v1/restaurants/{id}", h.GetRestaurant)
	mux.HandleFunc("GET /api/v1/restaurants/{id}/menu", h.GetMenu)
	mux.HandleFunc("GET /api/v1/categories", h.ListCategories)
	mux.HandleFunc("GET /api/v1/menu-items/popular", h.ListPopularItems)

	// Admin (auth)
	mux.Handle("POST /api/v1/admin/restaurants", authMW(http.HandlerFunc(h.CreateRestaurant)))
	mux.Handle("PUT /api/v1/admin/restaurants/{id}", authMW(http.HandlerFunc(h.UpdateRestaurant)))
	mux.Handle("POST /api/v1/admin/restaurants/{id}/activate", authMW(http.HandlerFunc(h.ActivateRestaurant)))
	mux.Handle("POST /api/v1/admin/restaurants/{id}/deactivate", authMW(http.HandlerFunc(h.DeactivateRestaurant)))
	mux.Handle("POST /api/v1/admin/menu-items", authMW(http.HandlerFunc(h.CreateMenuItem)))
	mux.Handle("PUT /api/v1/admin/menu-items/{id}", authMW(http.HandlerFunc(h.UpdateMenuItem)))
	mux.Handle("DELETE /api/v1/admin/menu-items/{id}", authMW(http.HandlerFunc(h.DeleteMenuItem)))
	mux.Handle("POST /api/v1/admin/categories", authMW(http.HandlerFunc(h.CreateCategory)))
}
