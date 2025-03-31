// Package postgres implements an SQL Database for Postgres servers.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

// UserBackend provides functions to manages users on a Postgres SQL Database.
type (
	UserBackend struct {
		Database
	}
	// Database contains methods to create, read, update, and delete data.
	Database interface {
		// Setup initializes the database by reading the files.
		Setup(ctx context.Context, files []io.Reader) error
		// Query reads from the database without updating it.
		Query(ctx context.Context, q sql.Query, dest ...interface{}) error
		// Exec makes a change to existing data, creating/modifying/removing it.
		Exec(ctx context.Context, queries ...sql.Query) error
	}
)

// Create adds the username/password pair.
func (ub *UserBackend) Create(ctx context.Context, u user.User) error {
	q := sql.NewExecFunction("user_create", u.Username, u.Password)
	if err := ub.Database.Exec(ctx, q); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Read queries the database for the user by username
func (ub *UserBackend) Read(ctx context.Context, u user.User) (*user.User, error) {
	cols := []string{
		"username",
		"password",
		"points",
	}
	q := sql.NewQueryFunction("user_read", cols, u.Username)
	var u2 user.User
	if err := ub.Database.Query(ctx, q, &u2.Username, &u2.Password, &u2.Points); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrIncorrectLogin
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return &u2, nil
}

// UpdatePassword updates the password for user identified by the username.
func (ub *UserBackend) UpdatePassword(ctx context.Context, u user.User) error {
	q := sql.NewExecFunction("user_update_password", u.Username, u.Password)
	if err := ub.Database.Exec(ctx, q); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement changes the points for all of the usernames.
func (ub *UserBackend) UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error {
	queries := make([]sql.Query, 0, len(usernamePoints))
	for username, points := range usernamePoints {
		queries = append(queries, sql.NewExecFunction("user_update_points_increment", username, points))
	}
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Args()[0].(string) < queries[j].Args()[0].(string)
	})
	if err := ub.Database.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes the user.
func (ub *UserBackend) Delete(ctx context.Context, u user.User) error {
	q := sql.NewExecFunction("user_delete", u.Username)
	if err := ub.Database.Exec(ctx, q); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}
