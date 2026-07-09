// Package http: audit HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"avex-backend/internal/modules/audit/domain"
	"avex-backend/internal/modules/audit/port"
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
	case err == domain.ErrAuditEntryNotFound:
		status = http.StatusNotFound
	case err == domain.ErrCannotModifyAuditEntry:
		status = http.StatusMethodNotAllowed
	case err == domain.ErrInvalidID || err == domain.ErrInvalidInput ||
		err == domain.ErrEmptyAction || err == domain.ErrEmptyResourceType ||
		err == domain.ErrEmptyActorID || err == domain.ErrInvalidSeverity || err == domain.ErrInvalidActorType:
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

// ===== Endpoints =====

// GET /api/v1/admin/audit/{id}
func (h *Handler) GetEntry(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetEntry(r.Context(), r.PathValue("id"))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/admin/audit?actor_type=...&actor_id=...&resource_type=...&resource_id=...&action=...&severity=...&from=...&to=...
func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)
	q := r.URL.Query()

	// Determine which filter to apply based on query params
	if at := q.Get("actor_type"); at != "" {
		if aid := q.Get("actor_id"); aid != "" {
			result, err := h.svc.ListByActor(r.Context(), at, aid, page)
			if err != nil { writeErr(w, h.logger, err); return }
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	if rt := q.Get("resource_type"); rt != "" {
		if rid := q.Get("resource_id"); rid != "" {
			result, err := h.svc.ListByResource(r.Context(), rt, rid, page)
			if err != nil { writeErr(w, h.logger, err); return }
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	if action := q.Get("action"); action != "" {
		result, err := h.svc.ListByAction(r.Context(), action, page)
		if err != nil { writeErr(w, h.logger, err); return }
		writeJSON(w, http.StatusOK, result)
		return
	}
	if sev := q.Get("severity"); sev != "" {
		result, err := h.svc.ListBySeverity(r.Context(), sev, page)
		if err != nil { writeErr(w, h.logger, err); return }
		writeJSON(w, http.StatusOK, result)
		return
	}
	if from := q.Get("from"); from != "" {
		if to := q.Get("to"); to != "" {
			fromTime, err1 := time.Parse(time.RFC3339, from)
			toTime, err2 := time.Parse(time.RFC3339, to)
			if err1 == nil && err2 == nil {
				result, err := h.svc.ListByTimeRange(r.Context(), fromTime, toTime, page)
				if err != nil { writeErr(w, h.logger, err); return }
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
	}
	// Default: list all
	result, err := h.svc.ListAll(r.Context(), page)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/admin/audit/stats?from=...&to=...
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	now := time.Now().UTC()
	fromTime := now.Add(-24 * time.Hour)
	toTime := now
	if from != "" { if t, err := time.Parse(time.RFC3339, from); err == nil { fromTime = t } }
	if to != "" { if t, err := time.Parse(time.RFC3339, to); err == nil { toTime = t } }
	result, err := h.svc.GetStats(r.Context(), fromTime, toTime)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/admin/audit — manually log an audit entry
func (h *Handler) LogAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActorType, ActorID, Action, ResourceType, ResourceID, Severity, Description string
		Metadata map[string]any
		IPAddress, UserAgent, CorrelationID, TraceID string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.Log(r.Context(), port.LogActionInput{
		ActorType: req.ActorType, ActorID: req.ActorID, Action: req.Action,
		ResourceType: req.ResourceType, ResourceID: req.ResourceID,
		Severity: req.Severity, Description: req.Description,
		Metadata: req.Metadata, IPAddress: req.IPAddress, UserAgent: req.UserAgent,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID,
	})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	mux.Handle("GET /api/v1/admin/audit", authMW(http.HandlerFunc(h.ListEntries)))
	mux.Handle("GET /api/v1/admin/audit/stats", authMW(http.HandlerFunc(h.GetStats)))
	mux.Handle("GET /api/v1/admin/audit/{id}", authMW(http.HandlerFunc(h.GetEntry)))
	mux.Handle("POST /api/v1/admin/audit", authMW(http.HandlerFunc(h.LogAction)))
}
