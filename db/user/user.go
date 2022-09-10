// Package user contains handles the state of users.
package user

import (
	"fmt"
	"unicode"
)

// User contains information for each player.
type User struct {
	Username string
	Password string
	Points   int
}

// New creates a new user with the specified name and password.
func New(u, p string) (*User, error) {
	if err := validateUsername(u); err != nil {
		return nil, err
	}
	if err := validatePassword(p); err != nil {
		return nil, err
	}
	user := User{
		Username: u,
		Password: p,
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
