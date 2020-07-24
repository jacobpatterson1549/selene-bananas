package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type (
	sqlDatabase struct {
		db          *sql.DB
		queryPeriod time.Duration
	}

	// SQLDatabaseConfig contains opnions for creating a new SQL database.
	SQLDatabaseConfig struct {
		// DriverName the type of SQL database.
		DriverName string
		// DatabaseURL is a connection url that the driver for the database can interpret.
		DatabaseURL string
		// QueryPeriod is the amount of time that any database action can take before it should timeout.
		QueryPeriod time.Duration
	}
)

// NewSQLDatabase creates a database from a databaseURL.
func (cfg SQLDatabaseConfig) NewSQLDatabase() (Database, error) {
	sqlDB, err := cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("creating sql database: validation: %w", err)
	}
	sDB := sqlDatabase{
		db:          sqlDB,
		queryPeriod: cfg.QueryPeriod,
	}
	return sDB, nil
}

func (cfg SQLDatabaseConfig) validate() (*sql.DB, error) {
	sqlDB, err := sql.Open(cfg.DriverName, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	if cfg.QueryPeriod <= 0 {
		return nil, fmt.Errorf("positive idle period required")
	}
	return sqlDB, nil
}

// query returns the row referenced by the query.
func (s sqlDatabase) query(ctx context.Context, q query) scanner {
	ctx, cancelFunc := context.WithTimeout(ctx, s.queryPeriod)
	defer cancelFunc()
	return s.db.QueryRowContext(ctx, q.cmd(), q.args()...)
}

// exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (s sqlDatabase) exec(ctx context.Context, queries ...query) error {
	ctx, cancelFunc := context.WithTimeout(ctx, s.queryPeriod)
	defer cancelFunc()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	for i, q := range queries {
		result, err := tx.ExecContext(ctx, q.cmd(), q.args()...)
		if f, ok := q.(sqlExecFunction); err == nil && ok {
			var n int64
			n, err = result.RowsAffected()
			if err == nil && n != 1 {
				err = fmt.Errorf("expected to update 1 row, but updated %d when calling %s", n, f.name)
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
