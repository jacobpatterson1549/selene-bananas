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
	testDatabaseURL = "postgres://username:password@host:port/db_name"
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
			DB: new(sql.DB),
		},
		{
			DB:     new(sql.DB),
			wantOk: true,
			Config: Config{
				QueryPeriod: 1 * time.Hour,
			},
		},
	}
	testDriver.OpenFunc = func(name string) (driver.Conn, error) {
		if testDriverName != name {
			return nil, fmt.Errorf("driver names not equal: wanted %v, got %v", testDriverName, name)
		}
		return new(MockConn), nil
	}
	for i, test := range newSQLDatabaseTests {
		cfg := Config{
			QueryPeriod: test.QueryPeriod,
		}
		sqlDB, err := cfg.NewDatabase(test.DB)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error creating new database", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error creating new database: %v", i, err)
		case sqlDB == nil:
			t.Errorf("Test %v: wanted database to be set", i)
		}
	}
}

func TestDatabaseSetup(t *testing.T) {
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error opening db to setup: %v", err)
	}
	setupTests := []struct {
		cancelled bool
		files     []io.Reader
		execErr   error
		wantOk    bool
	}{
		{
			wantOk: true,
		},
		{
			cancelled: true,
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
		result := MockResult{
			RowsAffectedFunc: func() (int64, error) {
				return 0, nil
			},
		}
		stmt := MockStmt{
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
		tx := MockTx{
			CommitFunc: func() error {
				return nil
			},
			RollbackFunc: func() error {
				return nil
			},
		}
		conn := MockConn{
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
		ctx, cancelFunc := context.WithCancel(ctx)
		if test.cancelled {
			cancelFunc()
		}
		err = db.Setup(ctx, test.files)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error setting up database", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error setting up database: %v", i, err)
		}
		cancelFunc()
	}
}

func TestDatabaseQuery(t *testing.T) {
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error opening database for query: %v", err)
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
		rows := MockRows{
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
		stmt := MockStmt{
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
		conn := MockConn{
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
		var got int
		err := db.Query(ctx, q, &got)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error quering database", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error quering database: %v", i, err)
		case want != got:
			t.Errorf("Test %v: value not set correctly, wanted %v, got %v", i, want, got)
		}
		cancelFunc()
	}
}

func TestDatabaseExec(t *testing.T) {
	sqlDB, err := sql.Open(testDriverName, testDatabaseURL)
	if err != nil {
		t.Fatalf("unwanted error opening database for exec: %v", err)
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
		result := MockResult{
			RowsAffectedFunc: func() (int64, error) {
				return test.rowsAffected, test.rowsAffectedErr
			},
		}
		stmt := MockStmt{
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
		tx := MockTx{
			CommitFunc: func() error {
				return test.commitErr
			},
			RollbackFunc: func() error {
				return test.rollbackErr
			},
		}
		conn := MockConn{
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
				t.Errorf("Test %v: unwanted error executing query: %v", i, err)
			}
		case err != nil:
			t.Errorf("Test %v: wanted error executing query", i)
		}
		cancelFunc()
	}
}
