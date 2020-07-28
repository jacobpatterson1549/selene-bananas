// Package dbtest contains mock testing utilities.
package dbtest

import (
	"context"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

type (
	// MockDatabase implements the db.Database interface.
	MockDatabase struct {
		// QueryFunc is called by Query.
		QueryFunc func(ctx context.Context, q db.Query) db.Scanner
		// ExecFunc is called by Exec.
		ExecFunc func(ctx context.Context, queries ...db.Query) error
	}

	// MockScanner implements the db.Scanner interface.
	MockScanner struct {
		// ScanFunc is called by Scan.
		ScanFunc func(dest ...interface{}) error
	}
)

// Query calls QueryFunc.
func (d MockDatabase) Query(ctx context.Context, q db.Query) db.Scanner {
	return d.QueryFunc(ctx, q)
}

// Exec calls ExecFunc.
func (d MockDatabase) Exec(ctx context.Context, queries ...db.Query) error {
	return d.ExecFunc(ctx, queries...)
}

// Scan calls ScanFunc.
func (s MockScanner) Scan(dest ...interface{}) error {
	return s.ScanFunc(dest...)
}
