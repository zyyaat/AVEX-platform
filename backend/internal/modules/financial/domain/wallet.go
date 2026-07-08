// Package domain wallet: Wallet aggregate root.
//
// A Wallet holds the balance for an owner (user, driver, or merchant).
// All mutations go through Credit/Debit methods that enforce invariants:
//   - Frozen wallets cannot be debited (but CAN be credited for refunds)
//   - Closed wallets cannot be mutated
//   - Balance can never go negative
//   - Currency is immutable after creation
//
// Optimistic locking is handled at the repository layer via the Version field.
// The service layer must call repo.Update with the current version; the repo
// will increment version and reject concurrent modifications.
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// OwnerType enumerates wallet owner categories.
type OwnerType string

const (
	OwnerTypeUser     OwnerType = "user"
	OwnerTypeDriver   OwnerType = "driver"
	OwnerTypeMerchant OwnerType = "merchant"
)

// WalletStatus enumerates wallet lifecycle states.
type WalletStatus string

const (
	WalletStatusActive WalletStatus = "active"
	WalletStatusFrozen WalletStatus = "frozen"
	WalletStatusClosed WalletStatus = "closed"
)

// Wallet is the aggregate root for a financial wallet.
// Fields are private; access goes through methods to enforce invariants.
type Wallet struct {
	id             string
	ownerType      OwnerType
	ownerID        string
	currency       string
	balance        Money // available balance (cents)
	pendingBalance Money // holds in progress (e.g. escrow for ongoing orders)
	status         WalletStatus
	version        int       // optimistic locking
	createdAt      time.Time
	updatedAt      time.Time
}

// NewWallet creates a new Wallet with validation.
// Returns an error if ownerType is invalid, ownerID is empty, or currency is invalid.
func NewWallet(id string, ownerType OwnerType, ownerID, currency string, now time.Time) (Wallet, error) {
	if id == "" {
		return Wallet{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if ownerType != OwnerTypeUser && ownerType != OwnerTypeDriver && ownerType != OwnerTypeMerchant {
		return Wallet{}, fmt.Errorf("%w: %s", ErrInvalidOwnerType, ownerType)
	}
	if ownerID == "" {
		return Wallet{}, ErrOwnerIDRequired
	}
	if len(currency) != 3 {
		return Wallet{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
	return Wallet{
		id:             id,
		ownerType:      ownerType,
		ownerID:        ownerID,
		currency:       currency,
		balance:        ZeroMoney(currency),
		pendingBalance: ZeroMoney(currency),
		status:         WalletStatusActive,
		version:        1,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// Rehydrate reconstructs a Wallet from persistence. Used by repository mappers.
// Does NOT validate invariants (already enforced at creation time).
func RehydrateWallet(
	id string,
	ownerType OwnerType,
	ownerID, currency string,
	balanceAmount, pendingAmount int64,
	status WalletStatus,
	version int,
	createdAt, updatedAt time.Time,
) Wallet {
	return Wallet{
		id:             id,
		ownerType:      ownerType,
		ownerID:        ownerID,
		currency:       currency,
		balance:        Money{amount: balanceAmount, currency: currency},
		pendingBalance: Money{amount: pendingAmount, currency: currency},
		status:         status,
		version:        version,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// ===== Accessors =====

func (w Wallet) ID() string             { return w.id }
func (w Wallet) OwnerType() OwnerType   { return w.ownerType }
func (w Wallet) OwnerID() string        { return w.ownerID }
func (w Wallet) Currency() string       { return w.currency }
func (w Wallet) Balance() Money         { return w.balance }
func (w Wallet) PendingBalance() Money  { return w.pendingBalance }
func (w Wallet) Status() WalletStatus   { return w.status }
func (w Wallet) Version() int           { return w.version }
func (w Wallet) CreatedAt() time.Time   { return w.createdAt }
func (w Wallet) UpdatedAt() time.Time   { return w.updatedAt }
func (w Wallet) IsActive() bool         { return w.status == WalletStatusActive }
func (w Wallet) IsFrozen() bool         { return w.status == WalletStatusFrozen }
func (w Wallet) IsClosed() bool         { return w.status == WalletStatusClosed }

// ===== Mutations =====
//
// All mutations return a NEW Wallet (immutable pattern). The repository layer
// then persists the new state. Callers must use the returned value, not the
// original.

// Credit adds amount to the wallet's balance.
// Returns error if wallet is closed, amount is not positive, or currency mismatches.
// Frozen wallets CAN be credited (e.g. refund of a previously deducted amount).
func (w Wallet) Credit(amount Money, now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if !amount.IsPositive() {
		return w, fmt.Errorf("%w: credit amount must be positive", ErrInvalidMoneyAmount)
	}
	if amount.Currency() != w.currency {
		return w, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, amount.Currency(), w.currency)
	}
	newBalance, err := w.balance.Add(amount)
	if err != nil {
		return w, err
	}
	w.balance = newBalance
	w.updatedAt = now
	return w, nil
}

// Debit subtracts amount from the wallet's balance.
// Returns error if wallet is frozen/closed, amount is not positive,
// currency mismatches, or insufficient funds.
func (w Wallet) Debit(amount Money, now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if w.IsFrozen() {
		return w, ErrWalletFrozen
	}
	if !amount.IsPositive() {
		return w, fmt.Errorf("%w: debit amount must be positive", ErrInvalidMoneyAmount)
	}
	if amount.Currency() != w.currency {
		return w, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, amount.Currency(), w.currency)
	}
	hasFunds, err := w.balance.GreaterThanOrEqual(amount)
	if err != nil {
		return w, err
	}
	if !hasFunds {
		return w, ErrInsufficientFunds
	}
	newBalance, err := w.balance.Subtract(amount)
	if err != nil {
		return w, err
	}
	w.balance = newBalance
	w.updatedAt = now
	return w, nil
}

// Hold moves funds from balance to pendingBalance (escrow for an ongoing order).
// Useful when an order is placed: money is held until delivery confirmation.
func (w Wallet) Hold(amount Money, now time.Time) (Wallet, error) {
	if w.IsClosed() || w.IsFrozen() {
		return w, fmt.Errorf("%w: cannot hold on a non-active wallet", ErrWalletFrozen)
	}
	if !amount.IsPositive() {
		return w, fmt.Errorf("%w: hold amount must be positive", ErrInvalidMoneyAmount)
	}
	if amount.Currency() != w.currency {
		return w, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, amount.Currency(), w.currency)
	}
	hasFunds, err := w.balance.GreaterThanOrEqual(amount)
	if err != nil {
		return w, err
	}
	if !hasFunds {
		return w, ErrInsufficientFunds
	}
	newBalance, _ := w.balance.Subtract(amount)
	newPending, _ := w.pendingBalance.Add(amount)
	w.balance = newBalance
	w.pendingBalance = newPending
	w.updatedAt = now
	return w, nil
}

// Release moves funds from pendingBalance back to balance (order cancelled).
func (w Wallet) Release(amount Money, now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if !amount.IsPositive() {
		return w, fmt.Errorf("%w: release amount must be positive", ErrInvalidMoneyAmount)
	}
	if amount.Currency() != w.currency {
		return w, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, amount.Currency(), w.currency)
	}
	hasPending, err := w.pendingBalance.GreaterThanOrEqual(amount)
	if err != nil {
		return w, err
	}
	if !hasPending {
		return w, fmt.Errorf("%w: pending balance too low", ErrInsufficientFunds)
	}
	newPending, _ := w.pendingBalance.Subtract(amount)
	newBalance, _ := w.balance.Add(amount)
	w.pendingBalance = newPending
	w.balance = newBalance
	w.updatedAt = now
	return w, nil
}

// Settle deducts from pendingBalance (order delivered, funds captured).
func (w Wallet) Settle(amount Money, now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if !amount.IsPositive() {
		return w, fmt.Errorf("%w: settle amount must be positive", ErrInvalidMoneyAmount)
	}
	if amount.Currency() != w.currency {
		return w, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, amount.Currency(), w.currency)
	}
	hasPending, err := w.pendingBalance.GreaterThanOrEqual(amount)
	if err != nil {
		return w, err
	}
	if !hasPending {
		return w, fmt.Errorf("%w: pending balance too low", ErrInsufficientFunds)
	}
	newPending, _ := w.pendingBalance.Subtract(amount)
	w.pendingBalance = newPending
	w.updatedAt = now
	return w, nil
}

// Freeze transitions the wallet to Frozen status.
func (w Wallet) Freeze(now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if w.IsFrozen() {
		return w, nil // idempotent
	}
	w.status = WalletStatusFrozen
	w.updatedAt = now
	return w, nil
}

// Unfreeze transitions the wallet back to Active status.
func (w Wallet) Unfreeze(now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, ErrWalletClosed
	}
	if w.IsActive() {
		return w, nil // idempotent
	}
	w.status = WalletStatusActive
	w.updatedAt = now
	return w, nil
}

// Close transitions the wallet to Closed status. Balance must be zero.
func (w Wallet) Close(now time.Time) (Wallet, error) {
	if w.IsClosed() {
		return w, nil // idempotent
	}
	if !w.balance.IsZero() || !w.pendingBalance.IsZero() {
		return w, fmt.Errorf("%w: cannot close wallet with non-zero balance", ErrInvalidInput)
	}
	w.status = WalletStatusClosed
	w.updatedAt = now
	return w, nil
}

// BumpVersion increments the version (called by repository after successful update).
func (w Wallet) BumpVersion() Wallet {
	w.version++
	return w
}
