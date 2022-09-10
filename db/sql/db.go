// Package sql implements a SQL database.
package sql

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

type (
	// Database is a SQL db.Database with additional configuration
	Database struct {
		DB *sql.DB
		db.Config
	}
)

// ErrNoRows is returned by the Scanner when there are no rows to scan.
var ErrNoRows = sql.ErrNoRows

// Setup initializes the database by reading the files and executing their contents as raw queries.
func (db Database) Setup(ctx context.Context, files []io.Reader) error {
	ctx, cancelFunc := context.WithTimeout(ctx, db.QueryPeriod)
	defer cancelFunc()
	queries := make([]Query, len(files))
	for i, f := range files {
		b, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading sql setup query %v: %w", i, err)
		}
		queries[i] = RawQuery(b)
	}
	if err := db.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("running setup queries %w", err)
	}
	return nil
}

// Query queries a single row, scanning into the destination array.
func (db Database) Query(ctx context.Context, q Query, dest ...interface{}) error {
	ctx, cancelFunc := context.WithTimeout(ctx, db.QueryPeriod)
	defer cancelFunc()
	row := db.DB.QueryRowContext(ctx, q.Cmd(), q.Args()...)
	if err := row.Scan(dest...); err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return fmt.Errorf("querying into destination arguments: %w", err)
	}
	return nil
}

// Exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (db Database) Exec(ctx context.Context, queries ...Query) error {
	ctx, cancelFunc := context.WithTimeout(ctx, db.QueryPeriod)
	defer cancelFunc()
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	for i, q := range queries {
		result, err := tx.ExecContext(ctx, q.Cmd(), q.Args()...)
		if f, ok := q.(ExecFunction); err == nil && ok {
			var n int64
			n, err = result.RowsAffected()
			if err == nil && n != 1 {
				err = fmt.Errorf("wanted to update 1 row, but updated %d when calling %s", n, f.name)
			}
		}
		if err != nil {
			err = fmt.Errorf("executing query %v: %w", i, err)
			err2 := tx.Rollback()
			if err2 != nil {
				return fmt.Errorf("rolling back transaction due to %v: %w", err, err2)
			}
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
