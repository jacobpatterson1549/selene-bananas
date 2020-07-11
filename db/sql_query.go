package db

import (
	"fmt"
	"strings"
)

type (
	sqlQueryFunction struct {
		name      string
		cols      []string
		arguments []interface{}
	}

	sqlExecFunction struct {
		name      string
		arguments []interface{}
	}

	sqlExecRaw struct {
		sqlRaw string
	}
)

func newSQLQueryFunction(name string, cols []string, args ...interface{}) sqlQueryFunction {
	return sqlQueryFunction{
		name:      name,
		cols:      cols,
		arguments: args,
	}
}

func newSQLExecFunction(name string, args ...interface{}) sqlExecFunction {
	return sqlExecFunction{
		name:      name,
		arguments: args,
	}
}

func (q sqlQueryFunction) cmd() string {
	argIndexes := make([]string, len(q.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s FROM %s(%s)", strings.Join(q.cols, ", "), q.name, strings.Join(argIndexes, ", "))
}

func (e sqlExecFunction) cmd() string {
	argIndexes := make([]string, len(e.arguments))
	for i := range argIndexes {
		argIndexes[i] = fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf("SELECT %s(%s)", e.name, strings.Join(argIndexes, ", "))
}

func (r sqlExecRaw) cmd() string {
	return r.sqlRaw
}

func (q sqlQueryFunction) args() []interface{} {
	return q.arguments
}

func (e sqlExecFunction) args() []interface{} {
	return e.arguments
}

func (sqlExecRaw) args() []interface{} {
	return nil
}
