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
	// Database is a SQL db.Database with additional configuration
	Database struct {
		*sql.DB
		Config
	}

	// Config contains opnions for how the database should run.
	Config struct {
		// QueryPeriod is the amount of time that any database action can take before it should timeout.
		QueryPeriod time.Duration
	}
)

// NewDatabase creates a SQL database from the database.
func (cfg Config) NewDatabase(db *sql.DB) (db.Database, error) {
	if err := cfg.validate(db); err != nil {
		return nil, fmt.Errorf("creating database: validation: %w", err)
	}
	sDB := Database{
		DB:     db,
		Config: cfg,
	}
	return sDB, nil
}

// validate ensures the configuration and parameters have no errors.
func (cfg Config) validate(db *sql.DB) error {
	switch {
	case db == nil:
		return fmt.Errorf("database required")
	case cfg.QueryPeriod <= 0:
		return fmt.Errorf("positive idle period required")
	}
	return nil
}

// Setup initializes the database by reading the files and executing their contents as raw queries.
func (s Database) Setup(ctx context.Context, files []io.Reader) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.QueryPeriod)
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
	ctx, cancelFunc := context.WithTimeout(ctx, s.QueryPeriod)
	defer cancelFunc()
	return s.DB.QueryRowContext(ctx, q.Cmd(), q.Args()...)
}

// Exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (s Database) Exec(ctx context.Context, queries ...db.Query) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.QueryPeriod)
	defer cancelFunc()
	tx, err := s.DB.BeginTx(ctx, nil)
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
