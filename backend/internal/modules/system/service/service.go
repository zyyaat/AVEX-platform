// Package service: system service implementation.
package service

import (
	"context"
	"runtime"
	"time"

	"avex-backend/internal/modules/system/domain"
	"avex-backend/internal/modules/system/port"
)

type Service struct {
	info          domain.SystemInfo
	checkers      []port.HealthChecker
	modules       []domain.ModuleInfo
	maintenance   port.MaintenanceChecker
}

var _ port.ServicePort = (*Service)(nil)

// New creates a new system Service.
func New(info domain.SystemInfo, checkers []port.HealthChecker, modules []domain.ModuleInfo, maintenance port.MaintenanceChecker) *Service {
	return &Service{info: info, checkers: checkers, modules: modules, maintenance: maintenance}
}

// ===== Health =====

func (s *Service) Health(ctx context.Context) (*port.SystemHealthDTO, error) {
	components := make([]domain.ComponentHealth, 0, len(s.checkers))
	for _, checker := range s.checkers {
		components = append(components, checker.CheckHealth(ctx))
	}
	health := domain.NewSystemHealth(components, s.info.Uptime(), s.info.Version())
	return port.ToSystemHealthDTO(health), nil
}

// ===== Liveness =====

func (s *Service) Liveness(_ context.Context) (*port.LivenessDTO, error) {
	return &port.LivenessDTO{
		Status: "alive",
		Uptime: s.info.Uptime().String(),
	}, nil
}

// ===== Readiness =====

func (s *Service) Readiness(ctx context.Context) (*port.ReadinessDTO, error) {
	dto := &port.ReadinessDTO{
		Status:      "ready",
		Database:    "unknown",
		Redis:       "unknown",
		Maintenance: false,
	}

	// Check each component
	for _, checker := range s.checkers {
		health := checker.CheckHealth(ctx)
		status := string(health.Status())
		switch checker.Name() {
		case "database":
			dto.Database = status
		case "redis":
			dto.Redis = status
		}
		if health.Status() == domain.HealthStatusUnhealthy {
			dto.Status = "not_ready"
		}
	}

	// Check maintenance mode
	if s.maintenance != nil {
		dto.Maintenance = s.maintenance.IsMaintenanceMode(ctx)
		if dto.Maintenance {
			dto.Status = "maintenance"
		}
	}

	return dto, nil
}

// ===== Info =====

func (s *Service) Info() *port.SystemInfoDTO {
	info := s.info
	// Update go version at runtime if not set
	if info.GoVersion() == "" {
		info = domain.NewSystemInfo(
			info.AppName(), info.Version(), info.BuildDate(), info.GitCommit(),
			runtime.Version(), info.Environment(), info.StartedAt(),
		)
	}
	return port.ToSystemInfoDTO(info)
}

// ===== ListModules =====

func (s *Service) ListModules() []port.ModuleInfoDTO {
	dtos := make([]port.ModuleInfoDTO, 0, len(s.modules))
	for _, m := range s.modules {
		dtos = append(dtos, port.ToModuleInfoDTO(m))
	}
	return dtos
}

// ===== IsMaintenanceMode =====

func (s *Service) IsMaintenanceMode(ctx context.Context) bool {
	if s.maintenance == nil {
		return false
	}
	return s.maintenance.IsMaintenanceMode(ctx)
}

// suppress unused import
var _ = time.Now
