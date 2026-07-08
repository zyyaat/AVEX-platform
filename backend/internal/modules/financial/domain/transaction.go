// Package domain transaction: Transaction entity (append-only ledger entry).
//
// Every money movement in the system creates exactly one Transaction row.
// Transactions are IMMUTABLE after creation except for status transitions
// (pending → completed/failed) and reversal (creating a new reversal transaction).
//
// A Transaction is associated with exactly one Wallet. Cross-wallet transfers
// are represented as two transactions (debit on source, credit on destination),
// linked by the same reference_id.
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// TransactionType enumerates the two movement directions.
type TransactionType string

const (
	TxnTypeCredit TransactionType = "credit"
	TxnTypeDebit  TransactionType = "debit"
)

// TransactionCategory describes the business reason for the money movement.
type TransactionCategory string

const (
	CategoryTopup        TransactionCategory = "topup"
	CategoryOrderPayment TransactionCategory = "order_payment"
	CategoryRefund       TransactionCategory = "refund"
	CategoryPayout       TransactionCategory = "payout"
	CategoryCommission   TransactionCategory = "commission"
	CategoryTip          TransactionCategory = "tip"
	CategoryAdjustment   TransactionCategory = "adjustment"
	CategoryPromotion    TransactionCategory = "promotion"
)

// TransactionStatus enumerates lifecycle states.
type TransactionStatus string

const (
	TxnStatusPending   TransactionStatus = "pending"
	TxnStatusCompleted TransactionStatus = "completed"
	TxnStatusFailed    TransactionStatus = "failed"
	TxnStatusReversed  TransactionStatus = "reversed"
)

// ReferenceType enumerates what kind of entity a transaction refers to.
type ReferenceType string

const (
	RefTypeOrder    ReferenceType = "order"
	RefTypePromo    ReferenceType = "promotion"
	RefTypeManual   ReferenceType = "manual"
	RefTypeSystem   ReferenceType = "system"
	RefTypePayout   ReferenceType = "payout"
	RefTypeTopup    ReferenceType = "topup"
)

// Transaction is an immutable ledger entry.
type Transaction struct {
	id              string
	walletID        string
	txnType         TransactionType
	category        TransactionCategory
	amount          Money
	status          TransactionStatus
	referenceType   ReferenceType
	referenceID     string
	description     string
	metadata        map[string]any
	idempotencyKey  string
	createdAt       time.Time
	completedAt     *time.Time
}

// NewTransaction creates a new Transaction with validation.
// Amount must be positive, currency must match the wallet's currency (caller verifies).
// New transactions start in Pending status.
func NewTransaction(
	id string,
	walletID string,
	txnType TransactionType,
	category TransactionCategory,
	amount Money,
	referenceType ReferenceType,
	referenceID, description string,
	metadata map[string]any,
	idempotencyKey string,
	now time.Time,
) (Transaction, error) {
	if id == "" {
		return Transaction{}, fmt.Errorf("%w: transaction id is required", ErrInvalidID)
	}
	if walletID == "" {
		return Transaction{}, fmt.Errorf("%w: wallet id is required", ErrInvalidInput)
	}
	if txnType != TxnTypeCredit && txnType != TxnTypeDebit {
		return Transaction{}, fmt.Errorf("%w: %s", ErrInvalidTransactionType, txnType)
	}
	if !isValidCategory(category) {
		return Transaction{}, fmt.Errorf("%w: %s", ErrInvalidTransactionCategory, category)
	}
	if !amount.IsPositive() {
		return Transaction{}, fmt.Errorf("%w: transaction amount must be positive", ErrInvalidMoneyAmount)
	}
	return Transaction{
		id:             id,
		walletID:       walletID,
		txnType:        txnType,
		category:       category,
		amount:         amount,
		status:         TxnStatusPending,
		referenceType:  referenceType,
		referenceID:    referenceID,
		description:    description,
		metadata:       metadata,
		idempotencyKey: idempotencyKey,
		createdAt:      now,
	}, nil
}

// RehydrateTransaction reconstructs a Transaction from persistence.
func RehydrateTransaction(
	id, walletID string,
	txnType TransactionType,
	category TransactionCategory,
	amount Money,
	status TransactionStatus,
	referenceType ReferenceType,
	referenceID, description string,
	metadata map[string]any,
	idempotencyKey string,
	createdAt time.Time,
	completedAt *time.Time,
) Transaction {
	return Transaction{
		id:             id,
		walletID:       walletID,
		txnType:        txnType,
		category:       category,
		amount:         amount,
		status:         status,
		referenceType:  referenceType,
		referenceID:    referenceID,
		description:    description,
		metadata:       metadata,
		idempotencyKey: idempotencyKey,
		createdAt:      createdAt,
		completedAt:    completedAt,
	}
}

func isValidCategory(c TransactionCategory) bool {
	switch c {
	case CategoryTopup, CategoryOrderPayment, CategoryRefund, CategoryPayout,
		CategoryCommission, CategoryTip, CategoryAdjustment, CategoryPromotion:
		return true
	}
	return false
}

// ===== Accessors =====

func (t Transaction) ID() string                       { return t.id }
func (t Transaction) WalletID() string                 { return t.walletID }
func (t Transaction) Type() TransactionType            { return t.txnType }
func (t Transaction) Category() TransactionCategory    { return t.category }
func (t Transaction) Amount() Money                    { return t.amount }
func (t Transaction) Status() TransactionStatus        { return t.status }
func (t Transaction) ReferenceType() ReferenceType     { return t.referenceType }
func (t Transaction) ReferenceID() string              { return t.referenceID }
func (t Transaction) Description() string              { return t.description }
func (t Transaction) Metadata() map[string]any         { return t.metadata }
func (t Transaction) IdempotencyKey() string           { return t.idempotencyKey }
func (t Transaction) CreatedAt() time.Time             { return t.createdAt }
func (t Transaction) CompletedAt() *time.Time          { return t.completedAt }

// IsCredit reports whether this is a credit transaction.
func (t Transaction) IsCredit() bool { return t.txnType == TxnTypeCredit }

// IsDebit reports whether this is a debit transaction.
func (t Transaction) IsDebit() bool { return t.txnType == TxnTypeDebit }

// ===== Status Transitions =====
//
// Valid transitions:
//   pending → completed
//   pending → failed
//   completed → reversed
// All other transitions are rejected.

// MarkCompleted transitions the transaction from Pending to Completed.
func (t Transaction) MarkCompleted(now time.Time) (Transaction, error) {
	if t.status == TxnStatusCompleted {
		return t, ErrTransactionAlreadyCompleted
	}
	if t.status != TxnStatusPending {
		return t, fmt.Errorf("%w: cannot complete a %s transaction", ErrTransactionCannotBeReversed, t.status)
	}
	t.status = TxnStatusCompleted
	ts := now
	t.completedAt = &ts
	return t, nil
}

// MarkFailed transitions the transaction from Pending to Failed.
func (t Transaction) MarkFailed(now time.Time) (Transaction, error) {
	if t.status == TxnStatusFailed {
		return t, ErrTransactionAlreadyFailed
	}
	if t.status != TxnStatusPending {
		return t, fmt.Errorf("%w: cannot fail a %s transaction", ErrTransactionCannotBeReversed, t.status)
	}
	t.status = TxnStatusFailed
	return t, nil
}

// MarkReversed transitions the transaction from Completed to Reversed.
// Reversal does NOT itself move money — the caller must create a compensating
// credit/debit transaction on the same wallet.
func (t Transaction) MarkReversed() (Transaction, error) {
	if t.status != TxnStatusCompleted {
		return t, fmt.Errorf("%w: only completed transactions can be reversed", ErrTransactionCannotBeReversed)
	}
	t.status = TxnStatusReversed
	return t, nil
}
