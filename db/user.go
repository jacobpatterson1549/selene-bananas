package db

import (
	"fmt"
	"unicode"

	"github.com/jacobpatterson1549/selene-bananas/db/bcrypt"
)

type (
	// User contains information for each player.
	User struct {
		Username string
		password string
		Points   int
		ph       passwordHandler
	}

	passwordHandler interface {
		Hash(password string) ([]byte, error)
		IsCorrect(hashedPassword []byte, password string) (bool, error)
	}
)

// NewUser creates a new user with the specified name and password.
func NewUser(u, p string) (*User, error) {
	if err := validateUsername(u); err != nil {
		return nil, err
	}
	if err := validatePassword(p); err != nil {
		return nil, err
	}
	bph := bcrypt.NewPasswordHandler()
	user := User{
		Username: u,
		password: p,
		ph:       bph,
	}
	return &user, nil
}

// validateUsername returns an error if the username is not valid.
func validateUsername(u string) error {
	switch {
	case len(u) < 1:
		return fmt.Errorf("username required")
	case len(u) > 32:
		return fmt.Errorf("username must be less than 32 characters long")
	default:
		for _, r := range u {
			if !unicode.IsLower(r) {
				return fmt.Errorf("username must be made of only lowercase letters")
			}
		}
	}
	return nil
}

// validatePassword returns an error if the password is not valid.
func validatePassword(p string) error {
	switch {
	case len(p) < 8:
		return fmt.Errorf("password must be at least 8 characters long")
	}
	return nil
}

// hashPassword creates a byte hash of the password that should be secure.
func (u User) hashPassword() ([]byte, error) {
	hashedPassword, err := u.ph.Hash(u.password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	return hashedPassword, nil
}

// isCorrectPassword returns whether or not the hashed form of the password is correct,
// returning an error if a problem occurs while checking it.
func (u User) isCorrectPassword(hashedPassword []byte) (bool, error) {
	ok, err := u.ph.IsCorrect(hashedPassword, u.password)
	if err != nil {
		return false, fmt.Errorf("checking to see if password is correct: %w", err)
	}
	return ok, nil
}
