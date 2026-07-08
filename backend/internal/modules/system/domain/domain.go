// Package domain: System module types + health status.
//
// The System module provides:
//   - Health check endpoints (/health, /live, /ready) for Kubernetes probes
//   - System info (version, build, uptime, modules)
//   - Maintenance mode checks (reads from settings module)
package domain

import (
	"errors"
	"fmt"
	"time"
)

// ===== Errors =====

var ErrInvalidInput = errors.New("invalid input")
var ErrModuleNotRegistered = errors.New("module not registered")

// HealthStatus enumerates the health of a component.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

func (s HealthStatus) IsValid() bool {
	switch s {
	case HealthStatusHealthy, HealthStatusDegraded, HealthStatusUnhealthy:
		return true
	}
	return false
}

// HTTPStatusCode returns the appropriate HTTP status code for the health status.
func (s HealthStatus) HTTPStatusCode() int {
	switch s {
	case HealthStatusHealthy:
		return 200
	case HealthStatusDegraded:
		return 200 // degraded is still "available" but with warnings
	case HealthStatusUnhealthy:
		return 503
	}
	return 503
}

// ComponentHealth represents the health of a single system component.
type ComponentHealth struct {
	name      string
	status    HealthStatus
	message   string
	latency   time.Duration
	checkedAt time.Time
	details   map[string]any
}

// NewComponentHealth creates a new ComponentHealth.
func NewComponentHealth(name string, status HealthStatus, message string, latency time.Duration, details map[string]any) ComponentHealth {
	return ComponentHealth{
		name:      name,
		status:    status,
		message:   message,
		latency:   latency,
		checkedAt: time.Now().UTC(),
		details:   details,
	}
}

func (c ComponentHealth) Name() string          { return c.name }
func (c ComponentHealth) Status() HealthStatus  { return c.status }
func (c ComponentHealth) Message() string       { return c.message }
func (c ComponentHealth) Latency() time.Duration { return c.latency }
func (c ComponentHealth) CheckedAt() time.Time  { return c.checkedAt }
func (c ComponentHealth) Details() map[string]any { return c.details }

// SystemHealth represents the overall system health.
type SystemHealth struct {
	status     HealthStatus
	components []ComponentHealth
	uptime     time.Duration
	checkedAt  time.Time
	version    string
}

// NewSystemHealth creates a new SystemHealth from component results.
// The overall status is the worst of all component statuses.
func NewSystemHealth(components []ComponentHealth, uptime time.Duration, version string) SystemHealth {
	overall := HealthStatusHealthy
	for _, c := range components {
		if c.Status() == HealthStatusUnhealthy {
			overall = HealthStatusUnhealthy
			break
		}
		if c.Status() == HealthStatusDegraded && overall != HealthStatusUnhealthy {
			overall = HealthStatusDegraded
		}
	}
	return SystemHealth{
		status:     overall,
		components: components,
		uptime:     uptime,
		checkedAt:  time.Now().UTC(),
		version:    version,
	}
}

func (h SystemHealth) Status() HealthStatus            { return h.status }
func (h SystemHealth) Components() []ComponentHealth   { return h.components }
func (h SystemHealth) Uptime() time.Duration           { return h.uptime }
func (h SystemHealth) CheckedAt() time.Time            { return h.checkedAt }
func (h SystemHealth) Version() string                 { return h.version }

// IsHealthy reports whether the system is fully healthy (no degraded/unhealthy).
func (h SystemHealth) IsHealthy() bool { return h.status == HealthStatusHealthy }

// IsLive reports whether the system is alive (at least running, even if degraded).
// Used for liveness probes — returns false only if completely unhealthy.
func (h SystemHealth) IsLive() bool { return h.status != HealthStatusUnhealthy }

// IsReady reports whether the system is ready to serve traffic.
// Used for readiness probes — returns false if any critical component is unhealthy.
func (h SystemHealth) IsReady() bool { return h.status != HealthStatusUnhealthy }

// SystemInfo holds build + runtime information about the application.
type SystemInfo struct {
	appName     string
	version     string
	buildDate   string
	gitCommit   string
	goVersion   string
	environment string
	startedAt   time.Time
}

// NewSystemInfo creates a new SystemInfo.
func NewSystemInfo(appName, version, buildDate, gitCommit, goVersion, environment string, startedAt time.Time) SystemInfo {
	return SystemInfo{
		appName: appName, version: version, buildDate: buildDate,
		gitCommit: gitCommit, goVersion: goVersion, environment: environment,
		startedAt: startedAt,
	}
}

func (s SystemInfo) AppName() string     { return s.appName }
func (s SystemInfo) Version() string     { return s.version }
func (s SystemInfo) BuildDate() string   { return s.buildDate }
func (s SystemInfo) GitCommit() string   { return s.gitCommit }
func (s SystemInfo) GoVersion() string   { return s.goVersion }
func (s SystemInfo) Environment() string { return s.environment }
func (s SystemInfo) StartedAt() time.Time { return s.startedAt }

// Uptime returns the duration since the system started.
func (s SystemInfo) Uptime() time.Duration {
	return time.Since(s.startedAt)
}

// ModuleInfo holds information about a registered module.
type ModuleInfo struct {
	name        string
	description string
	version     int
	enabled     bool
}

func NewModuleInfo(name, description string, version int, enabled bool) ModuleInfo {
	return ModuleInfo{name: name, description: description, version: version, enabled: enabled}
}

func (m ModuleInfo) Name() string        { return m.name }
func (m ModuleInfo) Description() string { return m.description }
func (m ModuleInfo) Version() int        { return m.version }
func (m ModuleInfo) Enabled() bool       { return m.enabled }

// suppress unused import
var _ = fmt.Sprintf
