// Package http: settings HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"avex-backend/internal/modules/settings/domain"
	"avex-backend/internal/modules/settings/port"
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
	case err == domain.ErrSettingNotFound || err == domain.ErrRevisionNotFound || err == domain.ErrFeatureFlagNotFound:
		status = http.StatusNotFound
	case err == domain.ErrSettingAlreadyExists || err == domain.ErrFeatureFlagAlreadyExists:
		status = http.StatusConflict
	case err == domain.ErrCannotDeleteProtected:
		status = http.StatusUnprocessableEntity
	case err == domain.ErrInvalidID || err == domain.ErrInvalidInput || err == domain.ErrEmptyKey || err == domain.ErrEmptyName ||
		err == domain.ErrInvalidSettingType || err == domain.ErrInvalidSettingValue || err == domain.ErrInvalidTargetType || err == domain.ErrInvalidRolloutPercentage:
		status = http.StatusBadRequest
	}
	if status >= 500 && logger != nil { logger.Error("internal error", "error", err) }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func parsePage(r *http.Request) port.PageQuery {
	limit, offset := 50, 0
	if l := r.URL.Query().Get("limit"); l != "" { if n, e := strconv.Atoi(l); e == nil { limit = n } }
	if o := r.URL.Query().Get("offset"); o != "" { if n, e := strconv.Atoi(o); e == nil { offset = n } }
	return port.PageQuery{Limit: limit, Offset: offset}
}

// ===== Setting Endpoints =====

func (h *Handler) CreateSetting(w http.ResponseWriter, r *http.Request) {
	var req struct{ Key, Description, Type, Value string; IsProtected bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.CreateSetting(r.Context(), port.CreateSettingInput{Key: req.Key, Description: req.Description, Type: req.Type, Value: req.Value, IsProtected: req.IsProtected})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetSetting(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetSetting(r.Context(), r.PathValue("id"))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetSettingByKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" { writeErr(w, h.logger, domain.ErrEmptyKey); return }
	result, err := h.svc.GetSettingByKey(r.Context(), key)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Value, ChangedBy, ChangeNote string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.UpdateSetting(r.Context(), id, port.UpdateSettingInput{Value: req.Value, ChangedBy: req.ChangedBy, ChangeNote: req.ChangeNote})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteSetting(r.Context(), r.PathValue("id")); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListSettings(w http.ResponseWriter, r *http.Request) {
	if t := r.URL.Query().Get("type"); t != "" {
		result, err := h.svc.ListSettingsByType(r.Context(), t)
		if err != nil { writeErr(w, h.logger, err); return }
		writeJSON(w, http.StatusOK, result)
		return
	}
	result, err := h.svc.ListSettings(r.Context(), parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListRevisions(r.Context(), r.PathValue("id"), parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RollbackSetting(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Version int; ChangedBy string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.RollbackSetting(r.Context(), id, req.Version, req.ChangedBy)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Feature Flag Endpoints =====

func (h *Handler) CreateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Description string; Enabled bool; TargetType, TargetValue string; RolloutPct int }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.CreateFeatureFlag(r.Context(), port.CreateFeatureFlagInput{Name: req.Name, Description: req.Description, Enabled: req.Enabled, TargetType: req.TargetType, TargetValue: req.TargetValue, RolloutPct: req.RolloutPct})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetFeatureFlag(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetFeatureFlag(r.Context(), r.PathValue("id"))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Enabled *bool; TargetType, TargetValue string; RolloutPct *int }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.UpdateFeatureFlag(r.Context(), id, port.UpdateFeatureFlagInput{Enabled: req.Enabled, TargetType: req.TargetType, TargetValue: req.TargetValue, RolloutPct: req.RolloutPct})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteFeatureFlag(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteFeatureFlag(r.Context(), r.PathValue("id")); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListFeatureFlags(r.Context(), parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Check =====

func (h *Handler) CheckFeatureFlag(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" { writeErr(w, h.logger, domain.ErrEmptyName); return }
	userID := r.URL.Query().Get("user_id")
	// roles as comma-separated
	var roles []string
	if rolesStr := r.URL.Query().Get("roles"); rolesStr != "" {
		// Simple split
		start := 0
		for i := 0; i <= len(rolesStr); i++ {
			if i == len(rolesStr) || rolesStr[i] == ',' {
				if i > start { roles = append(roles, rolesStr[start:i]) }
				start = i + 1
			}
		}
	}
	result, err := h.svc.IsFeatureEnabled(r.Context(), name, userID, roles)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	// Settings
	mux.Handle("POST /api/v1/admin/settings", authMW(http.HandlerFunc(h.CreateSetting)))
	mux.Handle("GET /api/v1/admin/settings", authMW(http.HandlerFunc(h.ListSettings)))
	mux.Handle("GET /api/v1/admin/settings/by-key", authMW(http.HandlerFunc(h.GetSettingByKey)))
	mux.Handle("GET /api/v1/admin/settings/{id}", authMW(http.HandlerFunc(h.GetSetting)))
	mux.Handle("PUT /api/v1/admin/settings/{id}", authMW(http.HandlerFunc(h.UpdateSetting)))
	mux.Handle("DELETE /api/v1/admin/settings/{id}", authMW(http.HandlerFunc(h.DeleteSetting)))
	mux.Handle("GET /api/v1/admin/settings/{id}/revisions", authMW(http.HandlerFunc(h.ListRevisions)))
	mux.Handle("POST /api/v1/admin/settings/{id}/rollback", authMW(http.HandlerFunc(h.RollbackSetting)))

	// Feature Flags
	mux.Handle("POST /api/v1/admin/feature-flags", authMW(http.HandlerFunc(h.CreateFeatureFlag)))
	mux.Handle("GET /api/v1/admin/feature-flags", authMW(http.HandlerFunc(h.ListFeatureFlags)))
	mux.Handle("GET /api/v1/admin/feature-flags/{id}", authMW(http.HandlerFunc(h.GetFeatureFlag)))
	mux.Handle("PUT /api/v1/admin/feature-flags/{id}", authMW(http.HandlerFunc(h.UpdateFeatureFlag)))
	mux.Handle("DELETE /api/v1/admin/feature-flags/{id}", authMW(http.HandlerFunc(h.DeleteFeatureFlag)))

	// Check (authenticated — any logged-in user can check flags)
	mux.Handle("GET /api/v1/feature-flags/check", authMW(http.HandlerFunc(h.CheckFeatureFlag)))
}
