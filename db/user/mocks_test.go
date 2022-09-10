package user

import (
	"context"
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

type mockBackend struct {
	createFunc                func(ctx context.Context, u User) error
	readFunc                  func(ctx context.Context, u User) (*User, error)
	updatePasswordFunc        func(ctx context.Context, u User) error
	updatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
	deleteFunc                func(ctx context.Context, u User) error
}

func (m mockBackend) Create(ctx context.Context, u User) error {
	return m.createFunc(ctx, u)
}

func (m mockBackend) Read(ctx context.Context, u User) (*User, error) {
	return m.readFunc(ctx, u)
}

func (m mockBackend) UpdatePassword(ctx context.Context, u User) error {
	return m.updatePasswordFunc(ctx, u)
}

func (m mockBackend) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return m.updatePointsIncrementFunc(ctx, userPoints)
}

func (m mockBackend) Delete(ctx context.Context, u User) error {
	return m.deleteFunc(ctx, u)
}
