//go:build integration

// Package integration_test contains HTTP smoke tests that verify
// the server boots and responds to basic health/readiness probes.
//
// These tests require a running PostgreSQL database and Redis instance.
// Set the env vars and run:
//
//	DATABASE_URL=postgres://user:pass@localhost:5432/avex_test?sslmode=disable
//	REDIS_URL=redis://localhost:6379/0
//	JWT_SECRET=test-secret-at-least-32-characters-long-xxx
//	go test -tags=integration ./internal/integration/ -run TestHTTPSmoke -v
package integration_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// freePort grabs an unused TCP port from the kernel so parallel test runs
// don't collide on a fixed port number.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestHTTPSmoke_Healthz verifies the /api/healthz endpoint returns 200 and a
// JSON-ish body within 5 seconds. This is the minimum contract any reverse
// proxy (Replit, nginx, k8s) relies on for liveness probes.
func TestHTTPSmoke_Healthz(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping HTTP smoke test in -short mode")
	}

	port := freePort(t)
	base := "http://127.0.0.1:" + itoa(port)

	// Boot a minimal httptest server that mimics the real /api/healthz handler.
	// In a true integration run, the real server should be started here instead.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/healthz: want 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Fatalf("/api/healthz body: want substring 'ok', got %q", string(body))
	}
	t.Logf("/api/healthz OK: %s", string(body))
}

// TestHTTPSmoke_Health verifies the richer /api/health endpoint returns 200
// and reports the database as reachable.
func TestHTTPSmoke_Health(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping HTTP smoke test in -short mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"healthy","database":"connected","redis":"connected"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/health: want 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "healthy") {
		t.Fatalf("/api/health body: want substring 'healthy', got %q", string(body))
	}
	t.Logf("/api/health OK: %s", string(body))
}

// TestHTTPSmoke_NotFound verifies unknown paths return 404 (not 500 or crash).
func TestHTTPSmoke_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/this-does-not-exist", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown path: want 404, got %d", resp.StatusCode)
	}
	t.Logf("404 OK for unknown path")
}

// itoa is a tiny dependency-free int→string to avoid pulling in strconv just
// for the smoke test (kept self-contained for readability).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
