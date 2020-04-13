package db

import (
	"database/sql"
	"fmt"
)

type (
	// Database contains methods to create, read, update, and delete date
	Database interface {
		queryRow(query string, args ...interface{}) row
		exec(query string, args ...interface{}) (sql.Result, error)
		begin() (transaction, error)
	}

	sqlDatabase struct {
		db *sql.DB
	}

	row interface {
		Scan(dest ...interface{}) error
	}


	transaction interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
		Commit() error
		Rollback() error
	}
)

// NewPostgresDatabase creates a postgres database from the datasourcename
func NewPostgresDatabase(dataSourceName string) (Database, error) {
	sqlDb, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("opening database %w", err)
	}
	return sqlDatabase{db: sqlDb}, nil
}

func (s sqlDatabase) queryRow(query string, args ...interface{}) row {
	return s.db.QueryRow(query, args)
}

func (s sqlDatabase) exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args)
}

func (s sqlDatabase) begin() (transaction, error) {
	return s.db.Begin()
}

func expectSingleRowAffected(r sql.Result) error {
	rows, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("expected to update 1 row, but updated %d", rows)
	}
	return nil
}