package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"
)

var mockDriver *MockDriver

const (
	mockDriverName  = "mockDB"
	testDatabaseURL = "postgres://username:password@host:port/dbname"
)

func init() {
	mockDriver = new(MockDriver)
	sql.Register(mockDriverName, mockDriver)
}
func TestNewPostgresDatabase(t *testing.T) {
	if _, err := NewSQLDatabase("imaginaryDB", testDatabaseURL); err == nil {
		t.Errorf("expeceted error when creating database for driver that is not registered")
	}
	db, err := NewSQLDatabase(mockDriverName, testDatabaseURL)
	switch {
	case err != nil:
		t.Errorf("creating a new SQL Database should not cause an error with the mock driver")
	case db == nil:
		t.Errorf("expected non-nil Database")
	}
}

func TestDatabaseQueryRow(t *testing.T) {
	queryTests := []struct {
		cancelled bool
		scanErr   error
		wantErr   bool
	}{
		{
			cancelled: true,
			wantErr:   true,
		},
		{
			scanErr: fmt.Errorf("problem reading user row"),
			wantErr: true,
		},
		{
			// [all ok]
		},
	}
	for i, test := range queryTests {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		switch {
		case test.cancelled:
			cancelFunc()
		default:
			defer cancelFunc()
		}
		want := 6
		rowNum := 0
		mockRows := MockDriverRows{
			ColumnsFunc: func() []string {
				return []string{"?column?"}
			},
			CloseFunc: func() error {
				return nil
			},
			NextFunc: func(dest []driver.Value) error {
				rowNum++
				if rowNum == 1 {
					dest[0] = want
					return nil
				}
				return io.EOF
			},
		}
		mockStmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				return 1
			},
			QueryFunc: func(args []driver.Value) (driver.Rows, error) {
				return mockRows, test.scanErr
			},
		}
		mockConn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return mockStmt, nil
			},
		}
		mockDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return mockConn, nil
		}
		q := sqlQueryFunction{
			name:      "SELECT ?;",
			cols:      []string{"?column?"},
			arguments: []interface{}{want},
		}
		db, _ := NewSQLDatabase(mockDriverName, testDatabaseURL)
		r := db.query(ctx, q)
		var got int
		err := r.Scan(&got)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		case want != got:
			t.Errorf("Test %v: value not set correctly, wanted %v, got %v", i, want, got)
		}
	}
}

func TestDatabaseExec(t *testing.T) {
	execTests := []struct {
		cancelled       bool
		beginErr        error
		execErr         error
		rowsAffectedErr error
		rowsAffected    int64
		rollbackErr     error
		commitErr       error
		q               query
		wantErr         bool
	}{
		{
			cancelled: true,
			wantErr:   true,
		},
		{
			beginErr: fmt.Errorf("problem beginning transaction"),
			wantErr:  true,
		},
		{
			execErr: fmt.Errorf("problem executing transaction"),
			wantErr: true,
		},
		{
			rowsAffectedErr: fmt.Errorf("problem getting rows affected count"),
			wantErr:         true,
		},
		{
			rowsAffected: 0,
			wantErr:      true,
		},
		{
			rowsAffected: 2,
			rollbackErr:  fmt.Errorf("problem rolling back transaction"),
			wantErr:      true,
		},
		{
			rowsAffected: 1,
			commitErr:    fmt.Errorf("problem committing transaction"),
			wantErr:      true,
		},
		{
			rowsAffected: 1,
			// [all ok]
		},
	}
	for i, test := range execTests {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		switch {
		case test.cancelled:
			cancelFunc()
		default:
			defer cancelFunc()
		}
		mockResult := MockDriverResult{
			RowsAffectedFunc: func() (int64, error) {
				if test.rowsAffectedErr != nil {
					return 0, test.rowsAffectedErr
				}
				return test.rowsAffected, nil
			},
		}
		mockStmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				return 2
			},
			ExecFunc: func(args []driver.Value) (driver.Result, error) {
				if test.execErr != nil {
					return nil, test.execErr
				}
				return mockResult, nil
			},
		}
		mockTx := MockDriverTx{
			CommitFunc: func() error {
				return test.commitErr
			},
			RollbackFunc: func() error {
				return test.rollbackErr
			},
		}
		mockConn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return mockStmt, nil
			},
			BeginFunc: func() (driver.Tx, error) {
				if test.beginErr != nil {
					return nil, test.beginErr
				}
				return mockTx, nil
			},
		}
		mockDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return mockConn, nil
		}
		q := sqlExecFunction{
			name: "UPDATE hobbits SET age = ? WHERE first_name = ?;",
			arguments: []interface{}{
				111,
				"Bilbo",
			},
		}
		db, _ := NewSQLDatabase(mockDriverName, testDatabaseURL)
		err := db.exec(ctx, q)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

// MockDriver implements the sql/driver.Driver interface.
type MockDriver struct {
	OpenFunc func(name string) (driver.Conn, error)
}

// Open creates a new connection.
func (m MockDriver) Open(name string) (driver.Conn, error) {
	return m.OpenFunc(name)
}

// MockDriverConn implements the sql/driver.Conn interface.
type MockDriverConn struct {
	PrepareFunc func(query string) (driver.Stmt, error)
	CloseFunc   func() error
	BeginFunc   func() (driver.Tx, error)
}

// Prepare returns a prepared statement, bound to this connection.
func (m MockDriverConn) Prepare(query string) (driver.Stmt, error) {
	return m.PrepareFunc(query)
}

// Close cancels the connection.
func (m MockDriverConn) Close() error {
	return m.CloseFunc()
}

// Begin starts and returns a new transaction.
func (m MockDriverConn) Begin() (driver.Tx, error) {
	return m.BeginFunc()
}

// MockDriverStmt implements the sql/driver.Stmt interface.
type MockDriverStmt struct {
	CloseFunc    func() error
	NumInputFunc func() int
	ExecFunc     func(args []driver.Value) (driver.Result, error)
	QueryFunc    func(args []driver.Value) (driver.Rows, error)
}

// Close closes the statement.
func (m MockDriverStmt) Close() error {
	return m.CloseFunc()
}

// NumInput returns the number of placeholder parameters.
func (m MockDriverStmt) NumInput() int {
	return m.NumInputFunc()
}

// Exec executes a query that doesn't return rows, such as an INSERT or UPDATE.
func (m MockDriverStmt) Exec(args []driver.Value) (driver.Result, error) {
	return m.ExecFunc(args)
}

// Query executes a query that may return rows, such as a SELECT.
func (m MockDriverStmt) Query(args []driver.Value) (driver.Rows, error) {
	return m.QueryFunc(args)
}

// MockDriverTx implements the sql/driver/Tx interface.
type MockDriverTx struct {
	CommitFunc   func() error
	RollbackFunc func() error
}

// Commit commits the transaction.
func (m MockDriverTx) Commit() error {
	return m.CommitFunc()
}

// Rollback aborts the transaction.
func (m MockDriverTx) Rollback() error {
	return m.RollbackFunc()
}

// MockDriverResult implements the sql/driver.Result interface.
type MockDriverResult struct {
	LastInsertIDFunc func() (int64, error)
	RowsAffectedFunc func() (int64, error)
}

// LastInsertId returns the database's auto-generated ID.
func (m MockDriverResult) LastInsertId() (int64, error) {
	return m.LastInsertIDFunc()
}

// RowsAffected returns the number of rows affected by the query.
func (m MockDriverResult) RowsAffected() (int64, error) {
	return m.RowsAffectedFunc()
}

// MockDriverRows implements the sql/driver.Rows interface.
type MockDriverRows struct {
	ColumnsFunc func() []string
	CloseFunc   func() error
	NextFunc    func(dest []driver.Value) error
}

// Columns returns the names of the columns.
func (m MockDriverRows) Columns() []string {
	return m.ColumnsFunc()
}

// Close closes the rows iterator.
func (m MockDriverRows) Close() error {
	return m.CloseFunc()
}

// Next is called to populate the next row of data into the provided slice.
// Next should return io.EOF when there are no more rows.
func (m MockDriverRows) Next(dest []driver.Value) error {
	return m.NextFunc(dest)
}
