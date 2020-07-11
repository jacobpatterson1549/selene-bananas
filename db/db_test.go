package db

import (
	"context"
)

type (
	mockDatabase struct {
		queryFunc func(ctx context.Context, q query) scanner
		execFunc  func(ctx context.Context, queries ...query) error
	}

	mockScanner struct {
		ScanFunc func(dest ...interface{}) error
	}
)

func (d mockDatabase) query(ctx context.Context, q query) scanner {
	return d.queryFunc(ctx, q)
}

func (d mockDatabase) exec(ctx context.Context, queries ...query) error {
	return d.execFunc(ctx, queries...)
}

func (s mockScanner) Scan(dest ...interface{}) error {
	return s.ScanFunc(dest...)
}
