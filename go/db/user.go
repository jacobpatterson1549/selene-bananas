package db

import (
	"fmt"
	"unicode"
)

type (
	// User contains information for each player
	User struct {
		Username username
		password password
		Points   int
	}

	username string

	password string
)

// NewUser creates a new user with the specified name and password.
func NewUser(u, p string) (*User, error) {
	username, err := newUsername(u)
	if err != nil {
		return nil, err
	}
	password, err := newPassword(p)
	if err != nil {
		return nil, err
	}
	user := User{
		Username: *username,
		password: *password,
	}
	return &user, nil
}

func newUsername(u string) (*username, error) {
	switch {
	case len(u) < 1:
		return nil, fmt.Errorf("username required")
	case len(u) > 32:
		return nil, fmt.Errorf("username must be less than 32 characters long")
	default:
		for _, r := range u {
			if !unicode.IsLower(r) {
				return nil, fmt.Errorf("username must be made of only lowercase letters")
			}
		}
	}
	username := username(u)
	return &username, nil
}

func newPassword(p string) (*password, error) {
	switch {
	case len(p) < 8:
		return nil, fmt.Errorf("password must be at least 8 characters long")
	}
	password := password(p)
	return &password, nil
}
