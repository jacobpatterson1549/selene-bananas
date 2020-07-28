package sql

import (
	"fmt"
	"strings"
)

type (
	// QueryFunction is a db Query that reads data.
	QueryFunction struct {
		name      string
		cols      []string
		arguments []interface{}
	}

	// ExecFunction is a db Query that changes data.
	ExecFunction struct {
		name      string
		arguments []interface{}
	}

	// RawQuery is a db Query that changes data and has no arguments..
	RawQuery string
)

// NewQueryFunction creates a Query to call a query function.
func NewQueryFunction(name string, cols []string, args ...interface{}) QueryFunction {
	q := QueryFunction{
		name:      name,
		cols:      cols,
		arguments: args,
	}
	return q
}

// NewExecFunction creates a Query to call an exec function.
func NewExecFunction(name string, args ...interface{}) ExecFunction {
	e := ExecFunction{
		name:      name,
		arguments: args,
	}
	return e
}

// Cmd returns a SQL string to execute the function with arguments.
func (q QueryFunction) Cmd() string {
	argIndexes := make([]string, len(q.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s FROM %s(%s)", strings.Join(q.cols, ", "), q.name, strings.Join(argIndexes, ", "))
}

// Cmd returns a SQL string  to execute the function with arguments.
func (e ExecFunction) Cmd() string {
	argIndexes := make([]string, len(e.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s(%s)", e.name, strings.Join(argIndexes, ", "))
}

// Cmd returns the raw SQL query.
func (r RawQuery) Cmd() string {
	return string(r)
}

// Args returns the arguments for the query function.
func (q QueryFunction) Args() []interface{} {
	return q.arguments
}

// Args returns the arguments for the exec function.
func (e ExecFunction) Args() []interface{} {
	return e.arguments
}

// Args returns nil for the raw SQL query.
func (RawQuery) Args() []interface{} {
	return nil
}
