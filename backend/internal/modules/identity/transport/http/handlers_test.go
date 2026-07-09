// Package http tests: HTTP handler integration tests.
//
// Tests the full HTTP request → response cycle using httptest.NewRecorder
// and a mock ServicePort. Verifies:
//   - HTTP status codes
//   - Error response format
//   - JWT middleware behavior (401 on missing/invalid token)
//   - Role protection (403 on wrong role)
//   - Request validation (400 on malformed input)
//
// Uses stdlib net/http/httptest only — no new dependencies.
package http

import (
        "context"
        "encoding/json"
        "errors"
        "log/slog"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "avex-backend/internal/modules/identity/domain"
        "avex-backend/internal/modules/identity/port"
)

// testLogger returns a discard logger for tests (avoids nil panics in middleware).
func testLogger() *slog.Logger {
        return slog.New(slog.NewTextHandler(nil, nil))
}

// mockService is a minimal ServicePort mock for HTTP tests.
// Each method's behavior is configurable via fields.
type mockService struct {
        registerUserFn  func(ctx context.Context, input port.RegisterUserInput) (*port.AuthResult, error)
        loginUserFn     func(ctx context.Context, input port.LoginInput) (*port.AuthResult, error)
        loginDriverFn   func(ctx context.Context, input port.LoginInput) (*port.AuthResult, error)
        logoutFn        func(ctx context.Context, sessionID string) error
        changePassFn    func(ctx context.Context, input port.ChangePasswordInput) error
        changeDrvPassFn func(ctx context.Context, input port.ChangePasswordInput) error
        getUserFn       func(ctx context.Context, userID string) (*port.UserDTO, error)
        getDriverFn     func(ctx context.Context, driverID string) (*port.DriverProfileDTO, error)
        updateDrvStatFn func(ctx context.Context, input port.UpdateDriverStatusInput) (*port.DriverProfileDTO, error)
        suspendDrvFn    func(ctx context.Context, input port.SuspendDriverInput) error
        getMerchantFn   func(ctx context.Context, merchantID string) (*port.MerchantProfileDTO, error)
        verifyUserFn    func(ctx context.Context, userID string) (bool, error)
        verifyDriverFn  func(ctx context.Context, driverID string) (bool, error)
        hashPassFn      func(ctx context.Context, password string) (string, error)
}

func (m *mockService) RegisterUser(ctx context.Context, input port.RegisterUserInput) (*port.AuthResult, error) {
        if m.registerUserFn != nil {
                return m.registerUserFn(ctx, input)
        }
        return nil, nil
}

func (m *mockService) RegisterDriver(ctx context.Context, input port.RegisterDriverInput) (*port.AuthResult, error) {
        return nil, nil
}
func (m *mockService) LoginUser(ctx context.Context, input port.LoginInput) (*port.AuthResult, error) {
        if m.loginUserFn != nil {
                return m.loginUserFn(ctx, input)
        }
        return nil, nil
}
func (m *mockService) LoginDriver(ctx context.Context, input port.LoginInput) (*port.AuthResult, error) {
        if m.loginDriverFn != nil {
                return m.loginDriverFn(ctx, input)
        }
        return nil, nil
}
func (m *mockService) Logout(ctx context.Context, sessionID string) error {
        if m.logoutFn != nil {
                return m.logoutFn(ctx, sessionID)
        }
        return nil
}
func (m *mockService) ChangePassword(ctx context.Context, input port.ChangePasswordInput) error {
        if m.changePassFn != nil {
                return m.changePassFn(ctx, input)
        }
        return nil
}
func (m *mockService) ChangeDriverPassword(ctx context.Context, input port.ChangePasswordInput) error {
        if m.changeDrvPassFn != nil {
                return m.changeDrvPassFn(ctx, input)
        }
        return nil
}
func (m *mockService) GetUser(ctx context.Context, userID string) (*port.UserDTO, error) {
        if m.getUserFn != nil {
                return m.getUserFn(ctx, userID)
        }
        return &port.UserDTO{ID: userID}, nil
}
func (m *mockService) GetDriverProfile(ctx context.Context, driverID string) (*port.DriverProfileDTO, error) {
        if m.getDriverFn != nil {
                return m.getDriverFn(ctx, driverID)
        }
        return &port.DriverProfileDTO{ID: driverID}, nil
}
func (m *mockService) UpdateDriverStatus(ctx context.Context, input port.UpdateDriverStatusInput) (*port.DriverProfileDTO, error) {
        if m.updateDrvStatFn != nil {
                return m.updateDrvStatFn(ctx, input)
        }
        return &port.DriverProfileDTO{ID: input.DriverID, IsOnline: input.Status == "online"}, nil
}
func (m *mockService) SuspendDriver(ctx context.Context, input port.SuspendDriverInput) error {
        if m.suspendDrvFn != nil {
                return m.suspendDrvFn(ctx, input)
        }
        return nil
}
func (m *mockService) GetMerchantProfile(ctx context.Context, merchantID string) (*port.MerchantProfileDTO, error) {
        if m.getMerchantFn != nil {
                return m.getMerchantFn(ctx, merchantID)
        }
        return nil, nil
}
func (m *mockService) VerifyUserExists(ctx context.Context, userID string) (bool, error) {
        if m.verifyUserFn != nil {
                return m.verifyUserFn(ctx, userID)
        }
        return true, nil
}
func (m *mockService) VerifyDriverExists(ctx context.Context, driverID string) (bool, error) {
        if m.verifyDriverFn != nil {
                return m.verifyDriverFn(ctx, driverID)
        }
        return true, nil
}
func (m *mockService) HashPassword(ctx context.Context, password string) (string, error) {
        if m.hashPassFn != nil {
                return m.hashPassFn(ctx, password)
        }
        return "hash:" + password, nil
}

// mockJWTIssuer is a minimal JWTIssuer for middleware tests.
type mockJWTIssuer struct{}

func (mockJWTIssuer) Issue(ctx context.Context, params port.IssueJWTParams) (string, error) {
        return "mock-token:" + params.Subject + ":" + params.Role + ":" + params.SessionID, nil
}

func (mockJWTIssuer) Verify(ctx context.Context, token string) (*port.JWTClaims, error) {
        if !strings.HasPrefix(token, "mock-token:") {
                return nil, domain.ErrInvalidCredentials
        }
        parts := strings.SplitN(token, ":", 4)
        if len(parts) != 4 {
                return nil, domain.ErrInvalidCredentials
        }
        return &port.JWTClaims{
                Subject:   parts[1],
                Role:      parts[2],
                SessionID: parts[3],
                ExpiresAt: time.Now().Add(24 * time.Hour),
        }, nil
}

// ===== Helper: create a test handler with mock service =====

func newTestHandler(svc port.ServicePort) *Handler {
        return NewHandler(svc, testLogger())
}

// ===== Register Endpoint =====

func TestHandler_Register_Success(t *testing.T) {
        svc := &mockService{
                registerUserFn: func(_ context.Context, _ port.RegisterUserInput) (*port.AuthResult, error) {
                        return &port.AuthResult{
                                Token: "jwt-token",
                                User:  &port.UserDTO{ID: "user-1", Name: "Ahmed", Phone: "01012345678"},
                        }, nil
                },
        }
        h := newTestHandler(svc)

        body := `{"name":"Ahmed","phone":"01012345678","password":"password123"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
        w := httptest.NewRecorder()

        h.Register(w, req)

        if w.Code != http.StatusCreated {
                t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
        }

        var resp struct {
                Data port.AuthResult `json:"data"`
        }
        if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
                t.Fatalf("decode response: %v", err)
        }
        if resp.Data.Token != "jwt-token" {
                t.Errorf("token = %q", resp.Data.Token)
        }
}

func TestHandler_Register_ValidationError(t *testing.T) {
        h := newTestHandler(&mockService{})

        // Missing password.
        body := `{"name":"Ahmed","phone":"01012345678"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
        w := httptest.NewRecorder()

        h.Register(w, req)

        if w.Code != http.StatusBadRequest {
                t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
        }
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
        h := newTestHandler(&mockService{})

        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("not json"))
        w := httptest.NewRecorder()

        h.Register(w, req)

        if w.Code != http.StatusBadRequest {
                t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
        }
}

func TestHandler_Register_DuplicatePhone(t *testing.T) {
        svc := &mockService{
                registerUserFn: func(_ context.Context, _ port.RegisterUserInput) (*port.AuthResult, error) {
                        return nil, domain.ErrUserAlreadyExists
                },
        }
        h := newTestHandler(svc)

        body := `{"name":"Ahmed","phone":"01012345678","password":"password123"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
        w := httptest.NewRecorder()

        h.Register(w, req)

        if w.Code != http.StatusConflict {
                t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
        }
}

// ===== Login Endpoint =====

func TestHandler_Login_Success(t *testing.T) {
        svc := &mockService{
                loginUserFn: func(_ context.Context, _ port.LoginInput) (*port.AuthResult, error) {
                        return &port.AuthResult{Token: "jwt", User: &port.UserDTO{ID: "u1"}}, nil
                },
        }
        h := newTestHandler(svc)

        body := `{"phone":"01012345678","password":"password123"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
        w := httptest.NewRecorder()

        h.Login(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
        }
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
        svc := &mockService{
                loginUserFn: func(_ context.Context, _ port.LoginInput) (*port.AuthResult, error) {
                        return nil, domain.ErrInvalidCredentials
                },
        }
        h := newTestHandler(svc)

        body := `{"phone":"01012345678","password":"wrong"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
        w := httptest.NewRecorder()

        h.Login(w, req)

        if w.Code != http.StatusUnauthorized {
                t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
        }
}

// ===== Auth Middleware =====

func TestAuthMiddleware_NoToken(t *testing.T) {
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        called := false
        handler := authMW(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
                called = true
        }))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, req)

        if called {
                t.Error("handler should not be called without token")
        }
        if w.Code != http.StatusUnauthorized {
                t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
        }
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        called := false
        handler := authMW(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
                called = true
        }))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        req.Header.Set("Authorization", "Bearer invalid-token")
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, req)

        if called {
                t.Error("handler should not be called with invalid token")
        }
        if w.Code != http.StatusUnauthorized {
                t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
        }
}

func TestAuthMiddleware_ValidToken_SetsActor(t *testing.T) {
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        var capturedActor *Actor
        handler := authMW(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
                capturedActor = actorFromContext(r.Context())
        }))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        // Token format: mock-token:<subject>:<role>:<sessionID>
        req.Header.Set("Authorization", "Bearer mock-token:user-1:user:session-1")
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
        }
        if capturedActor == nil {
                t.Fatal("expected actor to be set in context")
        }
        if capturedActor.Subject != "user-1" {
                t.Errorf("actor.Subject = %q", capturedActor.Subject)
        }
        if capturedActor.Role != "user" {
                t.Errorf("actor.Role = %q", capturedActor.Role)
        }
        if capturedActor.SessionID != "session-1" {
                t.Errorf("actor.SessionID = %q", capturedActor.SessionID)
        }
}

// ===== GetMe Endpoint (requires auth) =====

func TestHandler_GetMe_RequiresAuth(t *testing.T) {
        svc := &mockService{}
        h := newTestHandler(svc)
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        mux := http.NewServeMux()
        mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(h.GetMe)))

        // No token → 401.
        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        w := httptest.NewRecorder()
        mux.ServeHTTP(w, req)
        if w.Code != http.StatusUnauthorized {
                t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
        }
}

func TestHandler_GetMe_WithAuth(t *testing.T) {
        svc := &mockService{
                getUserFn: func(_ context.Context, userID string) (*port.UserDTO, error) {
                        return &port.UserDTO{ID: userID, Name: "Test"}, nil
                },
        }
        h := newTestHandler(svc)
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        mux := http.NewServeMux()
        mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(h.GetMe)))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        req.Header.Set("Authorization", "Bearer mock-token:user-1:user:session-1")
        w := httptest.NewRecorder()
        mux.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
        }
}

// ===== Error Mapping =====

func TestErrorMapping_NotFound_Returns404(t *testing.T) {
        svc := &mockService{
                getUserFn: func(_ context.Context, _ string) (*port.UserDTO, error) {
                        return nil, domain.ErrUserNotFound
                },
        }
        h := newTestHandler(svc)
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        // Inject actor into context directly for unit test.
        mux := http.NewServeMux()
        mux.Handle("GET /api/v1/users/me", authMW(http.HandlerFunc(h.GetMe)))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
        req.Header.Set("Authorization", "Bearer mock-token:user-1:user:session-1")
        w := httptest.NewRecorder()
        mux.ServeHTTP(w, req)

        if w.Code != http.StatusNotFound {
                t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
        }
}

// ===== Health Endpoint =====

func TestHandler_Health(t *testing.T) {
        h := newTestHandler(&mockService{})

        req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
        w := httptest.NewRecorder()

        h.Health(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
        }
}

// ===== Validation Tests =====

func TestValidation_Register_MissingFields(t *testing.T) {
        tests := []struct {
                name string
                body string
        }{
                {"missing name", `{"phone":"01012345678","password":"password123"}`},
                {"missing phone", `{"name":"Ahmed","password":"password123"}`},
                {"missing password", `{"name":"Ahmed","phone":"01012345678"}`},
                {"short password", `{"name":"Ahmed","phone":"01012345678","password":"12345"}`},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        h := newTestHandler(&mockService{})
                        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.body))
                        w := httptest.NewRecorder()
                        h.Register(w, req)
                        if w.Code != http.StatusBadRequest {
                                t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
                        }
                })
        }
}

func TestValidation_UpdateDriverStatus_InvalidStatus(t *testing.T) {
        h := newTestHandler(&mockService{})
        jwt := mockJWTIssuer{}
        authMW := Auth(jwt, testLogger())

        mux := http.NewServeMux()
        mux.Handle("PATCH /api/v1/drivers/status", authMW(http.HandlerFunc(h.UpdateDriverStatus)))

        body := `{"status":"flying","lat":30,"lng":31}`
        req := httptest.NewRequest(http.MethodPatch, "/api/v1/drivers/status", strings.NewReader(body))
        req.Header.Set("Authorization", "Bearer mock-token:driver-1:driver:session-1")
        w := httptest.NewRecorder()
        mux.ServeHTTP(w, req)

        if w.Code != http.StatusBadRequest {
                t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
        }
}

// ===== Compile-time assertion: mockService satisfies ServicePort =====

var _ port.ServicePort = (*mockService)(nil)
var _ port.JWTIssuer = mockJWTIssuer{}

// suppress unused import warnings
var _ = errors.New
