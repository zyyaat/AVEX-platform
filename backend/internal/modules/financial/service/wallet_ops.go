// Package service wallet_ops: Wallet CRUD, Credit/Debit/Transfer, Freeze/Unfreeze.
package service

import (
	"context"
	"errors"
	"fmt"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/events"
	"avex-backend/internal/modules/financial/port"
)

// ===== CreateWallet =====

func (s *Service) CreateWallet(ctx context.Context, input port.CreateWalletInput) (*port.WalletDTO, error) {
	ownerType := domain.OwnerType(input.OwnerType)
	if input.Currency == "" {
		input.Currency = "EGP"
	}
	now := s.deps.Clock.Now()
	id := s.deps.IDGenerator.NewID()

	wallet, err := domain.NewWallet(id, ownerType, input.OwnerID, input.Currency, now)
	if err != nil {
		return nil, err
	}

	if err := s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		if err := s.deps.Repos.Wallets.Create(ctx, exec, wallet); err != nil {
			return err
		}
		// Publish wallet.created event
		ec := s.eventContext(ctx, port.ActorContext{Type: "system", ID: input.OwnerID})
		envelope, err := events.WalletCreatedEnvelope(port.WalletCreatedPayload{
			WalletID:  wallet.ID(),
			OwnerType: string(wallet.OwnerType()),
			OwnerID:   wallet.OwnerID(),
			Currency:  wallet.Currency(),
		}, ec)
		if err != nil {
			return err
		}
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	}); err != nil {
		return nil, err
	}

	dto := port.ToWalletDTO(wallet)
	return &dto, nil
}

// ===== GetWallet / GetWalletByOwner / ListWalletsByOwner =====

func (s *Service) GetWallet(ctx context.Context, id string) (*port.WalletDTO, error) {
	w, err := s.deps.Repos.Wallets.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	dto := port.ToWalletDTO(*w)
	return &dto, nil
}

func (s *Service) GetWalletByOwner(ctx context.Context, ownerType, ownerID, currency string) (*port.WalletDTO, error) {
	if currency == "" {
		currency = "EGP"
	}
	w, err := s.deps.Repos.Wallets.GetByOwner(ctx, s.pool, domain.OwnerType(ownerType), ownerID, currency)
	if err != nil {
		return nil, err
	}
	dto := port.ToWalletDTO(*w)
	return &dto, nil
}

func (s *Service) ListWalletsByOwner(ctx context.Context, ownerType, ownerID string) ([]port.WalletDTO, error) {
	wallets, err := s.deps.Repos.Wallets.ListByOwner(ctx, s.pool, domain.OwnerType(ownerType), ownerID)
	if err != nil {
		return nil, err
	}
	dtos := make([]port.WalletDTO, 0, len(wallets))
	for _, w := range wallets {
		dtos = append(dtos, port.ToWalletDTO(w))
	}
	return dtos, nil
}

// ===== Credit =====

func (s *Service) Credit(ctx context.Context, input port.CreditInput) (*port.TransactionDTO, *port.WalletDTO, error) {
	if input.Currency == "" {
		input.Currency = "EGP"
	}
	// Idempotency check (before transaction).
	if input.IdempotencyKey != "" {
		existing, err := s.deps.Repos.Transactions.GetByIdempotencyKey(ctx, s.pool, input.IdempotencyKey)
		if err == nil && existing != nil {
			// Return existing transaction + current wallet state.
			w, wErr := s.deps.Repos.Wallets.GetByID(ctx, s.pool, existing.WalletID())
			if wErr != nil {
				return nil, nil, wErr
			}
			tDTO := port.ToTransactionDTO(*existing)
			wDTO := port.ToWalletDTO(*w)
			return &tDTO, &wDTO, nil
		}
		if err != nil && !errors.Is(err, domain.ErrTransactionNotFound) {
			return nil, nil, fmt.Errorf("idempotency check: %w", err)
		}
	}

	amount, err := domain.NewMoney(input.Amount, input.Currency)
	if err != nil {
		return nil, nil, err
	}

	var txDTO *port.TransactionDTO
	var wDTO *port.WalletDTO

	err = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		w, err := s.deps.Repos.Wallets.GetByID(ctx, exec, input.WalletID)
		if err != nil {
			return err
		}
		updated, err := w.Credit(amount, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, updated); err != nil {
			return err
		}

		// Create transaction record (status = completed for credits).
		txnID := s.deps.IDGenerator.NewID()
		txn, err := domain.NewTransaction(
			txnID, updated.ID(), domain.TxnTypeCredit,
			domain.TransactionCategory(input.Category),
			amount,
			domain.ReferenceType(input.ReferenceType),
			input.ReferenceID, input.Description, input.Metadata,
			input.IdempotencyKey,
			s.deps.Clock.Now(),
		)
		if err != nil {
			return err
		}
		// Mark as completed immediately (credit is atomic).
		completedAt := s.deps.Clock.Now()
		txn, _ = txn.MarkCompleted(completedAt)
		if err := s.deps.Repos.Transactions.Create(ctx, exec, txn); err != nil {
			// If duplicate idempotency, re-read.
			if errors.Is(err, domain.ErrDuplicateIdempotencyKey) && input.IdempotencyKey != "" {
				existing, _ := s.deps.Repos.Transactions.GetByIdempotencyKey(ctx, exec, input.IdempotencyKey)
				if existing != nil {
					txDTO = port.ToTransactionDTOPtr(*existing)
					wDTO = port.ToWalletDTOPtr(updated)
					return nil
				}
			}
			return err
		}

		// Publish wallet.credited event
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.WalletCreditedEnvelope(port.WalletCreditedPayload{
			WalletID:      updated.ID(),
			TransactionID: txn.ID(),
			AmountCents:   amount.Amount(),
			Currency:      amount.Currency(),
			Category:      string(txn.Category()),
			ReferenceType: string(txn.ReferenceType()),
			ReferenceID:   txn.ReferenceID(),
			NewBalance:    updated.Balance().Amount(),
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}

		txDTO = port.ToTransactionDTOPtr(txn)
		wDTO = port.ToWalletDTOPtr(updated)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return txDTO, wDTO, nil
}

// ===== Debit =====

func (s *Service) Debit(ctx context.Context, input port.DebitInput) (*port.TransactionDTO, *port.WalletDTO, error) {
	if input.Currency == "" {
		input.Currency = "EGP"
	}
	// Idempotency check.
	if input.IdempotencyKey != "" {
		existing, err := s.deps.Repos.Transactions.GetByIdempotencyKey(ctx, s.pool, input.IdempotencyKey)
		if err == nil && existing != nil {
			w, wErr := s.deps.Repos.Wallets.GetByID(ctx, s.pool, existing.WalletID())
			if wErr != nil {
				return nil, nil, wErr
			}
			tDTO := port.ToTransactionDTO(*existing)
			wDTO := port.ToWalletDTO(*w)
			return &tDTO, &wDTO, nil
		}
		if err != nil && !errors.Is(err, domain.ErrTransactionNotFound) {
			return nil, nil, fmt.Errorf("idempotency check: %w", err)
		}
	}

	amount, err := domain.NewMoney(input.Amount, input.Currency)
	if err != nil {
		return nil, nil, err
	}

	var txDTO *port.TransactionDTO
	var wDTO *port.WalletDTO

	err = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		w, err := s.deps.Repos.Wallets.GetByID(ctx, exec, input.WalletID)
		if err != nil {
			return err
		}
		updated, err := w.Debit(amount, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, updated); err != nil {
			return err
		}

		txnID := s.deps.IDGenerator.NewID()
		txn, err := domain.NewTransaction(
			txnID, updated.ID(), domain.TxnTypeDebit,
			domain.TransactionCategory(input.Category),
			amount,
			domain.ReferenceType(input.ReferenceType),
			input.ReferenceID, input.Description, input.Metadata,
			input.IdempotencyKey,
			s.deps.Clock.Now(),
		)
		if err != nil {
			return err
		}
		completedAt := s.deps.Clock.Now()
		txn, _ = txn.MarkCompleted(completedAt)
		if err := s.deps.Repos.Transactions.Create(ctx, exec, txn); err != nil {
			if errors.Is(err, domain.ErrDuplicateIdempotencyKey) && input.IdempotencyKey != "" {
				existing, _ := s.deps.Repos.Transactions.GetByIdempotencyKey(ctx, exec, input.IdempotencyKey)
				if existing != nil {
					txDTO = port.ToTransactionDTOPtr(*existing)
					wDTO = port.ToWalletDTOPtr(updated)
					return nil
				}
			}
			return err
		}

		// Publish wallet.debited event
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.WalletDebitedEnvelope(port.WalletDebitedPayload{
			WalletID:      updated.ID(),
			TransactionID: txn.ID(),
			AmountCents:   amount.Amount(),
			Currency:      amount.Currency(),
			Category:      string(txn.Category()),
			ReferenceType: string(txn.ReferenceType()),
			ReferenceID:   txn.ReferenceID(),
			NewBalance:    updated.Balance().Amount(),
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}

		txDTO = port.ToTransactionDTOPtr(txn)
		wDTO = port.ToWalletDTOPtr(updated)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return txDTO, wDTO, nil
}

// ===== Transfer =====

func (s *Service) Transfer(ctx context.Context, input port.TransferInput) (*port.TransactionDTO, *port.TransactionDTO, error) {
	if input.Currency == "" {
		input.Currency = "EGP"
	}
	if input.FromWalletID == input.ToWalletID {
		return nil, nil, fmt.Errorf("%w: from and to wallets cannot be the same", domain.ErrInvalidInput)
	}

	amount, err := domain.NewMoney(input.Amount, input.Currency)
	if err != nil {
		return nil, nil, err
	}

	var debitDTO, creditDTO *port.TransactionDTO

	err = s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		from, err := s.deps.Repos.Wallets.GetByID(ctx, exec, input.FromWalletID)
		if err != nil {
			return err
		}
		to, err := s.deps.Repos.Wallets.GetByID(ctx, exec, input.ToWalletID)
		if err != nil {
			return err
		}
		// Currency check
		if from.Currency() != to.Currency() {
			return fmt.Errorf("%w: cannot transfer between %s and %s wallets", domain.ErrCurrencyMismatch, from.Currency(), to.Currency())
		}

		// Debit source
		debitedFrom, err := from.Debit(amount, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, debitedFrom); err != nil {
			return err
		}

		// Credit destination
		creditedTo, err := to.Credit(amount, s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, creditedTo); err != nil {
			return err
		}

		// Create both transaction records linked by the same reference_id.
		debitTxnID := s.deps.IDGenerator.NewID()
		creditTxnID := s.deps.IDGenerator.NewID()

		debitTxn, err := domain.NewTransaction(
			debitTxnID, debitedFrom.ID(), domain.TxnTypeDebit,
			domain.TransactionCategory(input.Category),
			amount,
			domain.ReferenceType(input.ReferenceType),
			input.ReferenceID, input.Description, input.Metadata,
			input.IdempotencyKey+"-debit",
			s.deps.Clock.Now(),
		)
		if err != nil {
			return err
		}
		completedAt := s.deps.Clock.Now()
		debitTxn, _ = debitTxn.MarkCompleted(completedAt)
		if err := s.deps.Repos.Transactions.Create(ctx, exec, debitTxn); err != nil {
			return err
		}

		creditTxn, err := domain.NewTransaction(
			creditTxnID, creditedTo.ID(), domain.TxnTypeCredit,
			domain.TransactionCategory(input.Category),
			amount,
			domain.ReferenceType(input.ReferenceType),
			input.ReferenceID, input.Description, input.Metadata,
			input.IdempotencyKey+"-credit",
			s.deps.Clock.Now(),
		)
		if err != nil {
			return err
		}
		creditTxn, _ = creditTxn.MarkCompleted(completedAt)
		if err := s.deps.Repos.Transactions.Create(ctx, exec, creditTxn); err != nil {
			return err
		}

		// Publish transfer.completed event
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.TransferCompletedEnvelope(port.TransferCompletedPayload{
			FromWalletID:        debitedFrom.ID(),
			ToWalletID:          creditedTo.ID(),
			AmountCents:         amount.Amount(),
			Currency:            amount.Currency(),
			Category:            string(debitTxn.Category()),
			ReferenceType:       string(debitTxn.ReferenceType()),
			ReferenceID:         debitTxn.ReferenceID(),
			DebitTransactionID:  debitTxn.ID(),
			CreditTransactionID: creditTxn.ID(),
		}, ec)
		if err != nil {
			return err
		}
		if err := s.deps.EventPublisher.Publish(ctx, exec, envelope); err != nil {
			return err
		}

		debitDTO = port.ToTransactionDTOPtr(debitTxn)
		creditDTO = port.ToTransactionDTOPtr(creditTxn)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return debitDTO, creditDTO, nil
}

// ===== Freeze / Unfreeze =====

func (s *Service) FreezeWallet(ctx context.Context, id string) error {
	return s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		w, err := s.deps.Repos.Wallets.GetByID(ctx, exec, id)
		if err != nil {
			return err
		}
		frozen, err := w.Freeze(s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, frozen); err != nil {
			return err
		}
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.WalletFrozenEnvelope(port.WalletFrozenPayload{
			WalletID: frozen.ID(),
			OwnerID:  frozen.OwnerID(),
		}, ec)
		if err != nil {
			return err
		}
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	})
}

func (s *Service) UnfreezeWallet(ctx context.Context, id string) error {
	return s.deps.TxRunner.WithinTx(ctx, func(ctx context.Context, exec port.Executor) error {
		w, err := s.deps.Repos.Wallets.GetByID(ctx, exec, id)
		if err != nil {
			return err
		}
		active, err := w.Unfreeze(s.deps.Clock.Now())
		if err != nil {
			return err
		}
		if err := s.deps.Repos.Wallets.Update(ctx, exec, active); err != nil {
			return err
		}
		ec := s.eventContext(ctx, port.ActorContext{Type: "system"})
		envelope, err := events.WalletUnfrozenEnvelope(port.WalletUnfrozenPayload{
			WalletID: active.ID(),
			OwnerID:  active.OwnerID(),
		}, ec)
		if err != nil {
			return err
		}
		return s.deps.EventPublisher.Publish(ctx, exec, envelope)
	})
}

// ===== Transaction Queries =====

func (s *Service) GetTransaction(ctx context.Context, id string) (*port.TransactionDTO, error) {
	t, err := s.deps.Repos.Transactions.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	dto := port.ToTransactionDTO(*t)
	return &dto, nil
}

func (s *Service) ListTransactionsByWallet(ctx context.Context, walletID string, page port.PageQuery) (port.Page[port.TransactionDTO], error) {
	result, err := s.deps.Repos.Transactions.ListByWallet(ctx, s.pool, walletID, page)
	if err != nil {
		return port.Page[port.TransactionDTO]{}, err
	}
	dtos := make([]port.TransactionDTO, 0, len(result.Items))
	for _, t := range result.Items {
		dtos = append(dtos, port.ToTransactionDTO(t))
	}
	return port.Page[port.TransactionDTO]{
		Items:  dtos,
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
	}, nil
}
