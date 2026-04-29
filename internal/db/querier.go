package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is implemented by *Queries and QueriesWrapper (which supports WithTx returning Querier).
type Querier interface {
	CountTransactionsByFan(ctx context.Context, fanID pgtype.UUID) (int64, error)
	InsertTransaction(ctx context.Context, arg InsertTransactionParams) (Transaction, error)
	ListTransactionsByFan(ctx context.Context, arg ListTransactionsByFanParams) ([]Transaction, error)

	GetBalance(ctx context.Context, creatorID pgtype.UUID) (Balance, error)
	UpsertBalance(ctx context.Context, arg UpsertBalanceParams) (Balance, error)
	DecrementBalance(ctx context.Context, arg DecrementBalanceParams) (Balance, error)

	InsertPayout(ctx context.Context, arg InsertPayoutParams) (Payout, error)
	CountPayoutsByCreator(ctx context.Context, creatorID pgtype.UUID) (int64, error)
	ListPayoutsByCreator(ctx context.Context, arg ListPayoutsByCreatorParams) ([]Payout, error)

	GetRevenue(ctx context.Context, arg GetRevenueParams) (pgtype.Numeric, error)
	GetRevenueByTier(ctx context.Context, arg GetRevenueByTierParams) ([]GetRevenueByTierRow, error)

	WithTx(tx pgx.Tx) Querier
}

// QueriesWrapper wraps *Queries so that WithTx returns Querier (interface) rather than *Queries.
type QueriesWrapper struct {
	*Queries
}

func NewQuerier(dbtx DBTX) Querier {
	return QueriesWrapper{New(dbtx)}
}

func (w QueriesWrapper) WithTx(tx pgx.Tx) Querier {
	return QueriesWrapper{w.Queries.WithTx(tx)}
}
