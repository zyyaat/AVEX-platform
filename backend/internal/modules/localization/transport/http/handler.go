// Package http: localization HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"avex-backend/internal/modules/localization/domain"
	"avex-backend/internal/modules/localization/port"
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
	case err == domain.ErrLanguageNotFound || err == domain.ErrTranslationNotFound:
		status = http.StatusNotFound
	case err == domain.ErrLanguageAlreadyExists || err == domain.ErrTranslationAlreadyExists:
		status = http.StatusConflict
	case err == domain.ErrCannotDeleteDefaultLanguage:
		status = http.StatusUnprocessableEntity
	case err == domain.ErrInvalidID || err == domain.ErrInvalidInput || err == domain.ErrEmptyKey || err == domain.ErrEmptyValue || err == domain.ErrInvalidLanguageCode:
		status = http.StatusBadRequest
	}
	if status >= 500 && logger != nil { logger.Error("internal error", "error", err) }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ===== Public Endpoints (no auth) =====

// GET /api/v1/i18n/languages — list all active languages
func (h *Handler) ListLanguages(w http.ResponseWriter, r *http.Request) {
	langs, err := h.svc.ListActiveLanguages(r.Context())
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, langs)
}

// GET /api/v1/i18n/translate?lang=ar&key=orders.status.pending
func (h *Handler) Translate(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" { lang = "en" }
	key := r.URL.Query().Get("key")
	if key == "" { writeErr(w, h.logger, domain.ErrEmptyKey); return }
	result, err := h.svc.Translate(r.Context(), lang, key)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/i18n/translate/bulk — body: {"language_code":"ar","keys":["k1","k2"]}
func (h *Handler) BulkTranslate(w http.ResponseWriter, r *http.Request) {
	var req struct{ LanguageCode string; Keys []string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	if req.LanguageCode == "" { req.LanguageCode = "en" }
	result, err := h.svc.BulkTranslate(r.Context(), port.BulkTranslateInput{LanguageCode: req.LanguageCode, Keys: req.Keys})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/i18n/translations?lang=ar&prefix=orders.
func (h *Handler) ListTranslations(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" { lang = "en" }
	prefix := r.URL.Query().Get("prefix")
	if prefix != "" {
		result, err := h.svc.ListTranslationsByPrefix(r.Context(), lang, prefix)
		if err != nil { writeErr(w, h.logger, err); return }
		writeJSON(w, http.StatusOK, result)
		return
	}
	result, err := h.svc.ListTranslationsByLanguage(r.Context(), lang)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Admin Endpoints (auth required) =====

func (h *Handler) CreateLanguage(w http.ResponseWriter, r *http.Request) {
	var req struct{ Code, Name string; IsRTL, IsDefault, IsActive bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.CreateLanguage(r.Context(), port.CreateLanguageInput{Code: req.Code, Name: req.Name, IsRTL: req.IsRTL, IsDefault: req.IsDefault, IsActive: req.IsActive})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) DeleteLanguage(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteLanguage(r.Context(), r.PathValue("id")); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) UpsertTranslation(w http.ResponseWriter, r *http.Request) {
	var req struct{ LanguageCode, Key, Value string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.UpsertTranslation(r.Context(), port.UpsertTranslationInput{LanguageCode: req.LanguageCode, Key: req.Key, Value: req.Value})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteTranslation(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	key := r.URL.Query().Get("key")
	if lang == "" || key == "" { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	if err := h.svc.DeleteTranslation(r.Context(), lang, key); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	// Public (no auth)
	mux.HandleFunc("GET /api/v1/i18n/languages", h.ListLanguages)
	mux.HandleFunc("GET /api/v1/i18n/translate", h.Translate)
	mux.HandleFunc("POST /api/v1/i18n/translate/bulk", h.BulkTranslate)
	mux.HandleFunc("GET /api/v1/i18n/translations", h.ListTranslations)

	// Admin (auth)
	mux.Handle("POST /api/v1/admin/i18n/languages", authMW(http.HandlerFunc(h.CreateLanguage)))
	mux.Handle("DELETE /api/v1/admin/i18n/languages/{id}", authMW(http.HandlerFunc(h.DeleteLanguage)))
	mux.Handle("POST /api/v1/admin/i18n/translations", authMW(http.HandlerFunc(h.UpsertTranslation)))
	mux.Handle("DELETE /api/v1/admin/i18n/translations", authMW(http.HandlerFunc(h.DeleteTranslation)))
}
