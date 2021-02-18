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
			queryPeriod: 1,
			wantOk:      true,
		},
	}
	for i, test := range newSQLDatabaseTests {
		cfg := DatabaseConfig{
			DriverName:  test.driverName,
			DatabaseURL: testDatabaseURL,
			QueryPeriod: test.queryPeriod,
		}
		sqlDB, err := cfg.NewDatabase()
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
		db, err := cfg.NewDatabase()
		if err != nil {
			t.Errorf("Test %v: unwanted error: %v", i, err)
			continue
		}
		var got int
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
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
				if test.rawQuery {
					return 0
				}
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
		db, err := cfg.NewDatabase()
		if err != nil {
			t.Errorf("unwanted error: %v", err)
		}
		err = db.Exec(ctx, q)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}
