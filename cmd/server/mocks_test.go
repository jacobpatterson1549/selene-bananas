package main

import (
	"database/sql/driver"
)

// TestNoopDriver creates connections that have noop statements and transactions.
var TestNoopDriver driver.Driver = &mockDriver{
	OpenFunc: func(name string) (driver.Conn, error) {
		return mockConn{
			PrepareFunc: func(query string) (driver.Stmt, error) {
				return mockStmt{
					CloseFunc: func() error {
						return nil
					},
					NumInputFunc: func() int {
						return 0
					},
					ExecFunc: func(args []driver.Value) (driver.Result, error) {
						return nil, nil
					},
				}, nil
			},
			BeginFunc: func() (driver.Tx, error) {
				return mockTx{
					CommitFunc: func() error {
						return nil
					},
				}, nil
			},
		}, nil
	},
}

// mockDriver implements the sql/driver.Driver interface.
type mockDriver struct {
	OpenFunc func(name string) (driver.Conn, error)
}

func (m mockDriver) Open(name string) (driver.Conn, error) {
	return m.OpenFunc(name)
}

// mockConn implements the sql/driver.Conn interface.
type mockConn struct {
	PrepareFunc func(query string) (driver.Stmt, error)
	CloseFunc   func() error
	BeginFunc   func() (driver.Tx, error)
}

func (m mockConn) Prepare(query string) (driver.Stmt, error) {
	return m.PrepareFunc(query)
}

func (m mockConn) Close() error {
	return m.CloseFunc()
}

func (m mockConn) Begin() (driver.Tx, error) {
	return m.BeginFunc()
}

// mockTx implements the sql/driver.Tx interface.
type mockTx struct {
	CommitFunc   func() error
	RollbackFunc func() error
}

func (m mockTx) Commit() error {
	return m.CommitFunc()
}

func (m mockTx) Rollback() error {
	return m.RollbackFunc()
}

// mockStmt implements the sql/driver.Stmt interface.
type mockStmt struct {
	CloseFunc    func() error
	NumInputFunc func() int
	ExecFunc     func(args []driver.Value) (driver.Result, error)
	QueryFunc    func(args []driver.Value) (driver.Rows, error)
}

func (m mockStmt) Close() error {
	return m.CloseFunc()
}

func (m mockStmt) NumInput() int {
	return m.NumInputFunc()
}

func (m mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return m.ExecFunc(args)
}

func (m mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return m.QueryFunc(args)
}
