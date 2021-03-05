package user

import (
	"context"
	"fmt"
	"io"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

type (
	// Dao contains CRUD operations for user-related information.
	Dao struct {
		db Database
	}

	// Database contains methods to create, read, update, and delete data.
	Database interface {
		// Setup initializes the database by reading the files.
		Setup(ctx context.Context, files []io.Reader) error
		// Query reads from the database without updating it.
		Query(ctx context.Context, q db.Query) db.Scanner
		// Exec makes a change to existing data, creating/modifying/removing it.
		Exec(ctx context.Context, queries ...db.Query) error
	}
)

// ErrIncorrectLogin should be returned if a login attempt fails because the credentials are invalid.
var ErrIncorrectLogin error = fmt.Errorf("incorrect username/password")

// NewDao creates a Dao on the specified database.
func NewDao(db Database) (*Dao, error) {
	if err := validate(db); err != nil {
		return nil, fmt.Errorf("creating user dao: validation: %w", err)
	}
	d := Dao{
		db: db,
	}
	return &d, nil
}

// validate checks fields to set up the dao.
func validate(db Database) error {
	switch {
	case db == nil:
		return fmt.Errorf("database required")
	}
	return nil
}

// Create adds a user.
func (d Dao) Create(ctx context.Context, u User) error {
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := db.NewExecFunction("user_create", u.Username, hashedPassword)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Login gets ensures the username/password combination is valid and returns all information about the user.
func (d Dao) Login(ctx context.Context, u User) (*User, error) {
	cols := []string{
		"username",
		"password",
		"points",
	}
	q := db.NewQueryFunction("user_read", cols, u.Username)
	row := d.db.Query(ctx, q)
	var u2 User
	if err := row.Scan(&u2.Username, &u2.password, &u2.Points); err != nil {
		if err == db.ErrNoRows {
			return nil, ErrIncorrectLogin
		}
		return nil, fmt.Errorf("reading user: %w", err)
	}
	hashedPassword := []byte(u2.password)
	isCorrect, err := u.isCorrectPassword(hashedPassword)
	switch {
	case err != nil:
		return nil, fmt.Errorf("reading user: %w", err)
	case !isCorrect:
		return nil, fmt.Errorf("incorrect password")
	}
	return &u2, nil
}

// UpdatePassword sets the password of a user.
func (d Dao) UpdatePassword(ctx context.Context, u User, newP string) error {
	if _, err := d.Login(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	if err := validatePassword(newP); err != nil {
		return err
	}
	u.password = newP
	hashedPassword, err := u.hashPassword()
	if err != nil {
		return err
	}
	q := db.NewExecFunction("user_update_password", u.Username, hashedPassword)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for multiple users by the amount defined in the map.
func (d Dao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	queries := make([]db.Query, 0, len(userPoints))
	for username, points := range userPoints {
		queries = append(queries, db.NewExecFunction("user_update_points_increment", username, points))
	}
	if err := d.db.Exec(ctx, queries...); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes a user.
func (d Dao) Delete(ctx context.Context, u User) error {
	if _, err := d.Login(ctx, u); err != nil {
		return fmt.Errorf("checking password: %w", err)
	}
	q := db.NewExecFunction("user_delete", u.Username)
	if err := d.db.Exec(ctx, q); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}
