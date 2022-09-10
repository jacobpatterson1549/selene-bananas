package user

import (
	"context"
	"fmt"

	"github.com/jacobpatterson1549/selene-bananas/db/user/bcrypt"
)

type (
	// Dao contains CRUD operations for user-related information.
	Dao struct {
		backend         Backend
		passwordHandler passwordHandler
	}

	// Backend contains the operations to manage users
	Backend interface {
		// Create adds the username/password pair.
		Create(ctx context.Context, u User) error
		// Get validates the username/password pair and gets the points.
		Read(ctx context.Context, u User) (*User, error)
		// UpdatePassword updates the password for user identified by the username.
		UpdatePassword(ctx context.Context, u User) error
		// UpdatePointsIncrement increments the points for all of the usernames.
		UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error
		// Delete removes the user.
		Delete(ctx context.Context, u User) error
	}

	passwordHandler interface {
		Hash(password string) ([]byte, error)
		IsCorrect(hashedPassword []byte, password string) (bool, error)
	}
)

// ErrIncorrectLogin should be returned if a login attempt fails because the credentials are invalid.
var ErrIncorrectLogin error = fmt.Errorf("incorrect username/password")
var bph = bcrypt.NewPasswordHandler()

// NewDao creates a Dao using the specified backend.
func NewDao(b Backend) (*Dao, error) {
	if err := validate(b); err != nil {
		return nil, fmt.Errorf("creating user dao: validation: %w", err)
	}
	d := Dao{
		backend:         b,
		passwordHandler: bph,
	}
	return &d, nil
}

// validate checks fields to set up the dao.
func validate(b Backend) error {
	switch {
	case b == nil:
		return fmt.Errorf("backend required")
	}
	return nil
}

// Create adds a user.
func (d Dao) Create(ctx context.Context, u User) error {
	if err := u.Validate(); err != nil {
		return err
	}
	hashedPassword, err := d.passwordHandler.Hash(u.Password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	u.Password = string(hashedPassword)
	if err := d.backend.Create(ctx, u); err != nil {
		return formatBackendError("creating user", err)
	}
	return nil
}

// Login gets ensures the username/password combination is valid and returns all information about the user.
func (d Dao) Login(ctx context.Context, u User) (*User, error) {
	u2, err := d.backend.Read(ctx, u)
	if err != nil {
		if err != ErrIncorrectLogin {
			return nil, formatBackendError("reading user", err)
		}
		return nil, err
	}
	hashedPassword := []byte(u2.Password)
	isCorrect, err := d.passwordHandler.IsCorrect(hashedPassword, u.Password)
	switch {
	case err != nil:
		return nil, fmt.Errorf("checking password correctness: %w", err)
	case !isCorrect:
		return nil, ErrIncorrectLogin
	}
	return u2, nil
}

// UpdatePassword sets the password of a user.
func (d Dao) UpdatePassword(ctx context.Context, u User, newPassword string) error {
	if _, err := d.Login(ctx, u); err != nil {
		return err
	}
	u.Password = newPassword
	if err := u.Validate(); err != nil {
		return err
	}
	hashedPassword, err := d.passwordHandler.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	u.Password = string(hashedPassword)
	if err := d.backend.UpdatePassword(ctx, u); err != nil {
		return formatBackendError("updating user password", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for multiple users by the amount defined in the map.
func (d Dao) UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error {
	if err := d.backend.UpdatePointsIncrement(ctx, usernamePoints); err != nil {
		return formatBackendError("incrementing user points", err)
	}
	return nil
}

// Delete removes a user.
func (d Dao) Delete(ctx context.Context, u User) error {
	if _, err := d.Login(ctx, u); err != nil {
		return err
	}
	if err := d.backend.Delete(ctx, u); err != nil {
		return formatBackendError("deleting user", err)
	}
	return nil
}

// formatBackendError includes the name of the backend in the error message.
func formatBackendError(reason string, err error) error {
	return fmt.Errorf("%v (%T): %w", reason, err, err)
}
