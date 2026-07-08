// Package http: support HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"avex-backend/internal/modules/support/domain"
	"avex-backend/internal/modules/support/port"
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
	case err == domain.ErrTicketNotFound || err == domain.ErrMessageNotFound || err == domain.ErrAttachmentNotFound:
		status = http.StatusNotFound
	case err == domain.ErrTicketAlreadyExists || err == domain.ErrTicketAlreadyAssigned:
		status = http.StatusConflict
	case err == domain.ErrTicketClosed || err == domain.ErrTicketNotAssigned ||
		err == domain.ErrNotTicketOwner || err == domain.ErrNotAssignedAgent ||
		err == domain.ErrCannotEditMessage:
		status = http.StatusUnprocessableEntity
	case err == domain.ErrInvalidID || err == domain.ErrInvalidInput ||
		err == domain.ErrInvalidTicketStatus || err == domain.ErrInvalidTicketPriority ||
		err == domain.ErrInvalidTicketCategory || err == domain.ErrInvalidMessageType ||
		err == domain.ErrInvalidFileType || err == domain.ErrFileTooLarge ||
		err == domain.ErrEmptySubject || err == domain.ErrEmptyUserID ||
		err == domain.ErrEmptyMessage:
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

// ===== User Endpoints =====

// POST /api/v1/support/tickets
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID, OrderID, DriverID, RestaurantID, Subject, Description, Category, Priority, CreatedBy string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CreateTicket(r.Context(), port.CreateTicketInput{
		UserID: req.UserID, OrderID: req.OrderID, DriverID: req.DriverID, RestaurantID: req.RestaurantID,
		Subject: req.Subject, Description: req.Description,
		Category: req.Category, Priority: req.Priority, CreatedBy: req.CreatedBy,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/support/tickets/{id}
func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.GetTicket(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/support/tickets?user_id=...&status=...&agent_id=...
func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		result, err := h.svc.ListMyTickets(r.Context(), userID, page)
		if err != nil {
			writeErr(w, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		result, err := h.svc.ListAgentTickets(r.Context(), agentID, page)
		if err != nil {
			writeErr(w, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if status := r.URL.Query().Get("status"); status != "" {
		result, err := h.svc.ListTicketsByStatus(r.Context(), status, page)
		if err != nil {
			writeErr(w, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if unassigned := r.URL.Query().Get("unassigned"); unassigned == "true" {
		result, err := h.svc.ListUnassignedTickets(r.Context(), page)
		if err != nil {
			writeErr(w, h.logger, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	result, err := h.svc.ListAllTickets(r.Context(), page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/assign
func (h *Handler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ AgentID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.AssignTicket(r.Context(), port.AssignTicketInput{TicketID: id, AgentID: req.AgentID})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/unassign
func (h *Handler) UnassignTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.UnassignTicket(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/status
func (h *Handler) SetTicketStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Status string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.SetTicketStatus(r.Context(), id, req.Status)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/priority
func (h *Handler) SetTicketPriority(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Priority string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.SetTicketPriority(r.Context(), id, req.Priority)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/close
func (h *Handler) CloseTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ ClosedBy, Reason string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.CloseTicket(r.Context(), port.CloseTicketInput{TicketID: id, ClosedBy: req.ClosedBy, Reason: req.Reason})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /api/v1/support/tickets/{id}/reopen
func (h *Handler) ReopenTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.svc.ReopenTicket(r.Context(), id)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ===== Messages =====

// POST /api/v1/support/tickets/{id}/messages
func (h *Handler) ReplyToTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")
	var req struct{ SenderType, SenderID, Body string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.ReplyToTicket(r.Context(), port.ReplyTicketInput{
		TicketID: ticketID, SenderType: req.SenderType, SenderID: req.SenderID, Body: req.Body,
	})
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// GET /api/v1/support/tickets/{id}/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")
	page := parsePage(r)
	result, err := h.svc.ListMessages(r.Context(), ticketID, page)
	if err != nil {
		writeErr(w, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PUT /api/v1/support/messages/{id}
func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct{ Body string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, h.logger, domain.ErrInvalidInput)
		return
	}
	result, err := h.svc.EditMessage(r.Context(), id, req.Body)
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

	mux.Handle("POST /api/v1/support/tickets", authMW(http.HandlerFunc(h.CreateTicket)))
	mux.Handle("GET /api/v1/support/tickets", authMW(http.HandlerFunc(h.ListTickets)))
	mux.Handle("GET /api/v1/support/tickets/{id}", authMW(http.HandlerFunc(h.GetTicket)))
	mux.Handle("POST /api/v1/support/tickets/{id}/assign", authMW(http.HandlerFunc(h.AssignTicket)))
	mux.Handle("POST /api/v1/support/tickets/{id}/unassign", authMW(http.HandlerFunc(h.UnassignTicket)))
	mux.Handle("POST /api/v1/support/tickets/{id}/status", authMW(http.HandlerFunc(h.SetTicketStatus)))
	mux.Handle("POST /api/v1/support/tickets/{id}/priority", authMW(http.HandlerFunc(h.SetTicketPriority)))
	mux.Handle("POST /api/v1/support/tickets/{id}/close", authMW(http.HandlerFunc(h.CloseTicket)))
	mux.Handle("POST /api/v1/support/tickets/{id}/reopen", authMW(http.HandlerFunc(h.ReopenTicket)))
	mux.Handle("POST /api/v1/support/tickets/{id}/messages", authMW(http.HandlerFunc(h.ReplyToTicket)))
	mux.Handle("GET /api/v1/support/tickets/{id}/messages", authMW(http.HandlerFunc(h.ListMessages)))
	mux.Handle("PUT /api/v1/support/messages/{id}", authMW(http.HandlerFunc(h.EditMessage)))
}
