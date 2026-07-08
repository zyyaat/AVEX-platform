// Package http: permissions HTTP transport.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"avex-backend/internal/modules/permissions/domain"
	"avex-backend/internal/modules/permissions/port"
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
	case err == domain.ErrRoleNotFound || err == domain.ErrPermissionNotFound || err == domain.ErrRolePermissionNotFound || err == domain.ErrUserRoleNotFound:
		status = http.StatusNotFound
	case err == domain.ErrRoleAlreadyExists || err == domain.ErrPermissionAlreadyExists || err == domain.ErrRolePermissionAlreadyExists || err == domain.ErrUserRoleAlreadyExists:
		status = http.StatusConflict
	case err == domain.ErrCannotModifySystemRole || err == domain.ErrCannotRemoveLastAdmin:
		status = http.StatusUnprocessableEntity
	case err == domain.ErrInvalidID || err == domain.ErrInvalidInput || err == domain.ErrEmptyRoleName || err == domain.ErrEmptyPermissionName || err == domain.ErrEmptyUserID || err == domain.ErrInvalidPermissionFormat:
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

// ===== Role Endpoints =====

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Description string; IsSystem bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.CreateRole(r.Context(), port.CreateRoleInput{Name: req.Name, Description: req.Description, IsSystem: req.IsSystem})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetRole(r.Context(), r.PathValue("id"))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListRoles(r.Context(), parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteRole(r.Context(), r.PathValue("id")); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ===== Permission Endpoints =====

func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Description string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.CreatePermission(r.Context(), port.CreatePermissionInput{Name: req.Name, Description: req.Description})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	if module := r.URL.Query().Get("module"); module != "" {
		result, err := h.svc.ListPermissionsByModule(r.Context(), module)
		if err != nil { writeErr(w, h.logger, err); return }
		writeJSON(w, http.StatusOK, result)
		return
	}
	result, err := h.svc.ListPermissions(r.Context(), parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListPermissionsByRole(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ListPermissionsByRole(r.Context(), r.PathValue("id"))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Grant/Revoke =====

func (h *Handler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	var req struct{ RoleID, PermissionID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	if err := h.svc.GrantPermission(r.Context(), port.GrantPermissionInput{RoleID: req.RoleID, PermissionID: req.PermissionID}); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "granted"})
}

func (h *Handler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	var req struct{ RoleID, PermissionID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	if err := h.svc.RevokePermission(r.Context(), req.RoleID, req.PermissionID); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// ===== Assign/Unassign =====

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req struct{ UserID, RoleID, AssignedBy string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	result, err := h.svc.AssignRole(r.Context(), port.AssignRoleInput{UserID: req.UserID, RoleID: req.RoleID, AssignedBy: req.AssignedBy})
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) UnassignRole(w http.ResponseWriter, r *http.Request) {
	var req struct{ UserID, RoleID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeErr(w, h.logger, domain.ErrInvalidInput); return }
	if err := h.svc.UnassignRole(r.Context(), req.UserID, req.RoleID); err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "unassigned"})
}

func (h *Handler) ListRolesByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	result, err := h.svc.ListRolesByUser(r.Context(), userID)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListPermissionsByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	result, err := h.svc.ListPermissionsByUser(r.Context(), userID)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListUsersByRole(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("id")
	result, err := h.svc.ListUsersByRole(r.Context(), roleID, parsePage(r))
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Check =====

func (h *Handler) HasPermission(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	permission := r.URL.Query().Get("permission")
	if permission == "" { writeErr(w, h.logger, domain.ErrEmptyPermissionName); return }
	result, err := h.svc.HasPermission(r.Context(), userID, permission)
	if err != nil { writeErr(w, h.logger, err); return }
	writeJSON(w, http.StatusOK, result)
}

// ===== Routes =====

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, logger *slog.Logger, jwtIssuer idp.JWTIssuer) {
	h := NewHandler(svc, logger)
	authMW := idhttp.Auth(jwtIssuer, logger)

	// Roles
	mux.Handle("POST /api/v1/admin/roles", authMW(http.HandlerFunc(h.CreateRole)))
	mux.Handle("GET /api/v1/admin/roles", authMW(http.HandlerFunc(h.ListRoles)))
	mux.Handle("GET /api/v1/admin/roles/{id}", authMW(http.HandlerFunc(h.GetRole)))
	mux.Handle("DELETE /api/v1/admin/roles/{id}", authMW(http.HandlerFunc(h.DeleteRole)))

	// Permissions
	mux.Handle("POST /api/v1/admin/permissions", authMW(http.HandlerFunc(h.CreatePermission)))
	mux.Handle("GET /api/v1/admin/permissions", authMW(http.HandlerFunc(h.ListPermissions)))
	mux.Handle("GET /api/v1/admin/roles/{id}/permissions", authMW(http.HandlerFunc(h.ListPermissionsByRole)))

	// Grant/Revoke
	mux.Handle("POST /api/v1/admin/roles/grant", authMW(http.HandlerFunc(h.GrantPermission)))
	mux.Handle("POST /api/v1/admin/roles/revoke", authMW(http.HandlerFunc(h.RevokePermission)))

	// Assign/Unassign
	mux.Handle("POST /api/v1/admin/roles/assign", authMW(http.HandlerFunc(h.AssignRole)))
	mux.Handle("POST /api/v1/admin/roles/unassign", authMW(http.HandlerFunc(h.UnassignRole)))
	mux.Handle("GET /api/v1/admin/users/{userID}/roles", authMW(http.HandlerFunc(h.ListRolesByUser)))
	mux.Handle("GET /api/v1/admin/users/{userID}/permissions", authMW(http.HandlerFunc(h.ListPermissionsByUser)))
	mux.Handle("GET /api/v1/admin/roles/{id}/users", authMW(http.HandlerFunc(h.ListUsersByRole)))

	// Check
	mux.Handle("GET /api/v1/admin/users/{userID}/check", authMW(http.HandlerFunc(h.HasPermission)))
}
