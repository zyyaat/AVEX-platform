// Package postgres wallet_repository: WalletRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/port"
)

// WalletRepository implements port.WalletRepository using pgx/v5.
type WalletRepository struct{}

var _ port.WalletRepository = (*WalletRepository)(nil)

// Create inserts a new wallet. Returns ErrWalletAlreadyExists on unique violation.
func (r *WalletRepository) Create(ctx context.Context, exec port.Executor, wallet domain.Wallet) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.wallets (
			id, owner_type, owner_id, currency,
			balance, pending_balance, status, version,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10
		)
	`,
		wallet.ID(),
		string(wallet.OwnerType()),
		wallet.OwnerID(),
		wallet.Currency(),
		wallet.Balance().Amount(),
		wallet.PendingBalance().Amount(),
		string(wallet.Status()),
		wallet.Version(),
		wallet.CreatedAt(),
		wallet.UpdatedAt(),
	)
	if err != nil {
		return mapWalletWriteError(err)
	}
	return nil
}

// GetByID retrieves a wallet by UUID.
func (r *WalletRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Wallet, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `
		SELECT id, owner_type, owner_id, currency,
		       balance, pending_balance, status, version,
		       created_at, updated_at
		FROM financial.wallets
		WHERE id = $1
	`, id)
	w, err := scanWallet(row)
	if err != nil {
		return nil, mapWalletReadError(err)
	}
	return &w, nil
}

// GetByOwner retrieves the wallet for a given owner in a given currency.
func (r *WalletRepository) GetByOwner(ctx context.Context, exec port.Executor, ownerType domain.OwnerType, ownerID, currency string) (*domain.Wallet, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `
		SELECT id, owner_type, owner_id, currency,
		       balance, pending_balance, status, version,
		       created_at, updated_at
		FROM financial.wallets
		WHERE owner_type = $1 AND owner_id = $2 AND currency = $3
	`, string(ownerType), ownerID, currency)
	w, err := scanWallet(row)
	if err != nil {
		return nil, mapWalletReadError(err)
	}
	return &w, nil
}

// Update saves all fields of an existing wallet using optimistic locking.
// On success, the version is incremented by 1 in the DB.
func (r *WalletRepository) Update(ctx context.Context, exec port.Executor, wallet domain.Wallet) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE financial.wallets SET
			balance = $2,
			pending_balance = $3,
			status = $4,
			version = version + 1,
			updated_at = $5
		WHERE id = $1 AND version = $6
	`,
		wallet.ID(),
		wallet.Balance().Amount(),
		wallet.PendingBalance().Amount(),
		string(wallet.Status()),
		wallet.UpdatedAt(),
		wallet.Version(),
	)
	if err != nil {
		return mapWalletWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		// Either wallet does not exist OR version mismatch (concurrent modification).
		return fmt.Errorf("%w: optimistic lock failed for wallet %s", domain.ErrWalletNotFound, wallet.ID())
	}
	return nil
}

// UpdateBalanceAndStatus performs a partial update optimized for credit/debit.
func (r *WalletRepository) UpdateBalanceAndStatus(ctx context.Context, exec port.Executor, wallet domain.Wallet) error {
	return r.Update(ctx, exec, wallet)
}

// ListByOwner retrieves all wallets for a given owner (across currencies).
func (r *WalletRepository) ListByOwner(ctx context.Context, exec port.Executor, ownerType domain.OwnerType, ownerID string) ([]domain.Wallet, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT id, owner_type, owner_id, currency,
		       balance, pending_balance, status, version,
		       created_at, updated_at
		FROM financial.wallets
		WHERE owner_type = $1 AND owner_id = $2
		ORDER BY currency
	`, string(ownerType), ownerID)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()
	var wallets []domain.Wallet
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan wallet: %w", err)
		}
		wallets = append(wallets, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return wallets, nil
}

// ===== Mapper Helpers =====

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// walletColumns is the canonical SELECT column list.
const walletColumns = `id, owner_type, owner_id, currency, balance, pending_balance, status, version, created_at, updated_at`

// scanWallet reads a wallet row from either pgx.Row or pgx.Rows.
func scanWallet(s scanner) (domain.Wallet, error) {
	var (
		id, ownerID, currency   string
		ownerType, status       string
		balance, pendingBalance int64
		version                 int
		createdAt, updatedAt    time.Time
	)
	if err := s.Scan(
		&id, &ownerType, &ownerID, &currency,
		&balance, &pendingBalance, &status, &version,
		&createdAt, &updatedAt,
	); err != nil {
		return domain.Wallet{}, err
	}
	return domain.RehydrateWallet(
		id,
		domain.OwnerType(ownerType),
		ownerID,
		currency,
		balance,
		pendingBalance,
		domain.WalletStatus(status),
		version,
		createdAt,
		updatedAt,
	), nil
}

// ===== Error Mappers =====

func mapWalletWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 = unique_violation
		if pgErr.Code == "23505" {
			return domain.ErrWalletAlreadyExists
		}
	}
	return fmt.Errorf("wallet write: %w", err)
}

func mapWalletReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrWalletNotFound
	}
	return fmt.Errorf("wallet read: %w", err)
}
