package db

import (
	"fmt"
	"strings"
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
func NewUser(u, p string) (User, error) {
	var user User
	username := username(u)
	password := password(p)
	if !username.isValid() {
		return user, fmt.Errorf(username.helpText())
	}
	if !password.isValid() {
		return user, fmt.Errorf(password.helpText())
	}
	user = User{
		Username: username,
		password: password,
	}
	return user, nil
}

func (u username) isValid() bool {
	switch {
	case len(u) < 1:
		return false
	case len(u) > 32:
		return false
	default:
		validRunes := "abcdefghijklmnopqrstuvwxyz"
		for i := 0; i < len(u); i++ {
			if strings.IndexByte(validRunes, u[i]) < 0 {
				return false
			}
		}
	}
	return true
}

func (username) helpText() string {
	return "username must be made of only lowercase letters and be less than 32 characters long"
}

func (p password) isValid() bool {
	return len(p) >= 8
}

func (password) helpText() string {
	return "password must be at least 8 characters long"
}
