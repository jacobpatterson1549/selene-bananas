package sql

import "database/sql/driver"

// MockDriver implements the sql/driver.Driver interface.
type MockDriver struct {
	OpenFunc func(name string) (driver.Conn, error)
}

// Open creates a new connection.
func (m MockDriver) Open(name string) (driver.Conn, error) {
	return m.OpenFunc(name)
}

// MockConn implements the sql/driver.Conn interface.
type MockConn struct {
	PrepareFunc func(query string) (driver.Stmt, error)
	CloseFunc   func() error
	BeginFunc   func() (driver.Tx, error)
}

func (m MockConn) Prepare(query string) (driver.Stmt, error) {
	return m.PrepareFunc(query)
}

func (m MockConn) Close() error {
	return m.CloseFunc()
}

func (m MockConn) Begin() (driver.Tx, error) {
	return m.BeginFunc()
}

// MockStmt implements the sql/driver.Stmt interface.
type MockStmt struct {
	CloseFunc    func() error
	NumInputFunc func() int
	ExecFunc     func(args []driver.Value) (driver.Result, error)
	QueryFunc    func(args []driver.Value) (driver.Rows, error)
}

func (m MockStmt) Close() error {
	return m.CloseFunc()
}

func (m MockStmt) NumInput() int {
	return m.NumInputFunc()
}

func (m MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return m.ExecFunc(args)
}

func (m MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return m.QueryFunc(args)
}

// MockTx implements the sql/driver/Tx interface.
type MockTx struct {
	CommitFunc   func() error
	RollbackFunc func() error
}

func (m MockTx) Commit() error {
	return m.CommitFunc()
}

func (m MockTx) Rollback() error {
	return m.RollbackFunc()
}

// MockResult implements the sql/driver.Result interface.
type MockResult struct {
	LastInsertIDFunc func() (int64, error)
	RowsAffectedFunc func() (int64, error)
}

func (m MockResult) LastInsertId() (int64, error) {
	return m.LastInsertIDFunc()
}

func (m MockResult) RowsAffected() (int64, error) {
	return m.RowsAffectedFunc()
}

// MockRows implements the sql/driver.Rows interface.
type MockRows struct {
	ColumnsFunc func() []string
	CloseFunc   func() error
	NextFunc    func(dest []driver.Value) error
}

func (m MockRows) Columns() []string {
	return m.ColumnsFunc()
}

func (m MockRows) Close() error {
	return m.CloseFunc()
}

func (m MockRows) Next(dest []driver.Value) error {
	return m.NextFunc(dest)
}
