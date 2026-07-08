// Package service tests: service-layer unit tests using mock repositories.
package service

import (
        "context"
        "errors"
        "testing"
        "time"

        "avex-backend/internal/modules/financial/domain"
        "avex-backend/internal/modules/financial/port"
)

// ===== Mock Infrastructure =====

type mockClock struct{ t time.Time }

func (m *mockClock) Now() time.Time { return m.t }

type mockIDGen struct{ counter int }

func (m *mockIDGen) NewID() string {
        m.counter++
        return "id-" + time.Now().Format("150405.000000") + "-" + itoa(m.counter)
}

func itoa(n int) string {
        if n == 0 {
                return "0"
        }
        var buf [20]byte
        i := len(buf)
        for n > 0 {
                i--
                buf[i] = byte('0' + n%10)
                n /= 10
        }
        return string(buf[i:])
}

type mockLogger struct{}

func (mockLogger) Debug(string, ...any) {}
func (mockLogger) Info(string, ...any)  {}
func (mockLogger) Warn(string, ...any)  {}
func (mockLogger) Error(string, ...any) {}

type mockTxRunner struct{ committed bool }

func (m *mockTxRunner) WithinTx(_ context.Context, fn func(ctx context.Context, exec port.Executor) error) error {
        m.committed = true
        return fn(context.Background(), nil)
}

type mockEventPublisher struct{ published []port.EventEnvelope }

func (m *mockEventPublisher) Publish(_ context.Context, _ port.Executor, env port.EventEnvelope) error {
        m.published = append(m.published, env)
        return nil
}

// ===== Mock Repositories =====

type mockWalletRepo struct {
        wallets    map[string]*domain.Wallet
        byOwnerKey map[string]string // "user:user-1:EGP" -> wallet ID
}

func newMockWalletRepo() *mockWalletRepo {
        return &mockWalletRepo{
                wallets:    make(map[string]*domain.Wallet),
                byOwnerKey: make(map[string]string),
        }
}

func ownerKey(ot domain.OwnerType, oid, cur string) string {
        return string(ot) + ":" + oid + ":" + cur
}

func (r *mockWalletRepo) Create(_ context.Context, _ port.Executor, w domain.Wallet) error {
        if _, ok := r.byOwnerKey[ownerKey(w.OwnerType(), w.OwnerID(), w.Currency())]; ok {
                return domain.ErrWalletAlreadyExists
        }
        cp := w
        r.wallets[w.ID()] = &cp
        r.byOwnerKey[ownerKey(w.OwnerType(), w.OwnerID(), w.Currency())] = w.ID()
        return nil
}

func (r *mockWalletRepo) GetByID(_ context.Context, _ port.Executor, id string) (*domain.Wallet, error) {
        w, ok := r.wallets[id]
        if !ok {
                return nil, domain.ErrWalletNotFound
        }
        cp := *w
        return &cp, nil
}

func (r *mockWalletRepo) GetByOwner(_ context.Context, _ port.Executor, ot domain.OwnerType, oid, cur string) (*domain.Wallet, error) {
        for _, w := range r.wallets {
                if w.OwnerType() == ot && w.OwnerID() == oid && w.Currency() == cur {
                        cp := *w
                        return &cp, nil
                }
        }
        return nil, domain.ErrWalletNotFound
}

func (r *mockWalletRepo) Update(_ context.Context, _ port.Executor, w domain.Wallet) error {
        cp := w
        r.wallets[w.ID()] = &cp
        return nil
}

func (r *mockWalletRepo) UpdateBalanceAndStatus(ctx context.Context, exec port.Executor, w domain.Wallet) error {
        return r.Update(ctx, exec, w)
}

func (r *mockWalletRepo) ListByOwner(_ context.Context, _ port.Executor, ot domain.OwnerType, oid string) ([]domain.Wallet, error) {
        var out []domain.Wallet
        for _, w := range r.wallets {
                if w.OwnerType() == ot && w.OwnerID() == oid {
                        out = append(out, *w)
                }
        }
        return out, nil
}

// ===== Mock Transaction Repo =====

type mockTxnRepo struct {
        txns       map[string]*domain.Transaction
        byIdemKey  map[string]string
}

func newMockTxnRepo() *mockTxnRepo {
        return &mockTxnRepo{
                txns:      make(map[string]*domain.Transaction),
                byIdemKey: make(map[string]string),
        }
}

func (r *mockTxnRepo) Create(_ context.Context, _ port.Executor, txn domain.Transaction) error {
        if txn.IdempotencyKey() != "" {
                if _, ok := r.byIdemKey[txn.IdempotencyKey()]; ok {
                        return domain.ErrDuplicateIdempotencyKey
                }
                r.byIdemKey[txn.IdempotencyKey()] = txn.ID()
        }
        cp := txn
        r.txns[txn.ID()] = &cp
        return nil
}

func (r *mockTxnRepo) GetByID(_ context.Context, _ port.Executor, id string) (*domain.Transaction, error) {
        t, ok := r.txns[id]
        if !ok {
                return nil, domain.ErrTransactionNotFound
        }
        cp := *t
        return &cp, nil
}

func (r *mockTxnRepo) GetByIdempotencyKey(_ context.Context, _ port.Executor, key string) (*domain.Transaction, error) {
        if key == "" {
                return nil, domain.ErrTransactionNotFound
        }
        id, ok := r.byIdemKey[key]
        if !ok {
                return nil, domain.ErrTransactionNotFound
        }
        cp := *r.txns[id]
        return &cp, nil
}

func (r *mockTxnRepo) UpdateStatus(_ context.Context, _ port.Executor, id string, status domain.TransactionStatus, completedAt *time.Time) error {
        t, ok := r.txns[id]
        if !ok {
                return domain.ErrTransactionNotFound
        }
        // Mutate via rehydrate (mock only).
        *t = domain.RehydrateTransaction(
                t.ID(), t.WalletID(), t.Type(), t.Category(), t.Amount(),
                status, t.ReferenceType(), t.ReferenceID(), t.Description(),
                t.Metadata(), t.IdempotencyKey(), t.CreatedAt(), completedAt,
        )
        return nil
}

func (r *mockTxnRepo) ListByWallet(_ context.Context, _ port.Executor, walletID string, page port.PageQuery) (port.Page[domain.Transaction], error) {
        var items []domain.Transaction
        for _, t := range r.txns {
                if t.WalletID() == walletID {
                        items = append(items, *t)
                }
        }
        return port.Page[domain.Transaction]{Items: items, Total: int64(len(items)), Limit: page.Limit, Offset: page.Offset}, nil
}

func (r *mockTxnRepo) ListByReference(_ context.Context, _ port.Executor, refType domain.ReferenceType, refID string) ([]domain.Transaction, error) {
        var items []domain.Transaction
        for _, t := range r.txns {
                if t.ReferenceType() == refType && t.ReferenceID() == refID {
                        items = append(items, *t)
                }
        }
        return items, nil
}

// ===== Mock Outbox Repo (no-op) =====

type mockOutboxRepo struct{}

func (mockOutboxRepo) Save(_ context.Context, _ port.Executor, _ port.EventEnvelope) error { return nil }
func (mockOutboxRepo) GetPending(_ context.Context, _ port.Executor, _ int) ([]port.EventEnvelope, error) {
        return nil, nil
}
func (mockOutboxRepo) MarkPublished(_ context.Context, _ port.Executor, _ string) error { return nil }

// ===== Setup Helper =====

func newTestService(t *testing.T) (*Service, *mockWalletRepo, *mockTxnRepo, *mockEventPublisher) {
        t.Helper()
        walletRepo := newMockWalletRepo()
        txnRepo := newMockTxnRepo()
        repoSet := port.RepositorySet{
                Wallets:      walletRepo,
                Transactions: txnRepo,
                Outbox:       mockOutboxRepo{},
        }
        deps := port.Deps{
                Clock:          &mockClock{t: time.Now().UTC()},
                IDGenerator:    &mockIDGen{},
                EventPublisher: &mockEventPublisher{},
                Logger:         mockLogger{},
                TxRunner:       &mockTxRunner{},
                Repos:          repoSet,
        }
        svc := New(deps, nil)
        return svc, walletRepo, txnRepo, deps.EventPublisher.(*mockEventPublisher)
}

// ===== Tests =====

func TestCreateWalletSuccess(t *testing.T) {
        svc, walletRepo, _, pub := newTestService(t)
        dto, err := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user",
                OwnerID:   "user-1",
                Currency:  "EGP",
        })
        if err != nil {
                t.Fatalf("unexpected error: %v", err)
        }
        if dto.OwnerID != "user-1" {
                t.Errorf("owner_id: %s", dto.OwnerID)
        }
        if dto.Currency != "EGP" {
                t.Errorf("currency: %s", dto.Currency)
        }
        if dto.Balance != 0 {
                t.Errorf("expected balance 0, got %d", dto.Balance)
        }
        if dto.Status != "active" {
                t.Errorf("status: %s", dto.Status)
        }
        // Verify persisted
        if len(walletRepo.wallets) != 1 {
                t.Errorf("expected 1 wallet in repo, got %d", len(walletRepo.wallets))
        }
        // Verify event published
        if len(pub.published) != 1 {
                t.Errorf("expected 1 event, got %d", len(pub.published))
        }
        if pub.published[0].EventType != port.EventWalletCreated {
                t.Errorf("event type: %s", pub.published[0].EventType)
        }
}

func TestCreateWalletDuplicate(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        _, err := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        if err != nil {
                t.Fatalf("first create failed: %v", err)
        }
        _, err = svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        if !errors.Is(err, domain.ErrWalletAlreadyExists) {
                t.Fatalf("expected ErrWalletAlreadyExists, got %v", err)
        }
}

func TestCreateWalletInvalidOwnerType(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        _, err := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "bogus", OwnerID: "user-1", Currency: "EGP",
        })
        if !errors.Is(err, domain.ErrInvalidOwnerType) {
                t.Fatalf("expected ErrInvalidOwnerType, got %v", err)
        }
}

func TestCreditDebitFlow(t *testing.T) {
        svc, _, _, pub := newTestService(t)

        // Create wallet
        wallet, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })

        // Credit 1000 cents
        pub.published = nil
        _, wAfterCredit, err := svc.Credit(context.Background(), port.CreditInput{
                WalletID:       wallet.ID,
                Amount:         1000,
                Currency:       "EGP",
                Category:       "topup",
                ReferenceType:  "topup",
                ReferenceID:    "ref-1",
                IdempotencyKey: "idem-credit-1",
        })
        if err != nil {
                t.Fatalf("credit failed: %v", err)
        }
        if wAfterCredit.Balance != 1000 {
                t.Errorf("expected balance 1000, got %d", wAfterCredit.Balance)
        }
        // Verify event
        if len(pub.published) != 1 || pub.published[0].EventType != port.EventWalletCredited {
                t.Errorf("expected 1 credited event, got %v", pub.published)
        }

        // Debit 400 cents
        pub.published = nil
        _, wAfterDebit, err := svc.Debit(context.Background(), port.DebitInput{
                WalletID:       wallet.ID,
                Amount:         400,
                Currency:       "EGP",
                Category:       "order_payment",
                ReferenceType:  "order",
                ReferenceID:    "ord-1",
                IdempotencyKey: "idem-debit-1",
        })
        if err != nil {
                t.Fatalf("debit failed: %v", err)
        }
        if wAfterDebit.Balance != 600 {
                t.Errorf("expected balance 600, got %d", wAfterDebit.Balance)
        }

        // Idempotency: re-call credit with same key — should return same tx + wallet state.
        pub.published = nil
        tx2, w2, err := svc.Credit(context.Background(), port.CreditInput{
                WalletID:       wallet.ID,
                Amount:         9999, // different amount, should be IGNORED due to idempotency
                Currency:       "EGP",
                Category:       "topup",
                IdempotencyKey: "idem-credit-1",
        })
        if err != nil {
                t.Fatalf("idempotent credit failed: %v", err)
        }
        if tx2.Amount != 1000 {
                t.Errorf("expected original amount 1000, got %d", tx2.Amount)
        }
        if w2.Balance != 600 {
                t.Errorf("balance should be unchanged (600), got %d", w2.Balance)
        }
        if len(pub.published) != 0 {
                t.Errorf("idempotent call should not publish events, got %d", len(pub.published))
        }
}

func TestDebitInsufficientFunds(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        wallet, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        _, _, err := svc.Debit(context.Background(), port.DebitInput{
                WalletID:       wallet.ID,
                Amount:         100,
                Currency:       "EGP",
                Category:       "order_payment",
                IdempotencyKey: "idem-debit-x",
        })
        if !errors.Is(err, domain.ErrInsufficientFunds) {
                t.Fatalf("expected ErrInsufficientFunds, got %v", err)
        }
}

func TestFreezeUnfreezeWallet(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        wallet, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        // Credit some money so we can verify the freeze blocks debit
        svc.Credit(context.Background(), port.CreditInput{
                WalletID: wallet.ID, Amount: 1000, Currency: "EGP", Category: "topup", IdempotencyKey: "cr1",
        })

        // Freeze
        if err := svc.FreezeWallet(context.Background(), wallet.ID); err != nil {
                t.Fatalf("freeze failed: %v", err)
        }

        // Debit should now fail with ErrWalletFrozen
        _, _, err := svc.Debit(context.Background(), port.DebitInput{
                WalletID:       wallet.ID,
                Amount:         100,
                Currency:       "EGP",
                Category:       "order_payment",
                IdempotencyKey: "debit-after-freeze",
        })
        if !errors.Is(err, domain.ErrWalletFrozen) {
                t.Fatalf("expected ErrWalletFrozen, got %v", err)
        }

        // Unfreeze
        if err := svc.UnfreezeWallet(context.Background(), wallet.ID); err != nil {
                t.Fatalf("unfreeze failed: %v", err)
        }

        // Debit should now succeed
        _, _, err = svc.Debit(context.Background(), port.DebitInput{
                WalletID:       wallet.ID,
                Amount:         100,
                Currency:       "EGP",
                Category:       "order_payment",
                IdempotencyKey: "debit-after-unfreeze",
        })
        if err != nil {
                t.Fatalf("debit after unfreeze failed: %v", err)
        }
}

func TestTransferFlow(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        from, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        to, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "merchant", OwnerID: "merchant-1", Currency: "EGP",
        })
        // Credit from-wallet
        svc.Credit(context.Background(), port.CreditInput{
                WalletID: from.ID, Amount: 2000, Currency: "EGP", Category: "topup", IdempotencyKey: "tr-cr",
        })

        debit, credit, err := svc.Transfer(context.Background(), port.TransferInput{
                FromWalletID:   from.ID,
                ToWalletID:     to.ID,
                Amount:         500,
                Currency:       "EGP",
                Category:       "order_payment",
                ReferenceType:  "order",
                ReferenceID:    "ord-1",
                IdempotencyKey: "tr-1",
        })
        if err != nil {
                t.Fatalf("transfer failed: %v", err)
        }
        if debit.Type != "debit" {
                t.Errorf("expected debit type, got %s", debit.Type)
        }
        if credit.Type != "credit" {
                t.Errorf("expected credit type, got %s", credit.Type)
        }

        // Verify balances
        fromW, _ := svc.GetWallet(context.Background(), from.ID)
        if fromW.Balance != 1500 {
                t.Errorf("from balance: expected 1500, got %d", fromW.Balance)
        }
        toW, _ := svc.GetWallet(context.Background(), to.ID)
        if toW.Balance != 500 {
                t.Errorf("to balance: expected 500, got %d", toW.Balance)
        }
}

func TestTransferSameWalletFails(t *testing.T) {
        svc, _, _, _ := newTestService(t)
        w, _ := svc.CreateWallet(context.Background(), port.CreateWalletInput{
                OwnerType: "user", OwnerID: "user-1", Currency: "EGP",
        })
        _, _, err := svc.Transfer(context.Background(), port.TransferInput{
                FromWalletID:   w.ID,
                ToWalletID:     w.ID,
                Amount:         100,
                Currency:       "EGP",
                Category:       "order_payment",
                IdempotencyKey: "tr-same",
        })
        if !errors.Is(err, domain.ErrInvalidInput) {
                t.Fatalf("expected ErrInvalidInput, got %v", err)
        }
}
