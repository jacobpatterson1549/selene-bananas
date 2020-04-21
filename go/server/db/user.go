package db

import (
	"errors"
	"strings"
)

type (
	// User contains information for each player
	User struct {
		Username Username
		password password
		Points   int
	}

	// Username uniquely identifies a user
	Username string

	password string
)

// NewUser creates a new user with the specified name and password.
func NewUser(u, p string) (User, error) {
	var user User
	username := Username(u)
	password := password(p)
	if !username.isValid() {
		return user, errors.New(username.helpText())
	}
	if !password.isValid() {
		return user, errors.New(password.helpText())
	}
	user = User{
		Username: username,
		password: password,
	}
	return user, nil
}

func (u Username) isValid() bool {
	switch {
	case len(u) < 1:
		return false
	case len(u) > 32:
		return false
	default:
		validPasswordChars := "abcdefghijklmnopqrstuvwxyz"
		for i := 0; i < len(u); i++ {
			if strings.IndexByte(validPasswordChars, u[i]) < 0 {
				return false
			}
		}
	}
	return true
}

func (Username) helpText() string {
	return "username must be made of only lowercase letters and be less than 32 characters long"
}

func (p password) isValid() bool {
	return len(p) >= 8
}

func (password) helpText() string {
	return "password must be at least 8 characters long"
}
