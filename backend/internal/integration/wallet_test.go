//go:build integration

package integration_test

import (
	"context"
	"testing"

	financialport "avex-backend/internal/modules/financial/port"
)

// TestWalletOperations_CreditDebit tests the wallet credit + debit flow.
func TestWalletOperations_CreditDebit(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// 1. Create wallet
	wallet, err := financialMod.Service().CreateWallet(ctx, financialport.CreateWalletInput{
		OwnerType: "user",
		OwnerID:   "wallet-test-user-1",
		Currency:  "EGP",
	})
	if err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("expected initial balance 0, got %d", wallet.Balance)
	}

	// 2. Credit 1000 cents
	_, wAfterCredit, err := financialMod.Service().Credit(ctx, financialport.CreditInput{
		WalletID:       wallet.ID,
		Amount:         1000,
		Currency:       "EGP",
		Category:       "topup",
		ReferenceType:  "topup",
		ReferenceID:    "ref-1",
		IdempotencyKey: "wallet-credit-1",
	})
	if err != nil {
		t.Fatalf("Credit: %v", err)
	}
	if wAfterCredit.Balance != 1000 {
		t.Fatalf("expected balance 1000, got %d", wAfterCredit.Balance)
	}

	// 3. Debit 400 cents
	_, wAfterDebit, err := financialMod.Service().Debit(ctx, financialport.DebitInput{
		WalletID:       wallet.ID,
		Amount:         400,
		Currency:       "EGP",
		Category:       "order_payment",
		ReferenceType:  "order",
		ReferenceID:    "order-1",
		IdempotencyKey: "wallet-debit-1",
	})
	if err != nil {
		t.Fatalf("Debit: %v", err)
	}
	if wAfterDebit.Balance != 600 {
		t.Fatalf("expected balance 600, got %d", wAfterDebit.Balance)
	}

	// 4. Try to debit more than balance — should fail
	_, _, err = financialMod.Service().Debit(ctx, financialport.DebitInput{
		WalletID:       wallet.ID,
		Amount:         10000,
		Currency:       "EGP",
		Category:       "order_payment",
		ReferenceType:  "order",
		ReferenceID:    "order-2",
		IdempotencyKey: "wallet-debit-2",
	})
	if err == nil {
		t.Fatal("expected error for insufficient funds")
	}
	t.Logf("Insufficient funds correctly rejected: %v", err)
}

// TestWalletOperations_Transfer tests the wallet transfer flow.
func TestWalletOperations_Transfer(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	// Create two wallets
	from, _ := financialMod.Service().CreateWallet(ctx, financialport.CreateWalletInput{
		OwnerType: "user", OwnerID: "transfer-from-user", Currency: "EGP",
	})
	to, _ := financialMod.Service().CreateWallet(ctx, financialport.CreateWalletInput{
		OwnerType: "merchant", OwnerID: "transfer-to-merchant", Currency: "EGP",
	})

	// Credit the source wallet
	financialMod.Service().Credit(ctx, financialport.CreditInput{
		WalletID: from.ID, Amount: 2000, Currency: "EGP",
		Category: "topup", IdempotencyKey: "transfer-credit",
	})

	// Transfer 500
	debit, credit, err := financialMod.Service().Transfer(ctx, financialport.TransferInput{
		FromWalletID:   from.ID,
		ToWalletID:     to.ID,
		Amount:         500,
		Currency:       "EGP",
		Category:       "order_payment",
		ReferenceType:  "order",
		ReferenceID:    "order-transfer-1",
		IdempotencyKey: "transfer-1",
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if debit.Type != "debit" {
		t.Errorf("expected debit type, got %s", debit.Type)
	}
	if credit.Type != "credit" {
		t.Errorf("expected credit type, got %s", credit.Type)
	}

	// Verify balances
	fromW, _ := financialMod.Service().GetWallet(ctx, from.ID)
	if fromW.Balance != 1500 {
		t.Errorf("expected from balance 1500, got %d", fromW.Balance)
	}
	toW, _ := financialMod.Service().GetWallet(ctx, to.ID)
	if toW.Balance != 500 {
		t.Errorf("expected to balance 500, got %d", toW.Balance)
	}
}

// TestWalletOperations_FreezeUnfreeze tests freeze + unfreeze.
func TestWalletOperations_FreezeUnfreeze(t *testing.T) {
	cleanupAll(t)
	defer cleanupAll(t)
	ctx := context.Background()

	wallet, _ := financialMod.Service().CreateWallet(ctx, financialport.CreateWalletInput{
		OwnerType: "user", OwnerID: "freeze-test-user", Currency: "EGP",
	})
	financialMod.Service().Credit(ctx, financialport.CreditInput{
		WalletID: wallet.ID, Amount: 1000, Currency: "EGP",
		Category: "topup", IdempotencyKey: "freeze-credit",
	})

	// Freeze
	if err := financialMod.Service().FreezeWallet(ctx, wallet.ID); err != nil {
		t.Fatalf("FreezeWallet: %v", err)
	}

	// Debit should fail (frozen)
	_, _, err := financialMod.Service().Debit(ctx, financialport.DebitInput{
		WalletID: wallet.ID, Amount: 100, Currency: "EGP",
		Category: "order_payment", IdempotencyKey: "frozen-debit",
	})
	if err == nil {
		t.Fatal("expected error for debit on frozen wallet")
	}

	// Unfreeze
	if err := financialMod.Service().UnfreezeWallet(ctx, wallet.ID); err != nil {
		t.Fatalf("UnfreezeWallet: %v", err)
	}

	// Debit should now succeed
	_, _, err = financialMod.Service().Debit(ctx, financialport.DebitInput{
		WalletID: wallet.ID, Amount: 100, Currency: "EGP",
		Category: "order_payment", IdempotencyKey: "unfrozen-debit",
	})
	if err != nil {
		t.Fatalf("debit after unfreeze: %v", err)
	}
}
