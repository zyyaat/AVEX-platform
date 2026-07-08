// Package port repository: persistence interfaces for the financial module.
//
// Each repository interface covers one domain entity. Methods accept an
// Executor explicitly — transactions are never hidden in context.
//
// Design rules:
//   - Every method takes (ctx, exec, ...) where exec is either a pool
//     (for non-transactional ops) or a transaction (for atomic ops).
//   - Methods return domain entity pointers on success, nil + sentinel
//     domain error on failure.
//   - Repositories do NOT publish events — the service layer calls
//     EventPublisher within the same transaction.
//
// Imports: stdlib + domain only.
package port

import (
	"context"
	"time"

	"avex-backend/internal/modules/financial/domain"
)

// ===== WalletRepository =====

// WalletRepository persists Wallet entities.
type WalletRepository interface {
	// Create inserts a new wallet. Returns ErrWalletAlreadyExists if a
	// wallet with the same (owner_type, owner_id, currency) already exists.
	Create(ctx context.Context, exec Executor, wallet domain.Wallet) error

	// GetByID retrieves a wallet by its UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Wallet, error)

	// GetByOwner retrieves the wallet for a given owner in a given currency.
	// Returns ErrWalletNotFound if not found.
	GetByOwner(ctx context.Context, exec Executor, ownerType domain.OwnerType, ownerID, currency string) (*domain.Wallet, error)

	// Update saves all fields of an existing wallet. Uses optimistic locking:
	// the WHERE clause checks the current version; if mismatched, returns
	// ErrWalletNotFound (treating concurrent modification as not-found).
	// On success, the version in the DB is incremented.
	Update(ctx context.Context, exec Executor, wallet domain.Wallet) error

	// UpdateBalanceAndStatus performs a partial update of balance,
	// pending_balance, status, version, and updated_at in a single statement.
	// Uses the same optimistic-locking pattern as Update.
	UpdateBalanceAndStatus(ctx context.Context, exec Executor, wallet domain.Wallet) error

	// ListByOwner retrieves all wallets for a given owner (across currencies).
	ListByOwner(ctx context.Context, exec Executor, ownerType domain.OwnerType, ownerID string) ([]domain.Wallet, error)
}

// ===== TransactionRepository =====

// TransactionRepository persists Transaction entities (append-only ledger).
type TransactionRepository interface {
	// Create inserts a new transaction.
	// Returns ErrDuplicateIdempotencyKey if a transaction with the same
	// idempotency_key already exists (and that key is non-empty).
	Create(ctx context.Context, exec Executor, txn domain.Transaction) error

	// GetByID retrieves a transaction by its UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Transaction, error)

	// GetByIdempotencyKey retrieves a transaction by its idempotency key.
	// Returns ErrTransactionNotFound if not found or key is empty.
	// Used by the service layer to deduplicate credit/debit operations.
	GetByIdempotencyKey(ctx context.Context, exec Executor, key string) (*domain.Transaction, error)

	// UpdateStatus updates a transaction's status and (if applicable) completed_at.
	UpdateStatus(ctx context.Context, exec Executor, id string, status domain.TransactionStatus, completedAt *time.Time) error

	// ListByWallet retrieves a paginated list of transactions for a wallet.
	ListByWallet(ctx context.Context, exec Executor, walletID string, page PageQuery) (Page[domain.Transaction], error)

	// ListByReference retrieves transactions linked to a reference (e.g. order_id).
	ListByReference(ctx context.Context, exec Executor, refType domain.ReferenceType, refID string) ([]domain.Transaction, error)
}

// ===== PromotionRepository =====

// PromotionRepository persists Promotion entities.
type PromotionRepository interface {
	// Create inserts a new promotion. Returns ErrPromotionCodeAlreadyExists
	// if the code is taken.
	Create(ctx context.Context, exec Executor, promo domain.Promotion) error

	// GetByID retrieves a promotion by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Promotion, error)

	// GetByCode retrieves a promotion by its (case-insensitive) code.
	GetByCode(ctx context.Context, exec Executor, code string) (*domain.Promotion, error)

	// Update saves all fields. Used by IncrementUsage (atomic UPDATE ... SET usage_count = usage_count + 1).
	Update(ctx context.Context, exec Executor, promo domain.Promotion) error

	// IncrementUsage atomically increments usage_count by 1 with a WHERE
	// clause that checks usage_count < usage_limit (or usage_limit IS NULL).
	// Returns ErrPromotionUsageLimitReached if the limit would be exceeded.
	IncrementUsage(ctx context.Context, exec Executor, id string) error

	// ListActive retrieves all active promotions valid at the given time.
	ListActive(ctx context.Context, exec Executor, now time.Time) ([]domain.Promotion, error)

	// ListAll retrieves all promotions (admin view) with pagination.
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.Promotion], error)
}

// ===== PromotionRedemptionRepository =====

// PromotionRedemptionRepository persists PromotionRedemption records.
type PromotionRedemptionRepository interface {
	// Create inserts a new redemption record.
	// Returns ErrPromoAlreadyRedeemed if (promotion_id, user_id, order_id) already exists.
	Create(ctx context.Context, exec Executor, redemption domain.PromotionRedemption) error

	// CountByUserAndPromotion returns the number of redemptions for a given
	// (promotion_id, user_id) pair. Used to enforce per_user_limit.
	CountByUserAndPromotion(ctx context.Context, exec Executor, promotionID, userID string) (int, error)

	// ListByUser retrieves all redemptions for a user.
	ListByUser(ctx context.Context, exec Executor, userID string, page PageQuery) (Page[domain.PromotionRedemption], error)

	// ListByOrder retrieves all redemptions applied to a specific order.
	ListByOrder(ctx context.Context, exec Executor, orderID string) ([]domain.PromotionRedemption, error)
}

// ===== PricingRuleRepository =====

// PricingRuleRepository persists PricingRule entities.
type PricingRuleRepository interface {
	// Create inserts a new pricing rule.
	// Returns ErrPricingRuleAlreadyExists if an active rule for the same zone
	// and currency already exists at the given valid_from.
	Create(ctx context.Context, exec Executor, rule domain.PricingRule) error

	// GetByID retrieves a rule by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.PricingRule, error)

	// GetActiveForZone retrieves the active pricing rule applicable to the
	// given zone and currency at the given time. Returns ErrPricingRuleNotFound
	// if none matches.
	GetActiveForZone(ctx context.Context, exec Executor, zoneID, currency string, now time.Time) (*domain.PricingRule, error)

	// ListAll retrieves all pricing rules (admin view) with pagination.
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.PricingRule], error)
}

// ===== SurgeZoneRepository =====

// SurgeZoneRepository persists SurgeZone entities.
type SurgeZoneRepository interface {
	// Create inserts a new surge zone.
	Create(ctx context.Context, exec Executor, surge domain.SurgeZone) error

	// GetByID retrieves a surge zone by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.SurgeZone, error)

	// GetActiveForZone retrieves all active surge zones for a given zone,
	// then the service layer filters by IsApplicableAt(now).
	GetActiveForZone(ctx context.Context, exec Executor, zoneID string) ([]domain.SurgeZone, error)

	// ListAll retrieves all surge zones (admin view) with pagination.
	ListAll(ctx context.Context, exec Executor, page PageQuery) (Page[domain.SurgeZone], error)

	// Deactivate marks a surge zone as inactive.
	Deactivate(ctx context.Context, exec Executor, id string) error
}

// ===== TaxRepository =====

// TaxRepository persists Tax entities.
type TaxRepository interface {
	// Create inserts a new tax rule.
	Create(ctx context.Context, exec Executor, tax domain.Tax) error

	// GetByID retrieves a tax by UUID.
	GetByID(ctx context.Context, exec Executor, id string) (*domain.Tax, error)

	// ListActive retrieves all active tax rules.
	ListActive(ctx context.Context, exec Executor) ([]domain.Tax, error)
}

// ===== OutboxRepository =====

// OutboxRepository persists event envelopes for the outbox pattern.
type OutboxRepository interface {
	// Save persists an event envelope in the outbox within the given
	// transaction (or pool).
	Save(ctx context.Context, exec Executor, envelope EventEnvelope) error

	// GetPending retrieves up to limit unpublished events whose
	// next_retry_at has passed.
	GetPending(ctx context.Context, exec Executor, limit int) ([]EventEnvelope, error)

	// MarkPublished marks an event as successfully published.
	MarkPublished(ctx context.Context, exec Executor, eventID string) error
}

// ===== Aggregate =====

// RepositorySet aggregates all financial repository interfaces.
type RepositorySet struct {
	Wallets       WalletRepository
	Transactions  TransactionRepository
	Promotions    PromotionRepository
	Redemptions   PromotionRedemptionRepository
	PricingRules  PricingRuleRepository
	SurgeZones    SurgeZoneRepository
	Taxes         TaxRepository
	Outbox        OutboxRepository
}
