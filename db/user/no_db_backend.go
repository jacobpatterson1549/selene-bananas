package user

import (
	"context"
	"fmt"
)

type NoDatabaseBackend struct{}

// Create returns an error.
func (b NoDatabaseBackend) Create(ctx context.Context, u User) error {
	return fmt.Errorf("no database to create user")
}

// Read returns the user.
func (b NoDatabaseBackend) Read(ctx context.Context, u User) (*User, error) {
	return &u, nil
}

// UpdatePassword returns an error
func (b NoDatabaseBackend) UpdatePassword(ctx context.Context, u User) error {
	return fmt.Errorf("no database to update user password")
}

// UpdatePointsIncrement returns an error.
func (b NoDatabaseBackend) UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error {
	return fmt.Errorf("no database to increment user points")
}

// Delete returns an error.
func (b NoDatabaseBackend) Delete(ctx context.Context, u User) error {
	return fmt.Errorf("no database to delete user")
}
