// Package sql implements a SQL database.
package sql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

type (
	// Database is a SQL db.Database.
	Database struct {
		db          *sql.DB
		queryPeriod time.Duration
	}

	// DatabaseConfig contains opnions for creating a new SQL database.
	DatabaseConfig struct {
		// DriverName the type of SQL database.
		DriverName string
		// DatabaseURL is a connection url that the driver for the database can interpret.
		DatabaseURL string
		// QueryPeriod is the amount of time that any database action can take before it should timeout.
		QueryPeriod time.Duration
	}
)

// NewDatabase creates and opens a SQL database from a databaseURL.
func (cfg DatabaseConfig) NewDatabase() (db.Database, error) {
	sqlDB, err := sql.Open(cfg.DriverName, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	sDB := Database{
		db:          sqlDB,
		queryPeriod: cfg.QueryPeriod,
	}
	return sDB, nil
}

// Setup initializes the database by reading the files and executing their contents as raw queries.
func (s Database) Setup(ctx context.Context, files []io.Reader) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.queryPeriod)
	defer cancelFunc()
	queries := make([]db.Query, len(files))
	for i, f := range files {
		b, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading sql setup query %v: %w", i, err)
		}
		queries[i] = RawQuery(string(b))
	}
	if err := s.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("running setup queries %w", err)
	}
	return nil
}

// Query returns the row referenced by the query.
func (s Database) Query(ctx context.Context, q db.Query) db.Scanner {
	ctx, cancelFunc := context.WithTimeout(ctx, s.queryPeriod)
	defer cancelFunc()
	return s.db.QueryRowContext(ctx, q.Cmd(), q.Args()...)
}

// Exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (s Database) Exec(ctx context.Context, queries ...db.Query) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.queryPeriod)
	defer cancelFunc()
	tx, err := s.db.BeginTx(ctx, nil)
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
