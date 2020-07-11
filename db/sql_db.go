package db

import (
	"context"
	"database/sql"
	"fmt"
)

type (
	sqlDatabase struct {
		db *sql.DB
	}
)

// NewSQLDatabase creates a database from a databaseURL
func NewSQLDatabase(driverName, databaseURL string) (Database, error) {
	sqlDb, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	sDB := sqlDatabase{
		db: sqlDb,
	}
	return sDB, nil
}

// query returns the row referenced by the query
func (s sqlDatabase) query(ctx context.Context, q query) scanner {
	return s.db.QueryRowContext(ctx, q.cmd(), q.args()...)
}

// exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (s sqlDatabase) exec(ctx context.Context, queries ...query) error {
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
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
