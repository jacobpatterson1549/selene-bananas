package db

import (
	"strings"
)

type (
	username string
	password string

	validator interface {
		isValid() bool
		helpText() string
	}
)

func (u username) isValid() bool {
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

func (username) helpText() string {
	return "username must be made of only lowercase letters and be less than 32 characters long"
}

func (p password) isValid() bool {
	return len(p) >= 8
}

func (password) helpText() string {
	return "password must be at least 8 characters long"
}
