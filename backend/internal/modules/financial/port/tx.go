// Package port tx: transaction abstraction for the financial module.
//
// Mirrors the orders module's port/tx.go design:
//   - Executor is an opaque handle (pool or transaction)
//   - TxRunner runs a function within a transaction
//   - Row / Rows are minimal scanning interfaces (decouple from pgx)
//   - PageQuery / Page are pagination helpers
//   - Metadata is a JSON-compatible map for transaction metadata
package port

import (
	"context"
	"time"
)

// ===== Transaction Abstraction =====

// Executor is an opaque handle to either a database connection pool or an
// active transaction. Repository methods accept it explicitly so transaction
// boundaries are visible at the call site.
type Executor interface{}

// TxRunner executes a function within a database transaction.
//
// Semantics:
//   - If fn returns nil, the transaction is committed.
//   - If fn returns a non-nil error, the transaction is rolled back.
//   - The Executor passed to fn is valid only for the duration of fn.
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}

// ===== Minimal Row/Rows Interfaces =====

// Row represents a single database row for scanning.
type Row interface {
	Scan(dest ...any) error
}

// Rows represents a cursor of database rows for iteration.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

// ===== Pagination =====

// PageQuery holds pagination parameters for list queries.
type PageQuery struct {
	Limit  int
	Offset int
}

// Pagination defaults.
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

// Normalize returns a PageQuery with defaults applied and values clamped
// to valid ranges. Always call this before passing PageQuery to a repo.
func (p PageQuery) Normalize() PageQuery {
	if p.Limit <= 0 {
		p.Limit = DefaultPageLimit
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// Page holds a single page of results plus the total count (for UI paging).
type Page[T any] struct {
	Items  []T
	Total  int64
	Limit  int
	Offset int
}

// HasMore reports whether there are more items beyond this page.
func (p Page[T]) HasMore() bool {
	return int64(p.Offset+p.Limit) < p.Total
}

// NextPage returns the PageQuery for the next page, or the zero value
// if there is no next page.
func (p Page[T]) NextPage() PageQuery {
	if !p.HasMore() {
		return PageQuery{}
	}
	return PageQuery{Limit: p.Limit, Offset: p.Offset + p.Limit}
}

// ===== Metadata Type =====

// Metadata is a JSON-compatible map for transaction metadata.
// Used to store contextual data like order_id, surge_zone_id, promo_code.
type Metadata map[string]any

// ===== Time Alias =====
//
// Re-exported here so callers can import port only without importing time
// separately for method signatures that use time.Time.
var _ = time.Now
