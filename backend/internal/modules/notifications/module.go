// Package notifications is the composition root for the notifications module.
package notifications

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"avex-backend/internal/modules/notifications/events"
	"avex-backend/internal/modules/notifications/port"
	"avex-backend/internal/modules/notifications/providers"
	"avex-backend/internal/modules/notifications/repository/postgres"
	"avex-backend/internal/modules/notifications/service"
	httptransport "avex-backend/internal/modules/notifications/transport/http"
	idp "avex-backend/internal/modules/identity/port"
	"avex-backend/internal/platform/config"
	"avex-backend/internal/platform/inbox"
)

type Module struct {
	svc    port.ServicePort
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *Module {
	repos := postgres.NewRepositories()
	repoSet := repos.RepositorySet()

	eventPublisher := events.NewEventPublisher(repoSet, &uuidIDGen{})
	txRunner := &pgxTxRunner{pool: pool}

	pushProvider := providers.NewFCMProvider(logger)
	smsProvider := providers.NewTwilioProvider(logger)
	emailProvider := providers.NewSendGridProvider(logger)

	deps := port.Deps{
		Clock:          &realClock{},
		IDGenerator:    &uuidIDGen{},
		EventPublisher: eventPublisher,
		Logger:         loggerAdapter{logger},
		TxRunner:       txRunner,
		Repos:          repoSet,
		PushProvider:   pushProvider,
		SMSProvider:    smsProvider,
		EmailProvider:  emailProvider,
	}

	svc := service.New(deps, pool)

	return &Module{svc: svc, pool: pool, logger: logger}
}

func (m *Module) Service() port.ServicePort { return m.svc }

func (m *Module) RegisterRoutes(mux *http.ServeMux, jwtIssuer idp.JWTIssuer) {
	httptransport.RegisterRoutes(mux, m.svc, m.logger, jwtIssuer)
}

func (m *Module) NewInbox() inbox.Inbox {
	return &inboxAdapter{
		inner: inbox.NewPostgresInbox(m.pool, inbox.Config{Table: "notifications.inbox"}),
	}
}

func (m *Module) Close() {}

// ===== Adapters =====

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type uuidIDGen struct{}

func (*uuidIDGen) NewID() string { return newUUID() }

type pgxTxRunner struct{ pool *pgxpool.Pool }

func (r *pgxTxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, exec port.Executor) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type loggerAdapter struct{ l *slog.Logger }

func (a loggerAdapter) Debug(msg string, args ...any) { a.l.Debug(msg, args...) }
func (a loggerAdapter) Info(msg string, args ...any)  { a.l.Info(msg, args...) }
func (a loggerAdapter) Warn(msg string, args ...any)  { a.l.Warn(msg, args...) }
func (a loggerAdapter) Error(msg string, args ...any) { a.l.Error(msg, args...) }

type inboxAdapter struct {
	inner *inbox.PostgresInbox
}

func (a *inboxAdapter) IsProcessed(ctx context.Context, eventID, handlerName string) (bool, error) {
	return a.inner.IsProcessed(ctx, eventID, handlerName)
}

func (a *inboxAdapter) MarkProcessed(ctx context.Context, eventID, handlerName, eventType string) error {
	return a.inner.MarkProcessed(ctx, eventID, handlerName, eventType)
}

func (a *inboxAdapter) MarkProcessedTx(ctx context.Context, _ inbox.DBTX, eventID, handlerName, eventType string) error {
	return a.inner.MarkProcessed(ctx, eventID, handlerName, eventType)
}

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
