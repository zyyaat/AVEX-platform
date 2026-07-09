// Package transport/http: system HTTP transport.
//
// Provides health check endpoints for Kubernetes probes:
//   GET /health       — full system health (all components)
//   GET /health/live  — liveness probe (is the process alive?)
//   GET /health/ready — readiness probe (can we serve traffic?)
//   GET /system/info  — build + runtime info
//   GET /system/modules — list registered modules
package http

import (
	"encoding/json"
	"net/http"

	"avex-backend/internal/modules/system/port"
)

type Handler struct {
	svc port.ServicePort
}

func NewHandler(svc port.ServicePort) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Health(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}
	// Determine HTTP status from overall health
	status := http.StatusOK
	if result.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{"data": result})
}

// GET /health/live
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Liveness(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /health/ready
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Readiness(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error"})
		return
	}
	status := http.StatusOK
	if result.Status != "ready" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, result)
}

// GET /system/info
func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	result := h.svc.Info()
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

// GET /system/modules
func (h *Handler) Modules(w http.ResponseWriter, r *http.Request) {
	result := h.svc.ListModules()
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

// ===== Routes =====
//
// Health check endpoints do NOT require auth — they must be accessible
// by Kubernetes probes and monitoring tools.
// System info + modules endpoints are public for now (no sensitive data).

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort) {
	h := NewHandler(svc)
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /health/live", h.Liveness)
	mux.HandleFunc("GET /health/ready", h.Readiness)
	mux.HandleFunc("GET /system/info", h.Info)
	mux.HandleFunc("GET /system/modules", h.Modules)
}
