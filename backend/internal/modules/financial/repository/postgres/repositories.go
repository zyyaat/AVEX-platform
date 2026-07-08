// Package postgres implements the financial module's repository interfaces
// using pgx/v5 against a PostgreSQL database.
//
// Design rules (enforced by the port layer):
//   - No business logic. The repository only CRUDs + maps.
//   - No direct pool access inside methods. Every method receives a
//     port.Executor which is converted to database.DBTX via the
//     toDBTX adapter.
//   - No SQL mapping inside domain entities. All row <-> entity
//     conversion lives in mapper.go.
//   - Methods return domain sentinel errors (e.g. ErrWalletNotFound)
//     on expected failure paths, wrapped errors on infrastructure
//     failures.
//
// Schema: all tables live in the PostgreSQL schema "financial".
package postgres

import (
	"avex-backend/internal/modules/financial/port"
	"avex-backend/internal/platform/database"
)

// Repositories is the concrete implementation of port.RepositorySet.
type Repositories struct {
	wallets     *WalletRepository
	transactions *TransactionRepository
	promotions  *PromotionRepository
	redemptions *PromotionRedemptionRepository
	pricing     *PricingRuleRepository
	surge       *SurgeZoneRepository
	taxes       *TaxRepository
	outbox      *OutboxRepository
}

// NewRepositories constructs a Repositories backed by the given pgxpool.
func NewRepositories() *Repositories {
	return &Repositories{
		wallets:      &WalletRepository{},
		transactions: &TransactionRepository{},
		promotions:   &PromotionRepository{},
		redemptions:  &PromotionRedemptionRepository{},
		pricing:      &PricingRuleRepository{},
		surge:        &SurgeZoneRepository{},
		taxes:        &TaxRepository{},
		outbox:       &OutboxRepository{},
	}
}

// RepositorySet returns a port.RepositorySet backed by this Repositories.
func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{
		Wallets:      r.wallets,
		Transactions: r.transactions,
		Promotions:   r.promotions,
		Redemptions:  r.redemptions,
		PricingRules: r.pricing,
		SurgeZones:   r.surge,
		Taxes:        r.taxes,
		Outbox:       r.outbox,
	}
}

// toDBTX converts a port.Executor (opaque interface{}) into a
// database.DBTX. Panics on wiring error (fail fast).
func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok {
		panic("postgres: port.Executor does not satisfy database.DBTX — check composition root wiring")
	}
	return dbtx
}
