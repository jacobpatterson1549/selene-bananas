package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
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
func TestNewDatabase(t *testing.T) {
	newSQLDatabaseTests := []struct {
		driverName  string
		queryPeriod time.Duration
		openFunc    func(name string) (driver.Conn, error)
		setupSQL    [][]byte
		wantOk      bool
	}{
		{
			driverName:  "imaginary_mock_" + mockDriverName,
			queryPeriod: 1,
		},
		{
			driverName: mockDriverName,
		},
		{
			driverName:  mockDriverName,
			queryPeriod: 1 * time.Hour,
			setupSQL: [][]byte{
				[]byte("test"),
			},
			openFunc: func(name string) (driver.Conn, error) {
				return nil, fmt.Errorf("could not open connection for setup queries")
			},
			wantOk: true,
		},
		{ // no setupSQL
			driverName:  mockDriverName,
			queryPeriod: 1 * time.Hour,
			wantOk:      true,
		},
		{
			driverName:  mockDriverName,
			queryPeriod: 1 * time.Hour,
			openFunc: func(name string) (driver.Conn, error) {
				mockTx := MockDriverTx{
					CommitFunc: func() error {
						return nil
					},
				}
				mockConn := MockDriverConn{
					BeginFunc: func() (driver.Tx, error) {
						return mockTx, nil
					},
				}
				return mockConn, nil
			},
			setupSQL: [][]byte{
				[]byte("test"),
			},
			wantOk: true,
		},
	}
	for i, test := range newSQLDatabaseTests {
		cfg := DatabaseConfig{
			DriverName:  test.driverName,
			DatabaseURL: testDatabaseURL,
			QueryPeriod: test.queryPeriod,
		}
		ctx := context.Background()
		var setupSQL [][]byte
		mockDriver.OpenFunc = test.openFunc
		sqlDB, err := cfg.NewDatabase(ctx, setupSQL)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case sqlDB == nil:
			t.Errorf("Test %v: wanted database to be set", i)
		}
	}
}

func TestDatabaseQuery(t *testing.T) {
	queryTests := []struct {
		cancelled bool
		scanErr   error
		wantOk    bool
	}{
		{
			cancelled: true,
		},
		{
			scanErr: fmt.Errorf("problem reading user row"),
		},
		{
			wantOk: true,
		},
	}
	for i, test := range queryTests {
		want := 6
		mockRows := MockDriverRows{
			ColumnsFunc: func() []string {
				return []string{"?column?"}
			},
			CloseFunc: func() error {
				return nil
			},
			NextFunc: func(dest []driver.Value) error {
				dest[0] = want
				return nil
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
		q := QueryFunction{
			name:      "SELECT ?;",
			cols:      []string{"?column?"},
			arguments: []interface{}{want},
		}
		cfg := DatabaseConfig{
			DriverName:  mockDriverName,
			DatabaseURL: testDatabaseURL,
			QueryPeriod: 10 * time.Hour, // test takes real time to run, but this should be large enough
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var setupSQL [][]byte
		db, err := cfg.NewDatabase(ctx, setupSQL)
		if err != nil {
			t.Fatalf("Test %v: unwanted error: %v", i, err)
		}
		var got int
		if test.cancelled {
			cancelFunc()
		}
		r := db.Query(ctx, q)
		err = r.Scan(&got)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case want != got:
			t.Errorf("Test %v: value not set correctly, wanted %v, got %v", i, want, got)
		}
		cancelFunc()
	}
}

func TestDatabaseExec(t *testing.T) {
	cfg := DatabaseConfig{
		DriverName:  mockDriverName,
		DatabaseURL: testDatabaseURL,
		QueryPeriod: 10 * time.Second, // test takes real time to run
	}
	execTests := []struct {
		cancelled       bool
		beginErr        error
		execErr         error
		rowsAffectedErr error
		rowsAffected    int64
		rollbackErr     error
		commitErr       error
		rawQuery        bool
		wantOk          bool
	}{
		{
			cancelled: true,
		},
		{
			beginErr: fmt.Errorf("problem beginning transaction"),
		},
		{
			execErr: fmt.Errorf("problem executing transaction"),
		},
		{
			rowsAffectedErr: fmt.Errorf("problem getting rows affected count"),
		},
		{
			rowsAffected: 0,
		},
		{
			rowsAffected: 2,
			rollbackErr:  fmt.Errorf("problem rolling back transaction"),
		},
		{
			rowsAffected: 1,
			commitErr:    fmt.Errorf("problem committing transaction"),
		},
		{
			rowsAffected: 1,
			wantOk:       true,
		},
		{
			rawQuery: true,
			wantOk:   true,
		},
	}
	for i, test := range execTests {
		mockResult := MockDriverResult{
			RowsAffectedFunc: func() (int64, error) {
				return test.rowsAffected, test.rowsAffectedErr
			},
		}
		mockStmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				if test.rawQuery {
					return 0
				}
				return 2
			},
			ExecFunc: func(args []driver.Value) (driver.Result, error) {
				return mockResult, test.execErr
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
				return mockTx, test.beginErr
			},
		}
		mockDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return mockConn, nil
		}
		var q db.Query
		switch {
		case test.rawQuery:
			q = RawQuery("CREATE TABLE hobbits ( full_name VARCHAR(64) );")
		default:
			q = ExecFunction{
				name: "UPDATE hobbits SET age = ? WHERE first_name = ?;",
				arguments: []interface{}{
					111,
					"Bilbo",
				},
			}
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var setupSQL [][]byte
		db, err := cfg.NewDatabase(ctx, setupSQL)
		if err != nil {
			t.Errorf("unwanted error: %v", err)
		}
		if test.cancelled {
			cancelFunc()
		}
		err = db.Exec(ctx, q)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case err != nil:
			t.Errorf("Test %v: wanted error", i)
		}
		cancelFunc()
	}
}
