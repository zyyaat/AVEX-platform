// Package domain tests: Wallet aggregate.
package domain

import (
        "testing"
        "time"
)

func TestNewWallet(t *testing.T) {
        now := testNow()
        tests := []struct {
                name      string
                id        string
                ownerType OwnerType
                ownerID   string
                currency  string
                wantErr   error
        }{
                {"valid user", "w1", OwnerTypeUser, "u1", "EGP", nil},
                {"valid driver", "w2", OwnerTypeDriver, "d1", "EGP", nil},
                {"valid merchant", "w3", OwnerTypeMerchant, "m1", "EGP", nil},
                {"empty id", "", OwnerTypeUser, "u1", "EGP", ErrInvalidID},
                {"invalid owner type", "w4", OwnerType("admin"), "a1", "EGP", ErrInvalidOwnerType},
                {"empty owner id", "w5", OwnerTypeUser, "", "EGP", ErrOwnerIDRequired},
                {"short currency", "w6", OwnerTypeUser, "u1", "EG", ErrInvalidCurrency},
        }
        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        w, err := NewWallet(tt.id, tt.ownerType, tt.ownerID, tt.currency, now)
                        if tt.wantErr != nil {
                                if err == nil || !errIs(err, tt.wantErr) {
                                        t.Fatalf("expected %v, got %v", tt.wantErr, err)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                        if w.ID() != tt.id {
                                t.Errorf("id: expected %s, got %s", tt.id, w.ID())
                        }
                        if !w.IsActive() {
                                t.Errorf("expected active status")
                        }
                        if w.Version() != 1 {
                                t.Errorf("expected version 1, got %d", w.Version())
                        }
                })
        }
}

func TestWalletCredit(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)

        amt, _ := NewMoney(1000, "EGP")
        w, err := w.Credit(amt, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if w.Balance().Amount() != 1000 {
                t.Errorf("expected 1000, got %d", w.Balance().Amount())
        }

        // Second credit
        amt2, _ := NewMoney(500, "EGP")
        w, _ = w.Credit(amt2, now)
        if w.Balance().Amount() != 1500 {
                t.Errorf("expected 1500, got %d", w.Balance().Amount())
        }

        // Zero amount
        zero, _ := NewMoney(0, "EGP")
        _, err = w.Credit(zero, now)
        if !errIs(err, ErrInvalidMoneyAmount) {
                t.Fatalf("expected ErrInvalidMoneyAmount, got %v", err)
        }

        // Currency mismatch
        usd, _ := NewMoney(100, "USD")
        _, err = w.Credit(usd, now)
        if !errIs(err, ErrCurrencyMismatch) {
                t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
        }
}

func TestWalletDebit(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)
        amt, _ := NewMoney(1000, "EGP")
        w, _ = w.Credit(amt, now)

        // Debit 300
        deb, _ := NewMoney(300, "EGP")
        w, err := w.Debit(deb, now)
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if w.Balance().Amount() != 700 {
                t.Errorf("expected 700, got %d", w.Balance().Amount())
        }

        // Insufficient funds
        big, _ := NewMoney(1000, "EGP")
        _, err = w.Debit(big, now)
        if !errIs(err, ErrInsufficientFunds) {
                t.Fatalf("expected ErrInsufficientFunds, got %v", err)
        }

        // Frozen wallet cannot be debited
        frozen, _ := w.Freeze(now)
        _, err = frozen.Debit(deb, now)
        if !errIs(err, ErrWalletFrozen) {
                t.Fatalf("expected ErrWalletFrozen, got %v", err)
        }
}

func TestWalletFrozenCanStillBeCredited(t *testing.T) {
        // Refunds can be applied to frozen wallets.
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)
        frozen, _ := w.Freeze(now)

        amt, _ := NewMoney(1000, "EGP")
        w2, err := frozen.Credit(amt, now)
        if err != nil {
                t.Fatalf("credit on frozen wallet failed: %v", err)
        }
        if w2.Balance().Amount() != 1000 {
                t.Errorf("expected 1000, got %d", w2.Balance().Amount())
        }
}

func TestWalletFreezeUnfreeze(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)

        frozen, _ := w.Freeze(now)
        if !frozen.IsFrozen() {
                t.Errorf("expected frozen")
        }

        // Idempotent freeze
        frozen2, _ := frozen.Freeze(now)
        if !frozen2.IsFrozen() {
                t.Errorf("expected still frozen")
        }

        // Unfreeze
        active, _ := frozen.Unfreeze(now)
        if !active.IsActive() {
                t.Errorf("expected active")
        }

        // Idempotent unfreeze
        active2, _ := active.Unfreeze(now)
        if !active2.IsActive() {
                t.Errorf("expected still active")
        }
}

func TestWalletHoldReleaseSettle(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)
        amt, _ := NewMoney(1000, "EGP")
        w, _ = w.Credit(amt, now)

        // Hold 400 for an order
        hold, _ := NewMoney(400, "EGP")
        w, err := w.Hold(hold, now)
        if err != nil {
                t.Fatalf("hold failed: %v", err)
        }
        if w.Balance().Amount() != 600 {
                t.Errorf("balance: expected 600, got %d", w.Balance().Amount())
        }
        if w.PendingBalance().Amount() != 400 {
                t.Errorf("pending: expected 400, got %d", w.PendingBalance().Amount())
        }

        // Insufficient for second hold of 700
        big, _ := NewMoney(700, "EGP")
        _, err = w.Hold(big, now)
        if !errIs(err, ErrInsufficientFunds) {
                t.Fatalf("expected ErrInsufficientFunds, got %v", err)
        }

        // Settle 400 (order delivered)
        w, err = w.Settle(hold, now)
        if err != nil {
                t.Fatalf("settle failed: %v", err)
        }
        if w.Balance().Amount() != 600 {
                t.Errorf("balance after settle: expected 600, got %d", w.Balance().Amount())
        }
        if w.PendingBalance().Amount() != 0 {
                t.Errorf("pending after settle: expected 0, got %d", w.PendingBalance().Amount())
        }

        // New order: hold 200, then release (customer cancelled)
        hold2, _ := NewMoney(200, "EGP")
        w, _ = w.Hold(hold2, now)
        if w.Balance().Amount() != 400 {
                t.Errorf("balance: expected 400, got %d", w.Balance().Amount())
        }
        w, err = w.Release(hold2, now)
        if err != nil {
                t.Fatalf("release failed: %v", err)
        }
        if w.Balance().Amount() != 600 {
                t.Errorf("balance after release: expected 600, got %d", w.Balance().Amount())
        }
        if w.PendingBalance().Amount() != 0 {
                t.Errorf("pending after release: expected 0, got %d", w.PendingBalance().Amount())
        }
}

func TestWalletCloseRequiresZeroBalance(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)
        amt, _ := NewMoney(100, "EGP")
        w, _ = w.Credit(amt, now)

        // Cannot close with non-zero balance
        _, err := w.Close(now)
        if !errIs(err, ErrInvalidInput) {
                t.Fatalf("expected ErrInvalidInput, got %v", err)
        }

        // Drain and close
        w, _ = w.Debit(amt, now)
        closed, err := w.Close(now)
        if err != nil {
                t.Fatalf("close failed: %v", err)
        }
        if !closed.IsClosed() {
                t.Errorf("expected closed")
        }

        // Cannot credit a closed wallet
        _, err = closed.Credit(amt, now)
        if !errIs(err, ErrWalletClosed) {
                t.Fatalf("expected ErrWalletClosed, got %v", err)
        }
}

func TestWalletBumpVersion(t *testing.T) {
        now := testNow()
        w, _ := NewWallet("w1", OwnerTypeUser, "u1", "EGP", now)
        if w.Version() != 1 {
                t.Fatalf("expected version 1")
        }
        w = w.BumpVersion()
        if w.Version() != 2 {
                t.Fatalf("expected version 2")
        }
}

// testNow is a shared fixed time for deterministic tests.
func testNow() time.Time {
        t, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
        return t
}
