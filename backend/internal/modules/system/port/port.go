// Package port: interfaces + DTOs for the system module.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/system/domain"
)

// ===== HealthChecker =====
//
// HealthChecker is implemented by each infrastructure component (database,
// Redis, etc.) to provide its health status. The system service calls all
// registered checkers to build the overall health picture.

type HealthChecker interface {
	CheckHealth(ctx context.Context) domain.ComponentHealth
	Name() string
}

// ===== ServicePort =====

type ServicePort interface {
	// Health returns the full system health (all components).
	Health(ctx context.Context) (*SystemHealthDTO, error)

	// Liveness returns a simple liveness check (is the process alive?).
	// Used for Kubernetes liveness probes.
	Liveness(ctx context.Context) (*LivenessDTO, error)

	// Readiness returns a readiness check (can we serve traffic?).
	// Used for Kubernetes readiness probes.
	Readiness(ctx context.Context) (*ReadinessDTO, error)

	// Info returns system build + runtime information.
	Info() *SystemInfoDTO

	// ListModules returns all registered modules.
	ListModules() []ModuleInfoDTO

	// IsMaintenanceMode checks if the system is in maintenance mode
	// (reads from the settings module via the MaintenanceChecker interface).
	IsMaintenanceMode(ctx context.Context) bool
}

// MaintenanceChecker checks if the system is in maintenance mode.
// Implemented by the settings module (or a stub).
type MaintenanceChecker interface {
	IsMaintenanceMode(ctx context.Context) bool
}

// ===== DTOs =====

type ComponentHealthDTO struct {
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Message   string         `json:"message,omitempty"`
	LatencyMs int64          `json:"latency_ms"`
	CheckedAt time.Time      `json:"checked_at"`
	Details   map[string]any `json:"details,omitempty"`
}

type SystemHealthDTO struct {
	Status     string                `json:"status"`
	Version    string                `json:"version"`
	UptimeSecs int64                 `json:"uptime_seconds"`
	CheckedAt  time.Time             `json:"checked_at"`
	Components []ComponentHealthDTO  `json:"components"`
}

type LivenessDTO struct {
	Status   string `json:"status"`
	Uptime   string `json:"uptime"`
}

type ReadinessDTO struct {
	Status       string `json:"status"`
	Database     string `json:"database"`
	Redis        string `json:"redis"`
	Maintenance  bool   `json:"maintenance_mode"`
}

type SystemInfoDTO struct {
	AppName     string    `json:"app_name"`
	Version     string    `json:"version"`
	BuildDate   string    `json:"build_date"`
	GitCommit   string    `json:"git_commit"`
	GoVersion   string    `json:"go_version"`
	Environment string    `json:"environment"`
	StartedAt   time.Time `json:"started_at"`
	Uptime      string    `json:"uptime"`
}

type ModuleInfoDTO struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     int    `json:"version"`
	Enabled     bool   `json:"enabled"`
}

// ===== Mappers =====

func ToComponentHealthDTO(c domain.ComponentHealth) ComponentHealthDTO {
	return ComponentHealthDTO{
		Name: c.Name(), Status: string(c.Status()), Message: c.Message(),
		LatencyMs: c.Latency().Milliseconds(), CheckedAt: c.CheckedAt(), Details: c.Details(),
	}
}

func ToSystemHealthDTO(h domain.SystemHealth) *SystemHealthDTO {
	components := make([]ComponentHealthDTO, 0, len(h.Components()))
	for _, c := range h.Components() {
		components = append(components, ToComponentHealthDTO(c))
	}
	return &SystemHealthDTO{
		Status: string(h.Status()), Version: h.Version(),
		UptimeSecs: int64(h.Uptime().Seconds()), CheckedAt: h.CheckedAt(),
		Components: components,
	}
}

func ToSystemInfoDTO(s domain.SystemInfo) *SystemInfoDTO {
	return &SystemInfoDTO{
		AppName: s.AppName(), Version: s.Version(), BuildDate: s.BuildDate(),
		GitCommit: s.GitCommit(), GoVersion: s.GoVersion(), Environment: s.Environment(),
		StartedAt: s.StartedAt(), Uptime: s.Uptime().String(),
	}
}

func ToModuleInfoDTO(m domain.ModuleInfo) ModuleInfoDTO {
	return ModuleInfoDTO{Name: m.Name(), Description: m.Description(), Version: m.Version(), Enabled: m.Enabled()}
}
