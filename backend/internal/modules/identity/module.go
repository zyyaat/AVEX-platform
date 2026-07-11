// Package identity is the composition root for the identity module.
//
// module.go wires all dependencies:
//
//      platform implementations
//              ↓
//      adapters (port.* satisfaction)
//              ↓
//      repositories (postgres)
//              ↓
//      event publisher (stateless)
//              ↓
//      service (use cases)
//              ↓
//      transport handlers (HTTP)
//
// The Module struct returned by New() is the single object the application
// (cmd/server) interacts with. It exposes:
//   - ServicePort (for cross-module calls in the future)
//   - RegisterRoutes (for HTTP route registration)
//   - Close (for graceful shutdown)
//
// Dependency injection is explicit — no global state, no service locators.
package identity

import (
        "context"
        "log/slog"
        "net/http"
        "time"

        "github.com/golang-jwt/jwt/v5"
        "github.com/jackc/pgx/v5/pgxpool"

        "avex-backend/internal/modules/identity/events"
        "avex-backend/internal/modules/identity/port"
        "avex-backend/internal/modules/identity/repository/postgres"
        "avex-backend/internal/modules/identity/service"
        httptransport "avex-backend/internal/modules/identity/transport/http"
        "avex-backend/internal/platform/config"
        "avex-backend/internal/platform/crypto"
        "avex-backend/internal/platform/outbox"
)

// Module is the wired identity module. The application holds this and
// calls its methods to register routes, access the service, etc.
type Module struct {
        svc       port.ServicePort
        pool      *pgxpool.Pool
        jwtIssuer port.JWTIssuer
        logger    *slog.Logger
}

// New wires all identity dependencies and returns a ready-to-use Module.
//
// Parameters:
//   - cfg:        platform config (for JWT TTL, bcrypt cost, etc.)
//   - pool:       the pgxpool to use for DB access
//   - logger:     structured logger
//
// The pool is used both for the repository implementations and as the
// non-transactional Executor for service reads.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *Module {
        // ----- Platform Implementations (concrete) -----
        passwordHasher := crypto.NewBcryptHasher(cfg.Bcrypt.Cost)
        jwtImpl := crypto.NewHS256Issuer(cfg.JWT.Secret, cfg.JWT.Issuer)

        // ----- Adapters (port.* satisfaction) -----
        clockAdapter := &realClock{}
        idGenAdapter := &uuidGen{}
        jwtAdapter := &jwtIssuerAdapter{impl: jwtImpl}
        loggerAdapter := logger // *slog.Logger satisfies port.Logger natively

        // ----- Outbox -----
        identityOutbox := outbox.NewPostgresOutbox(pool, outbox.Config{
                Table:          "identity.outbox",
                MaxRetries:     cfg.Outbox.MaxRetries,
                RetryBaseDelay: cfg.Outbox.RetryBaseDelay,
        })

        // ----- Event Publisher (stateless) -----
        eventPublisher := events.NewEventPublisher(identityOutbox, idGenAdapter)

        // ----- Repositories -----
        repos := postgres.NewRepositories()
        repoSet := repos.RepositorySet()

        // ----- TxRunner -----
        txRunner := &pgxTxRunner{pool: pool}

        // ----- Service -----
        deps := port.Deps{
                Clock:          clockAdapter,
                IDGenerator:    idGenAdapter,
                PasswordHasher: passwordHasher,
                JWTIssuer:      jwtAdapter,
                EventPublisher: eventPublisher,
                Logger:         loggerAdapter,
                TxRunner:       txRunner,
                Repos:          repoSet,
        }

        svc := service.New(deps, pool, service.Config{
                AccessTokenTTL: cfg.JWT.AccessTTL,
        })

        return &Module{
                svc:       svc,
                pool:      pool,
                jwtIssuer: jwtAdapter,
                logger:    logger,
        }
}

// Service returns the identity ServicePort for cross-module calls.
// JWTIssuer returns the identity module's JWT issuer.
// Used by other modules (e.g. orders) for auth middleware.
func (m *Module) JWTIssuer() port.JWTIssuer {
        return m.jwtIssuer
}

func (m *Module) Service() port.ServicePort {
        return m.svc
}

// RegisterRoutes registers all identity HTTP routes on the given mux.
// The caller (cmd/server) is responsible for applying platform-level
// middleware (requestid, logging, recovery, CORS) before calling this.
func (m *Module) RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
        httptransport.RegisterRoutes(mux, m.svc, httptransport.RoutesConfig{
                JWTIssuer:      m.jwtIssuer,
                Logger:         m.logger,
                AllowedOrigins: cfg.CORS.AllowedOrigins,
                Pool:           m.pool,
        })
}

// Close releases any resources held by the module.
// The pool is NOT closed here — the caller (cmd/server) owns the pool
// and closes it during shutdown.
func (m *Module) Close() {
        // Nothing to close yet — repos and publisher are stateless.
}

// ===== Adapters =====
//
// These adapters convert platform implementations into port interfaces.
// They live here (not in platform/) because they're module-specific —
// each module may need slightly different adapter behavior.

// realClock adapts time.Now() to port.Clock.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// uuidGen adapts the id package to port.IDGenerator.
// For now we use a simple UUID v4 generator. The actual id package
// could be used, but to avoid an import cycle we inline a minimal version.
type uuidGen struct{}

func (*uuidGen) New() string {
        // Use google/uuid via the platform/id package would be ideal, but
        // to keep module.go self-contained, we generate a v4 UUID here.
        // This is a placeholder — in production, use platform/id.
        return newUUID()
}

// jwtIssuerAdapter converts between port.JWTClaims and crypto.Claims.
type jwtIssuerAdapter struct {
        impl crypto.JWTIssuer
}

func (a *jwtIssuerAdapter) Issue(ctx context.Context, params port.IssueJWTParams) (string, error) {
        claims := crypto.Claims{
                Role:      params.Role,
                SessionID: params.SessionID,
                RegisteredClaims: jwt.RegisteredClaims{
                        Subject:   params.Subject,
                        ExpiresAt: jwt.NewNumericDate(params.ExpiresAt),
                        IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
                },
        }
        return a.impl.Issue(claims)
}

func (a *jwtIssuerAdapter) Verify(ctx context.Context, token string) (*port.JWTClaims, error) {
        claims, err := a.impl.Verify(token)
        if err != nil {
                return nil, err
        }
        return &port.JWTClaims{
                Subject:   claims.Subject,
                Role:      claims.Role,
                SessionID: claims.SessionID,
                ExpiresAt: claims.ExpiresAt.Time,
        }, nil
}

// pgxTxRunner implements port.TxRunner using pgxpool.
type pgxTxRunner struct {
        pool *pgxpool.Pool
}

func (r *pgxTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context, exec port.Executor) error) error {
        tx, err := r.pool.Begin(ctx)
        if err != nil {
                return err
        }
        // Defer rollback — it's a no-op if the tx was committed.
        defer func() {
                _ = tx.Rollback(ctx)
        }()

        if err := fn(ctx, tx); err != nil {
                return err
        }
        return tx.Commit(ctx)
}

// Compile-time assertions that adapters satisfy port interfaces.
var (
        _ port.Clock       = (*realClock)(nil)
        _ port.IDGenerator = (*uuidGen)(nil)
        _ port.JWTIssuer   = (*jwtIssuerAdapter)(nil)
        _ port.TxRunner    = (*pgxTxRunner)(nil)
        _ port.Logger      = (*slog.Logger)(nil)
)

// newUUID generates a UUID v4 string.
// Implemented in uuid.go to keep module.go clean.
func newUUID() string {
        return uuidV4()
}
