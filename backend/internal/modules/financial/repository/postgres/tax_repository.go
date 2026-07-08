// Package postgres tax_repository: TaxRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/financial/domain"
	"avex-backend/internal/modules/financial/port"
)

// TaxRepository implements port.TaxRepository.
type TaxRepository struct{}

var _ port.TaxRepository = (*TaxRepository)(nil)

// Create inserts a new tax rule.
func (r *TaxRepository) Create(ctx context.Context, exec port.Executor, tax domain.Tax) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO financial.taxes (id, name, rate, applies_to, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		tax.ID(), tax.Name(), tax.Rate(), tax.AppliesTo(), tax.IsActive(), tax.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("tax create: %w", err)
	}
	return nil
}

// GetByID retrieves a tax by UUID.
func (r *TaxRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Tax, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+taxColumns+` FROM financial.taxes WHERE id = $1`, id)
	t, err := scanTax(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: tax %s", domain.ErrInvalidID, id)
		}
		return nil, fmt.Errorf("tax read: %w", err)
	}
	return &t, nil
}

// ListActive retrieves all active tax rules.
func (r *TaxRepository) ListActive(ctx context.Context, exec port.Executor) ([]domain.Tax, error) {
	dbtx := toDBTX(exec)
	rows, err := dbtx.Query(ctx, `
		SELECT `+taxColumns+`
		FROM financial.taxes
		WHERE is_active = TRUE
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list taxes: %w", err)
	}
	defer rows.Close()

	var items []domain.Tax
	for rows.Next() {
		t, err := scanTax(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}

// ===== Mapper Helpers =====

const taxColumns = `id, name, rate, applies_to, is_active, created_at`

func scanTax(s scanner) (domain.Tax, error) {
	var (
		id, name, appliesTo string
		rate                float64
		isActive            bool
		createdAt           time.Time
	)
	if err := s.Scan(&id, &name, &rate, &appliesTo, &isActive, &createdAt); err != nil {
		return domain.Tax{}, err
	}
	return domain.RehydrateTax(id, name, rate, appliesTo, isActive, createdAt), nil
}
