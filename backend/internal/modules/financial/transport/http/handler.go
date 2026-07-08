// Package http: financial HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/port"
	idp "avex-backend/internal/modules/identity/port"
	idhttp "avex-backend/internal/modules/identity/transport/http"
)

// Handler implements all financial HTTP endpoints.
type Handler struct {
	svc    port.ServicePort
	logger *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(svc port.ServicePort, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ===== Helpers =====

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeErr(w http.ResponseWriter, logger *slog.Logger, err error) {
	status := http.StatusInternalServerError
	switch {
	// Not Found
	case err == domain.ErrWalletNotFound ||
		err == domain.ErrTransactionNotFound ||
		err == domain.ErrPromotionNotFound ||
		err == domain.ErrPricingRuleNotFound:
		status = http.StatusNotFound
	// Conflict (already exists)
	case err == domain.ErrWalletAlreadyExists ||
		err == domain.ErrPromotionCodeAlreadyExists ||
		err == domain.ErrPricingRuleAlreadyExists ||
		err == domain.ErrDuplicateIdempotencyKey ||
		err == domain.ErrPromoAlreadyRedeemed:
		status = http.StatusConflict
	// Business rule violation (422)
	case err == domain.ErrWalletFrozen ||
		err == domain.ErrWalletClosed ||
		err == domain.ErrInsufficientFunds ||
		err == domain.ErrTransactionAlreadyCompleted ||
		err == domain.ErrTransactionAlreadyFailed ||
		err == domain.ErrTransactionCannotBeReversed ||
		err == domain.ErrPromotionInactive ||
		err == domain.ErrPromotionExpired ||
		err == domain.ErrPromotionNotYetValid ||
		err == domain.ErrPromotionUsageLimitReached ||
		err == domain.ErrPromotionPerUserLimitReached ||
		err == domain.ErrPromoMinOrderNotMet:
		status = http.StatusUnprocessableEntity
	// Bad request (validation)
	case err == domain.ErrInvalidInput ||
		err == domain.ErrInvalidID ||
		err == domain.ErrInvalidOwnerType ||
		err == domain.ErrOwnerIDRequired ||
		err == domain.ErrInvalidCurrency ||
		err == domain.ErrInvalidMoneyAmount ||
		err == domain.ErrInvalidTransactionType ||
		err == domain.ErrInvalidTransactionCategory ||
		err == domain.ErrInvalidPromoType ||
		err == domain.ErrInvalidDiscountValue ||
		err == domain.ErrInvalidPercentage ||
		err == domain.ErrInvalidDistance ||
		err == domain.ErrInvalidDuration ||
		err == domain.ErrSurgeMultiplierInvalid ||
		err == domain.ErrCurrencyMismatch ||
		err == domain.ErrNegativeMoneyResult:
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

// ===== Wallet Endpoints =====

// POST /api/v1/admin/wallets
func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OwnerType, OwnerID, Currency string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateWallet(r.Context(), port.CreateWalletInput{
		OwnerType: req.OwnerType, OwnerID: req.OwnerID, Currency: req.Currency,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/wallets/{id}
func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.GetWallet(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/wallets?owner_type=user&owner_id=...&currency=EGP
func (h *Handler) GetWalletByOwner(w http.ResponseWriter, r *http.Request) {
	ownerType := r.URL.Query().Get("owner_type")
	ownerID := r.URL.Query().Get("owner_id")
	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "EGP"
	}
	result, err := h.svc.GetWalletByOwner(r.Context(), ownerType, ownerID, currency)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/wallets/{id}/transactions
func (h *Handler) ListWalletTransactions(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("id")
	page := parsePage(r)
	result, err := h.svc.ListTransactionsByWallet(r.Context(), walletID, page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/wallets/{id}/credit
func (h *Handler) CreditWallet(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("id")
	var req struct {
		Amount         int64
		Currency       string
		Category       string
		ReferenceType  string
		ReferenceID    string
		Description    string
		Metadata       map[string]any
		IdempotencyKey string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	txn, wallet, err := h.svc.Credit(r.Context(), port.CreditInput{
		WalletID:       walletID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Category:       req.Category,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		Description:    req.Description,
		Metadata:       req.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transaction": txn, "wallet": wallet})
}

// POST /api/v1/admin/wallets/{id}/debit
func (h *Handler) DebitWallet(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("id")
	var req struct {
		Amount         int64
		Currency       string
		Category       string
		ReferenceType  string
		ReferenceID    string
		Description    string
		Metadata       map[string]any
		IdempotencyKey string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	txn, wallet, err := h.svc.Debit(r.Context(), port.DebitInput{
		WalletID:       walletID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Category:       req.Category,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		Description:    req.Description,
		Metadata:       req.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transaction": txn, "wallet": wallet})
}

// POST /api/v1/admin/wallets/transfer
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromWalletID   string
		ToWalletID     string
		Amount         int64
		Currency       string
		Category       string
		ReferenceType  string
		ReferenceID    string
		Description    string
		Metadata       map[string]any
		IdempotencyKey string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	debit, credit, err := h.svc.Transfer(r.Context(), port.TransferInput{
		FromWalletID:   req.FromWalletID,
		ToWalletID:     req.ToWalletID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Category:       req.Category,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		Description:    req.Description,
		Metadata:       req.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"debit_transaction": debit, "credit_transaction": credit})
}

// POST /api/v1/admin/wallets/{id}/freeze
func (h *Handler) FreezeWallet(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.FreezeWallet(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "frozen"})
}

// POST /api/v1/admin/wallets/{id}/unfreeze
func (h *Handler) UnfreezeWallet(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.UnfreezeWallet(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// GET /api/v1/transactions/{id}
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.GetTransaction(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ===== Pricing Endpoints =====

// POST /api/v1/pricing/quote (auth required)
func (h *Handler) CalculateQuote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ZoneID      string
		Currency    string
		DistanceKM  float64
		DurationMin int
		OrderTotal  int64
		PromoCode   string
		UserID      string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CalculateQuote(r.Context(), port.CalculateQuoteInput{
		ZoneID:      req.ZoneID,
		Currency:    req.Currency,
		DistanceKM:  req.DistanceKM,
		DurationMin: req.DurationMin,
		OrderTotal:  req.OrderTotal,
		PromoCode:   req.PromoCode,
		UserID:      req.UserID,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/pricing-rules
func (h *Handler) CreatePricingRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ZoneID                string
		Currency              string
		BaseFee               int64
		PerKmRate             int64
		PerMinRate            int64
		MinFee                int64
		MaxFee                *int64
		FreeDeliveryThreshold *int64
		ValidFrom             string
		ValidTo               *string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}

	var validFrom time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse(time.RFC3339, req.ValidFrom)
		if err != nil {
			writeErr(w, h.logger, domain.ErrInvalidInput)
			return
		}
		validFrom = t
	}
	var validTo *time.Time
	if req.ValidTo != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidTo)
		if err != nil {
			writeErr(w, h.logger, domain.ErrInvalidInput)
			return
		}
		validTo = &t
	}

	result, err := h.svc.CreatePricingRule(r.Context(), port.CreatePricingRuleInput{
		ZoneID:                req.ZoneID,
		Currency:              req.Currency,
		BaseFee:               req.BaseFee,
		PerKmRate:             req.PerKmRate,
		PerMinRate:            req.PerMinRate,
		MinFee:                req.MinFee,
		MaxFee:                req.MaxFee,
		FreeDeliveryThreshold: req.FreeDeliveryThreshold,
		ValidFrom:             validFrom,
		ValidTo:               validTo,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/admin/pricing-rules
func (h *Handler) ListPricingRules(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)
	result, err := h.svc.ListPricingRules(r.Context(), page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/surge-zones
func (h *Handler) CreateSurgeZone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ZoneID     string
		Multiplier float64
		Reason     string
		DayOfWeek  *int
		StartTime  string
		EndTime    string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateSurgeZone(r.Context(), port.CreateSurgeZoneInput{
		ZoneID:     req.ZoneID,
		Multiplier: req.Multiplier,
		Reason:     req.Reason,
		DayOfWeek:  req.DayOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/admin/surge-zones
func (h *Handler) ListSurgeZones(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)
	result, err := h.svc.ListSurgeZones(r.Context(), page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/surge-zones/{id}/deactivate
func (h *Handler) DeactivateSurgeZone(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeactivateSurgeZone(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// ===== Promotion Endpoints =====

// GET /api/v1/promotions (public, active only)
func (h *Handler) ListPromotions(w http.ResponseWriter, r *http.Request) {
	promos, err := h.svc.ListActivePromotions(r.Context())
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, promos)
}

// GET /api/v1/promotions/{id}
func (h *Handler) GetPromotion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.GetPromotion(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/promotions/validate (auth)
func (h *Handler) ValidatePromotion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code        string
		OrderTotal  int64
		Currency    string
		DeliveryFee int64
		UserID      string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.ValidatePromotion(r.Context(), port.ValidatePromoInput{
		Code:        req.Code,
		OrderTotal:  req.OrderTotal,
		Currency:    req.Currency,
		DeliveryFee: req.DeliveryFee,
		UserID:      req.UserID,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/promotions/redeem (auth)
func (h *Handler) RedeemPromotion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code        string
		UserID      string
		OrderID     string
		OrderTotal  int64
		Currency    string
		DeliveryFee int64
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.RedeemPromotion(r.Context(), port.RedeemPromoInput{
		Code:        req.Code,
		UserID:      req.UserID,
		OrderID:     req.OrderID,
		OrderTotal:  req.OrderTotal,
		Currency:    req.Currency,
		DeliveryFee: req.DeliveryFee,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/promotions
func (h *Handler) CreatePromotion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code              string
		Description       string
		PromoType         string
		Value             int64
		Currency          string
		MinOrderAmount    int64
		MaxDiscountAmount *int64
		UsageLimit        *int
		PerUserLimit      int
		ValidFrom         string
		ValidTo           *string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	var validFrom time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse(time.RFC3339, req.ValidFrom)
		if err != nil {
			writeErr(w, h.logger, domain.ErrInvalidInput)
			return
		}
		validFrom = t
	}
	var validTo *time.Time
	if req.ValidTo != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidTo)
		if err != nil {
			writeErr(w, h.logger, domain.ErrInvalidInput)
			return
		}
		validTo = &t
	}
	result, err := h.svc.CreatePromotion(r.Context(), port.CreatePromotionInput{
		Code:              req.Code,
		Description:       req.Description,
		PromoType:         req.PromoType,
		Value:             req.Value,
		Currency:          req.Currency,
		MinOrderAmount:    req.MinOrderAmount,
		MaxDiscountAmount: req.MaxDiscountAmount,
		UsageLimit:        req.UsageLimit,
		PerUserLimit:      req.PerUserLimit,
		ValidFrom:         validFrom,
		ValidTo:           validTo,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// ===== Routes =====

// RegisterRoutes wires all financial endpoints into the given mux.
// Routes are split into:
//   - Public read endpoints (no auth): list promotions, get wallet by query
//   - Authenticated endpoints: pricing quote, validate/redeem promo, get own wallet
//   - Admin endpoints (Bearer token required): all writes + freeze/unfreeze
func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	// Public (no auth)
	mux.HandleFunc("GET /api/v1/promotions", h.ListPromotions)
	mux.HandleFunc("GET /api/v1/promotions/{id}", h.GetPromotion)

	// Authenticated (Bearer)
	mux.Handle("GET /api/v1/wallets", authMW(http.HandlerFunc(h.GetWalletByOwner)))
	mux.Handle("GET /api/v1/wallets/{id}", authMW(http.HandlerFunc(h.GetWallet)))
	mux.Handle("GET /api/v1/wallets/{id}/transactions", authMW(http.HandlerFunc(h.ListWalletTransactions)))
	mux.Handle("GET /api/v1/transactions/{id}", authMW(http.HandlerFunc(h.GetTransaction)))
	mux.Handle("POST /api/v1/pricing/quote", authMW(http.HandlerFunc(h.CalculateQuote)))
	mux.Handle("POST /api/v1/promotions/validate", authMW(http.HandlerFunc(h.ValidatePromotion)))
	mux.Handle("POST /api/v1/promotions/redeem", authMW(http.HandlerFunc(h.RedeemPromotion)))

	// Admin (Bearer + role check would go here; for now just auth)
	mux.Handle("POST /api/v1/admin/wallets", authMW(http.HandlerFunc(h.CreateWallet)))
	mux.Handle("POST /api/v1/admin/wallets/{id}/credit", authMW(http.HandlerFunc(h.CreditWallet)))
	mux.Handle("POST /api/v1/admin/wallets/{id}/debit", authMW(http.HandlerFunc(h.DebitWallet)))
	mux.Handle("POST /api/v1/admin/wallets/transfer", authMW(http.HandlerFunc(h.Transfer)))
	mux.Handle("POST /api/v1/admin/wallets/{id}/freeze", authMW(http.HandlerFunc(h.FreezeWallet)))
	mux.Handle("POST /api/v1/admin/wallets/{id}/unfreeze", authMW(http.HandlerFunc(h.UnfreezeWallet)))
	mux.Handle("POST /api/v1/admin/pricing-rules", authMW(http.HandlerFunc(h.CreatePricingRule)))
	mux.Handle("GET /api/v1/admin/pricing-rules", authMW(http.HandlerFunc(h.ListPricingRules)))
	mux.Handle("POST /api/v1/admin/surge-zones", authMW(http.HandlerFunc(h.CreateSurgeZone)))
	mux.Handle("GET /api/v1/admin/surge-zones", authMW(http.HandlerFunc(h.ListSurgeZones)))
	mux.Handle("POST /api/v1/admin/surge-zones/{id}/deactivate", authMW(http.HandlerFunc(h.DeactivateSurgeZone)))
	mux.Handle("POST /api/v1/admin/promotions", authMW(http.HandlerFunc(h.CreatePromotion)))
}
