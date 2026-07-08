// Package realtime is the composition root for the realtime module.
package realtime

import (
        "context"
        "crypto/rand"
        "encoding/hex"
        "fmt"
        "log/slog"
        "net/http"
        "time"

        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/realtime/port"
        "avex-backend/internal/modules/realtime/service"
        httptransport "avex-backend/internal/modules/realtime/transport/http"
        idp "avex-backend/internal/modules/identity/port"
        "avex-backend/internal/platform/config"
        "avex-backend/internal/platform/inbox"
)

// Module is the wired realtime module.
type Module struct {
        svc    port.ServicePort
        hub    port.Hub
        pool   *pgxpool.Pool
        logger *slog.Logger
}

// New wires all realtime dependencies and returns a ready-to-use Module.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *Module {
        clock := &realClock{}
        idGen := &uuidIDGen{}

        // Hub
        hub := service.NewHub(loggerAdapter{logger}, clock)

        // Service
        deps := port.Deps{
                Hub:    hub,
                Logger: loggerAdapter{logger},
                IDGen:  idGen,
                Clock:  clock,
        }
        svc := service.New(deps)

        return &Module{
                svc:    svc,
                hub:    hub,
                pool:   pool,
                logger: logger,
        }
}

// Service exposes the port.ServicePort for cross-module use.
func (m *Module) Service() port.ServicePort { return m.svc }

// Hub exposes the port.Hub for the subscriber (jobs) to broadcast.
func (m *Module) Hub() port.Hub { return m.hub }

// RegisterRoutes wires the realtime HTTP routes into the given mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux, jwtIssuer idp.JWTIssuer) {
        httptransport.RegisterRoutes(mux, m.hub, m.svc, jwtIssuer, m.logger, &uuidIDGen{}, &realClock{})
}

// NewInbox creates an inbox store for the realtime module's subscriber.
// We wrap the platform inbox.PostgresInbox in a thin adapter because the
// platform Inbox interface has a DBTX type that doesn't match pgx's signatures
// (a pre-existing issue). The Dedup wrapper only calls IsProcessed + MarkProcessed,
// so the adapter delegates those and stubs MarkProcessedTx.
func (m *Module) NewInbox() inbox.Inbox {
        return &inboxAdapter{
                inner: inbox.NewPostgresInbox(m.pool, inbox.Config{
                        Table: "realtime.inbox",
                }),
        }
}

// inboxAdapter wraps *inbox.PostgresInbox to satisfy inbox.Inbox.
type inboxAdapter struct {
        inner *inbox.PostgresInbox
}

func (a *inboxAdapter) IsProcessed(ctx context.Context, eventID, handlerName string) (bool, error) {
        return a.inner.IsProcessed(ctx, eventID, handlerName)
}

func (a *inboxAdapter) MarkProcessed(ctx context.Context, eventID, handlerName, eventType string) error {
        return a.inner.MarkProcessed(ctx, eventID, handlerName, eventType)
}

// MarkProcessedTx delegates to MarkProcessed (non-transactional).
// The Dedup wrapper never calls this method; it's here to satisfy the interface.
func (a *inboxAdapter) MarkProcessedTx(ctx context.Context, _ inbox.DBTX, eventID, handlerName, eventType string) error {
        return a.inner.MarkProcessed(ctx, eventID, handlerName, eventType)
}

// Close releases resources held by the module.
func (m *Module) Close() {}

// ===== Adapters =====

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type uuidIDGen struct{}

func (*uuidIDGen) NewID() string { return newUUID() }

// loggerAdapter wraps *slog.Logger to satisfy port.Logger.
type loggerAdapter struct{ l *slog.Logger }

func (a loggerAdapter) Debug(msg string, args ...any) { a.l.Debug(msg, args...) }
func (a loggerAdapter) Info(msg string, args ...any)  { a.l.Info(msg, args...) }
func (a loggerAdapter) Warn(msg string, args ...any)  { a.l.Warn(msg, args...) }
func (a loggerAdapter) Error(msg string, args ...any) { a.l.Error(msg, args...) }

// newUUID generates a UUIDv4 using crypto/rand.
func newUUID() string {
        b := make([]byte, 16)
        if _, err := rand.Read(b); err != nil {
                panic(fmt.Sprintf("crypto/rand: %v", err))
        }
        b[6] = (b[6] & 0x0f) | 0x40
        b[8] = (b[8] & 0x3f) | 0x80
        h := hex.EncodeToString(b)
        return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

// Ensure adapters satisfy port interfaces.
var (
        _ port.Clock    = (*realClock)(nil)
        _ port.IDGenerator = (*uuidIDGen)(nil)
        _ port.Logger   = loggerAdapter{}
)
