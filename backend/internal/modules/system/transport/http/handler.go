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
        "fmt"
        "net/http"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/system/port"
)

type Handler struct {
        svc  port.ServicePort
        pool *pgxpool.Pool
}

func NewHandler(svc port.ServicePort, pool *pgxpool.Pool) *Handler {
        return &Handler{svc: svc, pool: pool}
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

func RegisterRoutes(mux *http.ServeMux, svc port.ServicePort, pool *pgxpool.Pool) {
        h := NewHandler(svc, pool)
        mux.HandleFunc("GET /health", h.Health)
        mux.HandleFunc("GET /health/live", h.Liveness)
        mux.HandleFunc("GET /health/ready", h.Readiness)
        mux.HandleFunc("GET /system/info", h.Info)
        mux.HandleFunc("GET /system/modules", h.Modules)

        // Zones management (admin only — no auth middleware for now, same as catalog)
        mux.HandleFunc("GET /api/v1/admin/zones", h.ListZones)
        mux.HandleFunc("POST /api/v1/admin/zones", h.CreateZone)
        mux.HandleFunc("PATCH /api/v1/admin/zones/{id}", h.UpdateZone)
        mux.HandleFunc("DELETE /api/v1/admin/zones/{id}", h.DeleteZone)
}

// ===== Zones Management =====

// ListZones handles GET /api/v1/admin/zones
func (h *Handler) ListZones(w http.ResponseWriter, r *http.Request) {
        rows, err := h.pool.Query(r.Context(), `
                SELECT id, name, name_ar, center_lat, center_lng, radius_m, polygon_geojson, is_active, created_at, updated_at
                FROM system.zones ORDER BY created_at DESC
        `)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        defer rows.Close()

        zones := []map[string]any{}
        for rows.Next() {
                var z map[string]any = map[string]any{}
                var nameAr, polygonGeojson *string
                if err := rows.Scan(&z["id"], &z["name"], &nameAr, &z["center_lat"], &z["center_lng"], &z["radius_m"], &polygonGeojson, &z["is_active"], &z["created_at"], &z["updated_at"]); err != nil {
                        continue
                }
                if nameAr != nil {
                        z["name_ar"] = *nameAr
                }
                if polygonGeojson != nil {
                        z["polygon_geojson"] = *polygonGeojson
                }
                zones = append(zones, z)
        }
        writeJSON(w, http.StatusOK, zones)
}

// CreateZone handles POST /api/v1/admin/zones
func (h *Handler) CreateZone(w http.ResponseWriter, r *http.Request) {
        var req struct {
                ID            string  `json:"id"`
                Name          string  `json:"name"`
                NameAr        string  `json:"name_ar"`
                CenterLat     float64 `json:"center_lat"`
                CenterLng     float64 `json:"center_lng"`
                RadiusM       int     `json:"radius_m"`
                PolygonGeojson string `json:"polygon_geojson"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, fmt.Errorf("invalid input"))
                return
        }
        if req.ID == "" {
                req.ID = "zone-" + fmt.Sprintf("%d", time.Now().Unix())
        }
        if req.RadiusM == 0 {
                req.RadiusM = 3000
        }

        var polygon interface{}
        if req.PolygonGeojson != "" {
                polygon = req.PolygonGeojson
        }

        _, err := h.pool.Exec(r.Context(), `
                INSERT INTO system.zones (id, name, name_ar, center_lat, center_lng, radius_m, polygon_geojson, is_active)
                VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
                ON CONFLICT (id) DO UPDATE SET name = $2, name_ar = $3, center_lat = $4, center_lng = $5, radius_m = $6, polygon_geojson = $7, updated_at = NOW()
        `, req.ID, req.Name, req.NameAr, req.CenterLat, req.CenterLng, req.RadiusM, polygon)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusCreated, map[string]any{
                "id": req.ID, "name": req.Name, "name_ar": req.NameAr,
                "center_lat": req.CenterLat, "center_lng": req.CenterLng,
                "radius_m": req.RadiusM, "status": "created",
        })
}

// UpdateZone handles PATCH /api/v1/admin/zones/{id}
func (h *Handler) UpdateZone(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        var req struct {
                Name          *string  `json:"name"`
                NameAr        *string  `json:"name_ar"`
                CenterLat     *float64 `json:"center_lat"`
                CenterLng     *float64 `json:"center_lng"`
                RadiusM       *int     `json:"radius_m"`
                PolygonGeojson *string `json:"polygon_geojson"`
                IsActive      *bool    `json:"is_active"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeErr(w, h.logger, fmt.Errorf("invalid input"))
                return
        }

        _, err := h.pool.Exec(r.Context(), `
                UPDATE system.zones SET
                        name = COALESCE($2, name),
                        name_ar = COALESCE($3, name_ar),
                        center_lat = COALESCE($4, center_lat),
                        center_lng = COALESCE($5, center_lng),
                        radius_m = COALESCE($6, radius_m),
                        polygon_geojson = COALESCE($7, polygon_geojson),
                        is_active = COALESCE($8, is_active),
                        updated_at = NOW()
                WHERE id = $1
        `, id, req.Name, req.NameAr, req.CenterLat, req.CenterLng, req.RadiusM, req.PolygonGeojson, req.IsActive)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteZone handles DELETE /api/v1/admin/zones/{id}
func (h *Handler) DeleteZone(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        _, err := h.pool.Exec(r.Context(), `DELETE FROM system.zones WHERE id = $1`, id)
        if err != nil {
                writeErr(w, h.logger, err)
                return
        }
        writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
