// Package identity integration tests: full HTTP E2E test.
//
// This test uses the E2E HTTP server set up in setup_integration_test.go
// to test the full request → response cycle against real PostgreSQL + Redis.
//
//go:build integration

package identity_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// doRequest makes an HTTP request to the E2E server and returns the status + body.
func doRequest(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, e2eServer.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func extractToken(t *testing.T, body map[string]any) string {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'data' in response, got: %v", body)
	}
	token, ok := data["token"].(string)
	if !ok {
		t.Fatalf("expected 'token' in data, got: %v", data)
	}
	return token
}

func extractUserID(t *testing.T, body map[string]any) string {
	t.Helper()
	data := body["data"].(map[string]any)
	user := data["user"].(map[string]any)
	return user["id"].(string)
}

// ===== E2E Tests =====

func TestE2E_FullUserFlow(t *testing.T) {
	cleanupIntegTables(t)

	// 1. Register.
	status, body := doRequest(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name":     "E2E Test User",
		"phone":    "01012345678",
		"password": "password123",
	})
	if status != http.StatusCreated {
		t.Fatalf("Register: status = %d, want %d, body = %v", status, http.StatusCreated, body)
	}
	token := extractToken(t, body)
	userID := extractUserID(t, body)
	t.Logf("Register OK: userID=%s", userID)

	// 2. GetMe with token.
	status, body = doRequest(t, "GET", "/api/v1/users/me", token, nil)
	if status != http.StatusOK {
		t.Fatalf("GetMe: status = %d, want %d", status, http.StatusOK)
	}
	data := body["data"].(map[string]any)
	if data["name"] != "E2E Test User" {
		t.Errorf("GetMe name = %v", data["name"])
	}
	t.Log("GetMe OK")

	// 3. Login (get a fresh token).
	status, body = doRequest(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"phone":    "01012345678",
		"password": "password123",
	})
	if status != http.StatusOK {
		t.Fatalf("Login: status = %d, want %d", status, http.StatusOK)
	}
	loginToken := extractToken(t, body)
	t.Log("Login OK")

	// 4. Change password.
	status, _ = doRequest(t, "POST", "/api/v1/auth/change-password", loginToken, map[string]any{
		"old_password": "password123",
		"new_password": "newpassword456",
	})
	if status != http.StatusOK {
		t.Fatalf("ChangePassword: status = %d, want %d", status, http.StatusOK)
	}
	t.Log("ChangePassword OK")

	// 5. Login with old password → should fail.
	status, _ = doRequest(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"phone":    "01012345678",
		"password": "password123",
	})
	if status != http.StatusUnauthorized {
		t.Errorf("Login with old password: status = %d, want %d", status, http.StatusUnauthorized)
	}
	t.Log("Old password rejected OK")

	// 6. Login with new password → should succeed.
	status, body = doRequest(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"phone":    "01012345678",
		"password": "newpassword456",
	})
	if status != http.StatusOK {
		t.Fatalf("Login with new password: status = %d, want %d", status, http.StatusOK)
	}
	newToken := extractToken(t, body)
	t.Log("New password works OK")

	// 7. Logout.
	status, _ = doRequest(t, "POST", "/api/v1/auth/logout", newToken, nil)
	if status != http.StatusOK {
		t.Fatalf("Logout: status = %d, want %d", status, http.StatusOK)
	}
	t.Log("Logout OK")
}

func TestE2E_Register_DuplicatePhone(t *testing.T) {
	cleanupIntegTables(t)

	status, _ := doRequest(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "First", "phone": "01012345678", "password": "password123",
	})
	if status != http.StatusCreated {
		t.Fatalf("first Register: status = %d", status)
	}

	status, body := doRequest(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Second", "phone": "01012345678", "password": "password456",
	})
	if status != http.StatusConflict {
		t.Errorf("duplicate Register: status = %d, want %d", status, http.StatusConflict)
	}
	errBody := body["error"].(map[string]any)
	if errBody["code"] != "user_already_exists" {
		t.Errorf("error code = %v, want 'user_already_exists'", errBody["code"])
	}
}

func TestE2E_Register_ValidationError(t *testing.T) {
	cleanupIntegTables(t)

	status, body := doRequest(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Test", "phone": "01012345678",
	})
	if status != http.StatusBadRequest {
		t.Errorf("missing password: status = %d, want %d", status, http.StatusBadRequest)
	}
	errBody := body["error"].(map[string]any)
	if errBody["code"] != "validation_error" {
		t.Errorf("error code = %v, want 'validation_error'", errBody["code"])
	}
}

func TestE2E_GetMe_Unauthorized(t *testing.T) {
	cleanupIntegTables(t)

	status, body := doRequest(t, "GET", "/api/v1/users/me", "", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("GetMe without token: status = %d, want %d", status, http.StatusUnauthorized)
	}
	errBody := body["error"].(map[string]any)
	if errBody["code"] != "unauthorized" {
		t.Errorf("error code = %v, want 'unauthorized'", errBody["code"])
	}
}

func TestE2E_GetMe_InvalidToken(t *testing.T) {
	cleanupIntegTables(t)

	status, body := doRequest(t, "GET", "/api/v1/users/me", "invalid-token", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("GetMe with invalid token: status = %d, want %d", status, http.StatusUnauthorized)
	}
	errBody := body["error"].(map[string]any)
	if errBody["code"] != "unauthorized" {
		t.Errorf("error code = %v, want 'unauthorized'", errBody["code"])
	}
}

func TestE2E_HealthCheck(t *testing.T) {
	status, body := doRequest(t, "GET", "/healthz", "", nil)
	if status != http.StatusOK {
		t.Fatalf("Health: status = %d, want %d", status, http.StatusOK)
	}
	data := body["data"].(map[string]any)
	if data["status"] != "ok" {
		t.Errorf("status = %v, want 'ok'", data["status"])
	}
}

func TestE2E_Login_WrongPassword(t *testing.T) {
	cleanupIntegTables(t)

	_, _ = doRequest(t, "POST", "/api/v1/auth/register", "", map[string]any{
		"name": "Test", "phone": "01012345678", "password": "password123",
	})

	status, body := doRequest(t, "POST", "/api/v1/auth/login", "", map[string]any{
		"phone": "01012345678", "password": "wrong",
	})
	if status != http.StatusUnauthorized {
		t.Errorf("Login wrong password: status = %d, want %d", status, http.StatusUnauthorized)
	}
	errBody := body["error"].(map[string]any)
	if errBody["code"] != "invalid_credentials" {
		t.Errorf("error code = %v, want 'invalid_credentials'", errBody["code"])
	}
	// Verify no user enumeration.
	if errBody["message"] != "invalid phone or password" {
		t.Errorf("error message = %v, should not reveal which is wrong", errBody["message"])
	}
}
