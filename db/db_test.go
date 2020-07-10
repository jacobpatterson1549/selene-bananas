package db

import (
	"context"
	"database/sql"
)

type (
	mockDatabase struct {
		queryRowFunc        func(ctx context.Context, q sqlQuery) row
		execFunc            func(ctx context.Context, q sqlQuery) (sql.Result, error)
		execTransactionFunc func(ctx context.Context, queries []sqlQuery) error
	}
)

func (d mockDatabase) queryRow(ctx context.Context, q sqlQuery) row {
	return d.queryRowFunc(ctx, q)
}

func (d mockDatabase) exec(ctx context.Context, q sqlQuery) (sql.Result, error) {
	return d.execFunc(ctx, q)
}

func (d mockDatabase) execTransaction(ctx context.Context, queries []sqlQuery) error {
	return d.execTransactionFunc(ctx, queries)
}
