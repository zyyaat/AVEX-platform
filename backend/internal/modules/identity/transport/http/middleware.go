// Package http middleware: request-scoped middleware for the identity
// HTTP transport layer.
//
// Middleware order (outermost to innermost):
//  1. RequestID     — generates/propagates correlation ID
//  2. Logging       — structured request log with correlation ID
//  3. Recovery      — panic recovery (returns 500)
//  4. CORS          — CORS headers
//  5. Auth (per-route) — extracts JWT, sets actor context
//
// Auth middleware does NOT know about domain entities. It only knows
// about port.JWTClaims (Subject, Role, SessionID) — set by the
// JWTIssuer.Verify call.
package http

import (
        "context"
        "encoding/json"
        "fmt"
        "log/slog"
        "net/http"
        "runtime/debug"
        "strings"
        "time"

        "avex-backend/internal/modules/identity/port"
)

// ===== Context Keys =====

type contextKey string

const (
        ctxKeyCorrelationID contextKey = "correlation_id"
        ctxKeyActor         contextKey = "actor"
)

// Actor represents the authenticated identity making a request.
// Set by the Auth middleware; read by handlers to construct ChangePasswordInput, etc.
type Actor struct {
        Subject   string
        Role      string
        SessionID string
}

// actorFromContext retrieves the actor from the request context.
// Returns nil if not authenticated (e.g. on public routes).
func actorFromContext(ctx context.Context) *Actor {
        a, ok := ctx.Value(ctxKeyActor).(*Actor)
        if !ok {
                return nil
        }
        return a
}

// ActorFromContext is the exported version of actorFromContext.
// Other modules (orders, dispatch) use this to get the authenticated
// actor from the JWT context, instead of reading X-Driver-ID headers.
func ActorFromContext(ctx context.Context) *Actor {
        return actorFromContext(ctx)
}

// correlationIDFromContext retrieves the correlation ID from the context.
func correlationIDFromContext(ctx context.Context) string {
        if v, ok := ctx.Value(ctxKeyCorrelationID).(string); ok {
                return v
        }
        return ""
}

// ===== RequestID Middleware =====

// RequestID generates a correlation ID for each request and stores it
// in the request context. If the client sends an X-Request-Id header,
// it is reused (after basic sanitization).
//
// The correlation ID is also set in the response header so clients can
// reference it when reporting issues.
func RequestID(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                id := r.Header.Get("X-Request-Id")
                if id == "" || len(id) > 64 {
                        // Generate a simple ID. Using timestamp + counter would be simpler,
                        // but for now we use a basic format. The ID generator from deps
                        // is not available here (middleware is a func, not a method).
                        id = fmt.Sprintf("req-%d", time.Now().UnixNano())
                }
                w.Header().Set("X-Request-Id", id)
                ctx := context.WithValue(r.Context(), ctxKeyCorrelationID, id)
                next.ServeHTTP(w, r.WithContext(ctx))
        })
}

// ===== Logging Middleware =====

// Logging logs each request with method, path, status, duration, and
// correlation ID. Uses slog for structured logging.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        start := time.Now()
                        ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

                        next.ServeHTTP(ww, r)

                        duration := time.Since(start)
                        correlationID := correlationIDFromContext(r.Context())

                        logger.Info("http request",
                                "method", r.Method,
                                "path", r.URL.Path,
                                "status", ww.status,
                                "duration_ms", duration.Milliseconds(),
                                "correlation_id", correlationID,
                                "remote_addr", r.RemoteAddr,
                        )
                })
        }
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
        http.ResponseWriter
        status int
}

func (w *statusWriter) WriteHeader(status int) {
        w.status = status
        w.ResponseWriter.WriteHeader(status)
}

// ===== Recovery Middleware =====

// Recovery catches panics, logs them, and returns a 500 response.
// Prevents a single panic from crashing the entire server.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        defer func() {
                                if rec := recover(); rec != nil {
                                        logger.Error("panic recovered",
                                                "panic", rec,
                                                "stack", string(debug.Stack()),
                                                "correlation_id", correlationIDFromContext(r.Context()),
                                        )
                                        w.Header().Set("Content-Type", "application/json")
                                        w.WriteHeader(http.StatusInternalServerError)
                                        _ = json.NewEncoder(w).Encode(errorResponse{Error: errorBody{
                                                Message: "internal server error",
                                                Code:    "internal_error",
                                        }})
                                }
                        }()
                        next.ServeHTTP(w, r)
                })
        }
}

// ===== CORS Middleware =====

// CORS adds CORS headers to responses. Allowed origins are configurable.
// In production, this should be a specific list — not "*".
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
        originSet := make(map[string]bool, len(allowedOrigins))
        for _, o := range allowedOrigins {
                originSet[o] = true
        }

        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        origin := r.Header.Get("Origin")
                        if originSet[origin] || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
                                w.Header().Set("Access-Control-Allow-Origin", origin)
                                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
                                w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-Request-Id")
                                w.Header().Set("Access-Control-Allow-Credentials", "true")
                                w.Header().Set("Access-Control-Max-Age", "3600")
                        }

                        // Handle preflight.
                        if r.Method == http.MethodOptions {
                                w.WriteHeader(http.StatusNoContent)
                                return
                        }
                        next.ServeHTTP(w, r)
                })
        }
}

// ===== Auth Middleware =====

// Auth extracts the JWT from the Authorization header, verifies it via
// the JWTIssuer, and sets the Actor in the request context.
//
// If the token is missing or invalid, it returns 401.
// This middleware is applied only to protected routes — public routes
// (register, login) do not use it.
func Auth(jwtIssuer port.JWTIssuer, logger *slog.Logger) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        // Extract Bearer token.
                        authHeader := r.Header.Get("Authorization")
                        if authHeader == "" {
                                writeAuthError(w, "authorization header is required")
                                return
                        }

                        parts := strings.SplitN(authHeader, " ", 2)
                        if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
                                writeAuthError(w, "authorization header must be Bearer token")
                                return
                        }

                        token := strings.TrimSpace(parts[1])
                        if token == "" {
                                writeAuthError(w, "token is empty")
                                return
                        }

                        // Verify token.
                        claims, err := jwtIssuer.Verify(r.Context(), token)
                        if err != nil {
                                logger.Debug("jwt verification failed",
                                        "error", err,
                                        "correlation_id", correlationIDFromContext(r.Context()),
                                )
                                writeAuthError(w, "invalid or expired token")
                                return
                        }

                        // Set actor in context.
                        actor := &Actor{
                                Subject:   claims.Subject,
                                Role:      claims.Role,
                                SessionID: claims.SessionID,
                        }
                        ctx := context.WithValue(r.Context(), ctxKeyActor, actor)
                        next.ServeHTTP(w, r.WithContext(ctx))
                })
        }
}

// RequireRole wraps Auth and additionally checks that the actor has
// one of the allowed roles. Used for admin-only endpoints.
//
// FIXED: Removed dead code (authHandler variable was assigned but never
// used). The implementation is now clean and straightforward.
func RequireRole(jwtIssuer port.JWTIssuer, logger *slog.Logger, allowedRoles ...string) func(http.Handler) http.Handler {
        auth := Auth(jwtIssuer, logger)
        allowed := make(map[string]bool, len(allowedRoles))
        for _, r := range allowedRoles {
                allowed[r] = true
        }

        // The role-check handler runs AFTER auth sets the actor in context.
        roleCheck := func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        actor := actorFromContext(r.Context())
                        if actor == nil {
                                writeAuthError(w, "authentication required")
                                return
                        }
                        if !allowed[actor.Role] {
                                w.Header().Set("Content-Type", "application/json")
                                w.WriteHeader(http.StatusForbidden)
                                _ = json.NewEncoder(w).Encode(errorResponse{Error: errorBody{
                                        Message: "insufficient permissions",
                                        Code:    "forbidden",
                                }})
                                return
                        }
                        next.ServeHTTP(w, r)
                })
        }

        // Compose: auth first (sets actor), then role check.
        return func(next http.Handler) http.Handler {
                return auth(roleCheck(next))
        }
}

// writeAuthError writes a 401 Unauthorized response.
func writeAuthError(w http.ResponseWriter, message string) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        _ = json.NewEncoder(w).Encode(errorResponse{Error: errorBody{
                Message: message,
                Code:    "unauthorized",
        }})
}
