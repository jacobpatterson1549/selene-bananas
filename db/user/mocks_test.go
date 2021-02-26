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

func (ph mockPasswordHandler) Hash(password string) ([]byte, error) {
	return ph.hashFunc(password)
}

func (ph mockPasswordHandler) IsCorrect(hashedPassword []byte, password string) (bool, error) {
	return ph.isCorrectFunc(hashedPassword, password)
}

type mockDatabase struct {
	queryFunc func(ctx context.Context, q db.Query) db.Scanner
	execFunc  func(ctx context.Context, queries ...db.Query) error
}

func (d mockDatabase) Setup(ctx context.Context, files []io.Reader) error {
	return fmt.Errorf("Setup should not be called by the server")
}

func (d mockDatabase) Query(ctx context.Context, q db.Query) db.Scanner {
	return d.queryFunc(ctx, q)
}

func (d mockDatabase) Exec(ctx context.Context, queries ...db.Query) error {
	return d.execFunc(ctx, queries...)
}

type mockScanner struct {
	scanFunc func(dest ...interface{}) error
}

func (s mockScanner) Scan(dest ...interface{}) error {
	return s.scanFunc(dest...)
}
