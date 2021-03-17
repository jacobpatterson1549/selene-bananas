package user

import (
	"context"
	"fmt"
	"io"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

type mockPasswordHandler struct {
	hashFunc      func(password string) ([]byte, error)
	isCorrectFunc func(hashedPassword []byte, password string) (bool, error)
}

func (m mockPasswordHandler) Hash(password string) ([]byte, error) {
	return m.hashFunc(password)
}

func (m mockPasswordHandler) IsCorrect(hashedPassword []byte, password string) (bool, error) {
	return m.isCorrectFunc(hashedPassword, password)
}

type mockDatabase struct {
	queryFunc func(ctx context.Context, q db.Query, dest ...interface{}) error
	execFunc  func(ctx context.Context, queries ...db.Query) error
}

func (m mockDatabase) Setup(ctx context.Context, files []io.Reader) error {
	return fmt.Errorf("Setup should not be called by the server")
}

func (m mockDatabase) Query(ctx context.Context, q db.Query, dest ...interface{}) error {
	return m.queryFunc(ctx, q, dest...)
}

func (m mockDatabase) Exec(ctx context.Context, queries ...db.Query) error {
	return m.execFunc(ctx, queries...)
}
