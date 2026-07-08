// Package http: notifications HTTP transport.
package http

import (
        "encoding/json"
        "fmt"
        "log/slog"
        "net/http"

        "avex-backend/internal/modules/notifications/domain"
        "avex-backend/internal/modules/notifications/port"
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
        case err == domain.ErrNotificationNotFound || err == domain.ErrPreferenceNotFound:
                status = http.StatusNotFound
        case err == domain.ErrNotificationAlreadyExists || err == domain.ErrPreferenceAlreadyExists || err == domain.ErrDeviceAlreadyExists:
                status = http.StatusConflict
        case err == domain.ErrNotificationAlreadySent || err == domain.ErrNotificationAlreadyFailed ||
                err == domain.ErrNotificationCannotRetry || err == domain.ErrMaxRetriesReached ||
                err == domain.ErrNoChannelEnabled:
                status = http.StatusUnprocessableEntity
        case err == domain.ErrInvalidID || err == domain.ErrInvalidInput ||
                err == domain.ErrInvalidChannel || err == domain.ErrInvalidNotificationType ||
                err == domain.ErrInvalidNotificationStatus || err == domain.ErrInvalidPlatform ||
                err == domain.ErrEmptyRecipientID || err == domain.ErrEmptyTitle || err == domain.ErrEmptyBody:
                status = http.StatusBadRequest
        }
        if status >= 500 && logger != nil {
                logger.Error("internal error", "error", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ===== Endpoints =====

// POST /api/v1/notifications/send
func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
        var req struct {
                RecipientType string
                RecipientID   string
                Type          string
                Channels      []string
                Title         string
                TitleAr       string
                Body          string
                BodyAr        string
                Data          map[string]any
                Priority      string
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        results, err := h.svc.SendNotification(r.Context(), port.SendNotificationInput{
                RecipientType: req.RecipientType,
                RecipientID:   req.RecipientID,
                Type:          req.Type,
                Channels:      req.Channels,
                Title:         req.Title,
                TitleAr:       req.TitleAr,
                Body:          req.Body,
                BodyAr:        req.BodyAr,
                Data:          req.Data,
                Priority:      req.Priority,
        })
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, results)
}

// GET /api/v1/notifications/{id}
func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.GetNotification(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/notifications?recipient_type=user&recipient_id=...
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
        recipientType := r.URL.Query().Get("recipient_type")
        recipientID := r.URL.Query().Get("recipient_id")
        if recipientType == "" || recipientID == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        limit, offset := 50, 0
        if l := r.URL.Query().Get("limit"); l != "" {
                _, _ = fmt.Sscanf(l, "%d", &limit)
        }
        if o := r.URL.Query().Get("offset"); o != "" {
                _, _ = fmt.Sscanf(o, "%d", &offset)
        }
        result, err := h.svc.ListNotificationsByRecipient(r.Context(), recipientType, recipientID, port.PageQuery{Limit: limit, Offset: offset})
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/notifications/{id}/retry
func (h *Handler) RetryFailed(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        result, err := h.svc.RetryFailed(r.Context(), id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/notifications/preferences?recipient_type=user&recipient_id=...
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
        recipientType := r.URL.Query().Get("recipient_type")
        recipientID := r.URL.Query().Get("recipient_id")
        if recipientType == "" || recipientID == "" {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.GetPreferences(r.Context(), recipientType, recipientID)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// PUT /api/v1/notifications/preferences
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
        var req struct {
                RecipientType string
                RecipientID   string
                PhoneNumber   string
                Email         string
                DeviceToken   string
                RemoveToken   string
                Prefs         map[string]bool
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, domain.ErrInvalidInput)
                return
        }
        result, err := h.svc.UpdatePreferences(r.Context(), port.UpdatePreferenceInput{
                RecipientType: req.RecipientType,
                RecipientID:   req.RecipientID,
                PhoneNumber:   req.PhoneNumber,
                Email:         req.Email,
                DeviceToken:   req.DeviceToken,
                RemoveToken:   req.RemoveToken,
                Prefs:         req.Prefs,
        })
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, result)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
        h := NewHandler(svc, logger)
        authMW := idhttp.Auth(jwtIssuer, logger)

        // Authenticated
        mux.Handle("POST /api/v1/notifications/send", authMW(http.HandlerFunc(h.SendNotification)))
        mux.Handle("GET /api/v1/notifications/{id}", authMW(http.HandlerFunc(h.GetNotification)))
        mux.Handle("GET /api/v1/notifications", authMW(http.HandlerFunc(h.ListNotifications)))
        mux.Handle("POST /api/v1/notifications/{id}/retry", authMW(http.HandlerFunc(h.RetryFailed)))
        mux.Handle("GET /api/v1/notifications/preferences", authMW(http.HandlerFunc(h.GetPreferences)))
        mux.Handle("PUT /api/v1/notifications/preferences", authMW(http.HandlerFunc(h.UpdatePreferences)))
}
