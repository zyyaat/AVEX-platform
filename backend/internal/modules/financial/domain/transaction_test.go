// Package domain tests: Transaction entity.
package domain

import "testing"

func TestNewTransaction(t *testing.T) {
	now := testNow()
	amt, _ := NewMoney(500, "EGP")

	tests := []struct {
		name      string
		id        string
		walletID  string
		txnType   TransactionType
		category  TransactionCategory
		amount    Money
		wantErr   error
	}{
		{"valid credit", "t1", "w1", TxnTypeCredit, CategoryTopup, amt, nil},
		{"valid debit", "t2", "w1", TxnTypeDebit, CategoryOrderPayment, amt, nil},
		{"empty id", "", "w1", TxnTypeCredit, CategoryTopup, amt, ErrInvalidID},
		{"empty wallet", "t3", "", TxnTypeCredit, CategoryTopup, amt, ErrInvalidInput},
		{"invalid type", "t4", "w1", TransactionType("bogus"), CategoryTopup, amt, ErrInvalidTransactionType},
		{"invalid category", "t5", "w1", TxnTypeCredit, TransactionCategory("bogus"), amt, ErrInvalidTransactionCategory},
		{"zero amount", "t6", "w1", TxnTypeCredit, CategoryTopup, ZeroMoney("EGP"), ErrInvalidMoneyAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewTransaction(tt.id, tt.walletID, tt.txnType, tt.category, tt.amount, RefTypeManual, "ref-1", "desc", nil, "", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tx.Status() != TxnStatusPending {
				t.Errorf("expected pending, got %s", tx.Status())
			}
			if tx.CompletedAt() != nil {
				t.Errorf("expected nil completed_at")
			}
		})
	}
}

func TestTransactionStatusTransitions(t *testing.T) {
	now := testNow()
	amt, _ := NewMoney(500, "EGP")
	tx, _ := NewTransaction("t1", "w1", TxnTypeCredit, CategoryTopup, amt, RefTypeManual, "ref-1", "", nil, "", now)

	// pending -> completed
	tx, err := tx.MarkCompleted(now)
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	if tx.Status() != TxnStatusCompleted {
		t.Errorf("expected completed, got %s", tx.Status())
	}
	if tx.CompletedAt() == nil {
		t.Errorf("expected non-nil completed_at")
	}

	// complete -> completed (idempotent error)
	_, err = tx.MarkCompleted(now)
	if !errIs(err, ErrTransactionAlreadyCompleted) {
		t.Fatalf("expected ErrTransactionAlreadyCompleted, got %v", err)
	}

	// complete -> reversed
	tx, err = tx.MarkReversed()
	if err != nil {
		t.Fatalf("reverse failed: %v", err)
	}
	if tx.Status() != TxnStatusReversed {
		t.Errorf("expected reversed, got %s", tx.Status())
	}

	// reversed -> reversed (error)
	_, err = tx.MarkReversed()
	if !errIs(err, ErrTransactionCannotBeReversed) {
		t.Fatalf("expected ErrTransactionCannotBeReversed, got %v", err)
	}
}

func TestTransactionMarkFailed(t *testing.T) {
	now := testNow()
	amt, _ := NewMoney(500, "EGP")
	tx, _ := NewTransaction("t1", "w1", TxnTypeDebit, CategoryOrderPayment, amt, RefTypeOrder, "ord-1", "", nil, "", now)

	// pending -> failed
	tx, err := tx.MarkFailed(now)
	if err != nil {
		t.Fatalf("fail failed: %v", err)
	}
	if tx.Status() != TxnStatusFailed {
		t.Errorf("expected failed, got %s", tx.Status())
	}

	// failed -> failed (idempotent error)
	_, err = tx.MarkFailed(now)
	if !errIs(err, ErrTransactionAlreadyFailed) {
		t.Fatalf("expected ErrTransactionAlreadyFailed, got %v", err)
	}

	// failed -> completed (invalid)
	_, err = tx.MarkCompleted(now)
	if !errIs(err, ErrTransactionCannotBeReversed) {
		t.Fatalf("expected ErrTransactionCannotBeReversed, got %v", err)
	}
}

func TestTransactionIsCreditDebit(t *testing.T) {
	now := testNow()
	amt, _ := NewMoney(500, "EGP")
	credit, _ := NewTransaction("t1", "w1", TxnTypeCredit, CategoryTopup, amt, RefTypeManual, "", "", nil, "", now)
	if !credit.IsCredit() {
		t.Errorf("expected credit")
	}
	if credit.IsDebit() {
		t.Errorf("did not expect debit")
	}

	debit, _ := NewTransaction("t2", "w1", TxnTypeDebit, CategoryOrderPayment, amt, RefTypeOrder, "o1", "", nil, "", now)
	if !debit.IsDebit() {
		t.Errorf("expected debit")
	}
}
