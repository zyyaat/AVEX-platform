// Package postgres transaction_repository: TransactionRepository implementation.
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

// TransactionRepository implements port.TransactionRepository using pgx/v5.
type TransactionRepository struct{}

var _ port.TransactionRepository = (*TransactionRepository)(nil)

// Create inserts a new transaction.
func (r *TransactionRepository) Create(ctx context.Context, exec port.Executor, txn domain.Transaction) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.transactions (
			id, wallet_id, type, category,
			amount, currency, status,
			reference_type, reference_id, description, metadata,
			idempotency_key, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14
		)
	`,
		txn.ID(),
		txn.WalletID(),
		string(txn.Type()),
		string(txn.Category()),
		txn.Amount().Amount(),
		txn.Amount().Currency(),
		string(txn.Status()),
		string(txn.ReferenceType()),
		txn.ReferenceID(),
		txn.Description(),
		txn.Metadata(),
		txn.IdempotencyKey(),
		txn.CreatedAt(),
		txn.CompletedAt(),
	)
	if err != nil {
		return mapTransactionWriteError(err)
	}
	return nil
}

// GetByID retrieves a transaction by UUID.
func (r *TransactionRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Transaction, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+transactionColumns+` FROM financial.transactions WHERE id = $1`, id)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, mapTransactionReadError(err)
	}
	return &t, nil
}

// GetByIdempotencyKey retrieves a transaction by idempotency_key.
// Returns ErrTransactionNotFound if key is empty or no match.
func (r *TransactionRepository) GetByIdempotencyKey(ctx context.Context, exec port.Executor, key string) (*domain.Transaction, error) {
	if key == "" {
		return nil, domain.ErrTransactionNotFound
	}
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+transactionColumns+` FROM financial.transactions WHERE idempotency_key = $1`, key)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, mapTransactionReadError(err)
	}
	return &t, nil
}

// UpdateStatus updates a transaction's status and completed_at.
func (r *TransactionRepository) UpdateStatus(ctx context.Context, exec port.Executor, id string, status domain.TransactionStatus, completedAt *time.Time) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE financial.transactions
		SET status = $2, completed_at = $3
		WHERE id = $1
	`, id, string(status), completedAt)
	if err != nil {
		return fmt.Errorf("update txn status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}

// ListByWallet retrieves a paginated list of transactions for a wallet.
func (r *TransactionRepository) ListByWallet(ctx context.Context, exec port.Executor, walletID string, page port.PageQuery) (port.Page[domain.Transaction], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	// Count total
	var total int64
	err := dbtx.QueryRow(ctx, `SELECT COUNT(*) FROM financial.transactions WHERE wallet_id = $1`, walletID).Scan(&total)
	if err != nil {
		return port.Page[domain.Transaction]{}, fmt.Errorf("count txns: %w", err)
	}

	// Fetch page
	rows, err := dbtx.Query(ctx, `
		SELECT `+transactionColumns+`
		FROM financial.transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, walletID, page.Limit, page.Offset)
	if err != nil {
		return port.Page[domain.Transaction]{}, fmt.Errorf("list txns: %w", err)
	}
	defer rows.Close()

	var items []domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return port.Page[domain.Transaction]{}, fmt.Errorf("scan txn: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.Transaction]{}, fmt.Errorf("rows: %w", err)
	}

	return port.Page[domain.Transaction]{
		Items:  items,
		Total:  total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}, nil
}

// ListByReference retrieves transactions linked to a reference.
func (r *TransactionRepository) ListByReference(ctx context.Context, exec port.Executor, refType domain.ReferenceType, refID string) ([]domain.Transaction, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+transactionColumns+`
		FROM financial.transactions
		WHERE reference_type = $1 AND reference_id = $2
		ORDER BY created_at ASC
	`, string(refType), refID)
	if err != nil {
		return nil, fmt.Errorf("list by reference: %w", err)
	}
	defer rows.Close()

	var items []domain.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan txn: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ===== Mapper Helpers =====

const transactionColumns = `id, wallet_id, type, category, amount, currency, status, reference_type, reference_id, description, metadata, idempotency_key, created_at, completed_at`

func scanTransaction(s scanner) (domain.Transaction, error) {
	var (
		id, walletID, txnType, category string
		amount                          int64
		currency, status                string
		refType, refID                  *string
		description                     *string
		metadata                        []byte
		idempotencyKey                  *string
		createdAt                       time.Time
		completedAt                     *time.Time
	)
	if err := s.Scan(
		&id, &walletID, &txnType, &category,
		&amount, &currency, &status,
		&refType, &refID, &description, &metadata,
		&idempotencyKey, &createdAt, &completedAt,
	); err != nil {
		return domain.Transaction{}, err
	}

	// Convert nullable fields
	var refTypeStr, refIDStr, descStr, idempotencyStr string
	if refType != nil {
		refTypeStr = *refType
	}
	if refID != nil {
		refIDStr = *refID
	}
	if description != nil {
		descStr = *description
	}
	if idempotencyKey != nil {
		idempotencyStr = *idempotencyKey
	}

	// Parse metadata JSONB
	var metaMap map[string]any
	if len(metadata) > 0 {
		// Use encoding/json for the JSONB byte slice.
		if err := jsonUnmarshal(metadata, &metaMap); err != nil {
			return domain.Transaction{}, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	amt, _ := domain.NewMoney(amount, currency)

	return domain.RehydrateTransaction(
		id,
		walletID,
		domain.TransactionType(txnType),
		domain.TransactionCategory(category),
		amt,
		domain.TransactionStatus(status),
		domain.ReferenceType(refTypeStr),
		refIDStr,
		descStr,
		metaMap,
		idempotencyStr,
		createdAt,
		completedAt,
	), nil
}

func mapTransactionWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domain.ErrDuplicateIdempotencyKey
		}
	}
	return fmt.Errorf("transaction write: %w", err)
}

func mapTransactionReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrTransactionNotFound
	}
	return fmt.Errorf("transaction read: %w", err)
}
