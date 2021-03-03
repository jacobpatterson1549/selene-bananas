package db

import "database/sql/driver"

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
