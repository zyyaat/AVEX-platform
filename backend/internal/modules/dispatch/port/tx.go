// Package port tx: transaction abstraction for the dispatch module.
//
// Mirrors the same pattern as orders/financial modules.
package port

import (
	"context"
	"time"
)

// Executor is an opaque handle (pool or transaction).
type Executor interface{}

// TxRunner executes a function within a database transaction.
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}

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

// PageQuery holds pagination parameters.
type PageQuery struct {
	Limit  int
	Offset int
}

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

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

// Page holds a single page of results plus the total count.
type Page[T any] struct {
	Items  []T
	Total  int64
	Limit  int
	Offset int
}

func (p Page[T]) HasMore() bool {
	return int64(p.Offset+p.Limit) < p.Total
}

// Metadata is a JSON-compatible map for offer metadata.
type Metadata map[string]any

var _ = time.Now
