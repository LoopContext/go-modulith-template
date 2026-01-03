// Package events provides an optional adapter for the outbox pattern.
// This file is kept for the ContextWithTx helper function.
// The actual outbox integration is handled explicitly in modules when needed.
package events

import (
	"context"
	"database/sql"
)

// ctxKey is a type for context keys.
type ctxKey string

const txKey ctxKey = "events.tx"

// ContextWithTx adds a database transaction to the context.
// This allows modules to detect when they're in a transaction and use outbox if enabled.
func ContextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// TxFromContext extracts the transaction from the context, if present.
func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}
