package db

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
)

var testDriver *MockDriver

const (
	testDriverName  = "mockDB"
	testDatabaseURL = "postgres://username:password@host:port/dbname"
)

func init() {
	testDriver = new(MockDriver)
	sql.Register(testDriverName, testDriver)
}
func TestNewDatabase(t *testing.T) {
	newSQLDatabaseTests := []struct {
		*sql.DB
		Config
		wantOk bool
	}{
		{},
		{
			DB: &sql.DB{},
		},
		{
			DB:     &sql.DB{},
			wantOk: true,
			Config: Config{
				QueryPeriod: 1 * time.Hour,
			},
		},
	}
	testDriver.OpenFunc = func(name string) (driver.Conn, error) {
		if testDriverName != name {
			return nil, fmt.Errorf("draver names not equal: wanted %v, got %v", testDriverName, name)
		}
		return MockDriverConn{}, nil
	}
	for i, test := range newSQLDatabaseTests {
		cfg := Config{
			QueryPeriod: test.QueryPeriod,
		}
		sqlDB, err := cfg.NewDatabase(test.DB)
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
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error: %v", err)
	}
	setupTests := []struct {
		files   []io.Reader
		execErr error
		wantOk  bool
	}{
		{
			wantOk: true,
		},
		{
			files: []io.Reader{
				strings.NewReader("1"),
				iotest.ErrReader(fmt.Errorf("error reading file 2")),
				strings.NewReader("3"),
			},
		},
		{
			files: []io.Reader{
				strings.NewReader("1"),
			},
			execErr: fmt.Errorf("error executing files"),
		},
		{
			files: []io.Reader{
				strings.NewReader("1"),
				strings.NewReader("2"),
				strings.NewReader("3"),
			},
			wantOk: true,
		},
	}
	for i, test := range setupTests {
		result := MockDriverResult{
			RowsAffectedFunc: func() (int64, error) {
				return 0, nil
			},
		}
		stmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				return 0
			},
			ExecFunc: func(args []driver.Value) (driver.Result, error) {
				return result, test.execErr
			},
		}
		tx := MockDriverTx{
			CommitFunc: func() error {
				return nil
			},
			RollbackFunc: func() error {
				return nil
			},
		}
		conn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return stmt, nil
			},
			BeginFunc: func() (driver.Tx, error) {
				return tx, nil
			},
		}
		testDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return conn, nil
		}
		db := Database{
			DB: sqlDB,
			Config: Config{
				QueryPeriod: 1 * time.Hour,
			},
		}
		ctx := context.Background()
		err = db.Setup(ctx, test.files)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestDatabaseQuery(t *testing.T) {
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error: %v", err)
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
		rows := MockDriverRows{
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
		stmt := MockDriverStmt{
			CloseFunc: func() error {
				return nil
			},
			NumInputFunc: func() int {
				return 1
			},
			QueryFunc: func(args []driver.Value) (driver.Rows, error) {
				return rows, test.scanErr
			},
		}
		conn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return stmt, nil
			},
		}
		testDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return conn, nil
		}
		q := QueryFunction{
			name:      "SELECT ?;",
			cols:      []string{"?column?"},
			arguments: []interface{}{want},
		}
		db := Database{
			DB: sqlDB,
			Config: Config{
				QueryPeriod: 1 * time.Hour,
			},
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
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error: %v", err)
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
		result := MockDriverResult{
			RowsAffectedFunc: func() (int64, error) {
				return test.rowsAffected, test.rowsAffectedErr
			},
		}
		stmt := MockDriverStmt{
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
				return result, test.execErr
			},
		}
		tx := MockDriverTx{
			CommitFunc: func() error {
				return test.commitErr
			},
			RollbackFunc: func() error {
				return test.rollbackErr
			},
		}
		conn := MockDriverConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return stmt, nil
			},
			BeginFunc: func() (driver.Tx, error) {
				return tx, test.beginErr
			},
		}
		testDriver.OpenFunc = func(name string) (driver.Conn, error) {
			return conn, nil
		}
		var q Query
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
		db := Database{
			DB: sqlDB,
			Config: Config{
				QueryPeriod: 1 * time.Hour,
			},
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