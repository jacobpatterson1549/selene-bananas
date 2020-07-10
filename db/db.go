// Package db interacts with storing user attributes so they can be retrieved after the server restarts
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type (
	// Database contains methods to create, read, update, and delete data
	Database interface {
		queryRow(ctx context.Context, q sqlQuery) row
		exec(ctx context.Context, queries ...sqlQuery) error
	}

	sqlDatabase struct {
		db *sql.DB
	}

	row interface {
		Scan(dest ...interface{}) error
	}

	transaction interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
		Commit() error
		Rollback() error
	}

	sqlQuery interface {
		sql() string
		args() []interface{}
	}

	querySQLFunction struct {
		name      string
		cols      []string
		arguments []interface{}
	}

	execSQLFunction struct {
		name      string
		arguments []interface{}
	}

	execSQLRaw struct {
		sqlRaw string
	}
)

// NewPostgresDatabase creates a postgres database from a databaseURL
func NewPostgresDatabase(databaseURL string) (Database, error) {
	sqlDb, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	return sqlDatabase{db: sqlDb}, nil
}

// query returns the row referenced by the query
func (s sqlDatabase) queryRow(ctx context.Context, q sqlQuery) row {
	return s.db.QueryRowContext(ctx, q.sql(), q.args()...)
}

// exec evaluates multiple queries in a transaction, ensuring each execSQLFunction one only updates one row.
func (s sqlDatabase) exec(ctx context.Context, queries ...sqlQuery) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	for i, q := range queries {
		result, err := tx.ExecContext(ctx, q.sql(), q.args()...)
		if f, ok := q.(execSQLFunction); err != nil && ok {
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

func newQuerySQLFunction(name string, cols []string, args ...interface{}) querySQLFunction {
	return querySQLFunction{
		name:      name,
		cols:      cols,
		arguments: args,
	}
}

func newExecSQLFunction(name string, args ...interface{}) execSQLFunction {
	return execSQLFunction{
		name:      name,
		arguments: args,
	}
}

func (f querySQLFunction) sql() string {
	argIndexes := make([]string, len(f.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s FROM %s(%s)", strings.Join(f.cols, ", "), f.name, strings.Join(argIndexes, ", "))
}

func (f execSQLFunction) sql() string {
	argIndexes := make([]string, len(f.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s(%s)", f.name, strings.Join(argIndexes, ", "))
}

func (r execSQLRaw) sql() string {
	return r.sqlRaw
}

func (f querySQLFunction) args() []interface{} {
	return f.arguments
}

func (f execSQLFunction) args() []interface{} {
	return f.arguments
}

func (r execSQLRaw) args() []interface{} {
	return nil
}
