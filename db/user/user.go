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
	IsOauth2 bool
}

// Validate checks if the username and password are valid.
func (u User) Validate() error {
	if err := u.validateUsername(); err != nil {
		return err
	}
	if err := u.validatePassword(); err != nil {
		return err
	}
	return nil
}

// validateUsername returns an error if the username is not valid.
func (u User) validateUsername() error {
	switch {
	case len(u.Username) < 1:
		return fmt.Errorf("username required")
	case len(u.Username) > 32:
		return fmt.Errorf("username must be less than 32 characters long")
	case !u.IsOauth2:
		for _, r := range u.Username {
			if !unicode.IsLower(r) {
				return fmt.Errorf("username must be made of only lowercase letters")
			}
		}
	}
	return nil
}

// validatePassword returns an error if the password is not valid.
func (u User) validatePassword() error {
	switch {
	case !u.IsOauth2 && len(u.Password) < 8:
		return fmt.Errorf("password must be at least 8 characters long")
	}
	return nil
}
