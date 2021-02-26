package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
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
			driverName:  mockDriverName,
			queryPeriod: 1 * time.Hour,
			wantOk:      true,
		},
	}
	mockDriver.OpenFunc = func(name string) (driver.Conn, error) {
		if mockDriverName != name {
			return nil, fmt.Errorf("draver names not equal: wanted %v, got %v", mockDriverName, name)
		}
		return MockDriverConn{}, nil
	}
	for i, test := range newSQLDatabaseTests {
		cfg := DatabaseConfig{
			DriverName:  test.driverName,
			DatabaseURL: testDatabaseURL,
			QueryPeriod: test.queryPeriod,
		}
		sqlDB, err := cfg.NewDatabase()
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case sqlDB == nil:
			t.Errorf("Test %v: wanted database to be set", i)
		}
	}
}

func TestDatabaseSetup(t *testing.T) {
	cfg := DatabaseConfig{
		DriverName:  mockDriverName,
		DatabaseURL: testDatabaseURL,
		QueryPeriod: 1 * time.Hour,
	}
	setupTests := []struct {
		files   []io.Reader
		execErr error
		wantErr bool
	}{
		{},
		{
			files: []io.Reader{
				strings.NewReader("1"),
				iotest.ErrReader(fmt.Errorf("error reading file 2")),
				strings.NewReader("3"),
			},
			wantErr: true,
		},
		{
			files: []io.Reader{
				strings.NewReader("1"),
			},
			execErr: fmt.Errorf("error executing files"),
			wantErr: true,
		},
		{
			files: []io.Reader{
				strings.NewReader("1"),
				strings.NewReader("2"),
				strings.NewReader("3"),
			},
		},
	}
	for i, test := range setupTests {
		mockResult := MockDriverResult{
			RowsAffectedFunc: func() (int64, error) {
				return 0, nil
			},
		}
		mockStmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				return 0
			},
			ExecFunc: func(args []driver.Value) (driver.Result, error) {
				return mockResult, test.execErr
			},
		}
		mockTx := MockDriverTx{
			CommitFunc: func() error {
				return nil
			},
			RollbackFunc: func() error {
				return nil
			},
		}
		mockConn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return mockStmt, nil
			},
			BeginFunc: func() (driver.Tx, error) {
				return mockTx, nil
			},
		}
		mockDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return mockConn, nil
		}
		db, err := cfg.NewDatabase()
		if err != nil {
			t.Errorf("unwanted error: %v", err)
			continue
		}
		ctx := context.Background()
		err = db.Setup(ctx, test.files)
		switch {
		case test.wantErr:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestDatabaseQuery(t *testing.T) {
	cfg := DatabaseConfig{
		DriverName:  mockDriverName,
		DatabaseURL: testDatabaseURL,
		QueryPeriod: 1 * time.Hour,
	}
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
		db, err := cfg.NewDatabase()
		if err != nil {
			t.Errorf("unwanted error: %v", err)
			continue
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		if test.cancelled {
			cancelFunc()
		}
		r := db.Query(ctx, q)
		var got int
		err = r.Scan(&got)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
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
		QueryPeriod: 1 * time.Hour,
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
		db, err := cfg.NewDatabase()
		if err != nil {
			t.Errorf("unwanted error: %v", err)
			continue
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
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
