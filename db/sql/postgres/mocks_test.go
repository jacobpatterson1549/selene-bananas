package postgres

import (
	"context"
	"io"

	"github.com/jacobpatterson1549/selene-bananas/db/sql"
)

type mockDatabase struct {
	SetupFunc func(ctx context.Context, files []io.Reader) error
	QueryFunc func(ctx context.Context, q sql.Query, dest ...interface{}) error
	ExecFunc  func(ctx context.Context, queries ...sql.Query) error
}

func (m mockDatabase) Setup(ctx context.Context, files []io.Reader) error {
	return m.SetupFunc(ctx, files)
}
func (m mockDatabase) Query(ctx context.Context, q sql.Query, dest ...interface{}) error {
	return m.QueryFunc(ctx, q, dest...)
}
func (m mockDatabase) Exec(ctx context.Context, queries ...sql.Query) error {
	return m.ExecFunc(ctx, queries...)
}
