package db

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type (
	passwordHandler interface {
		hashPassword(password string) (string, error)
		isCorrect(password, hashedPassword string) (bool, error)
		isValid(password string) bool
	}

	bcryptPasswordHandler struct {
		minLength int
		maxLength int
	}
)

func (bcryptPasswordHandler) hashPassword(p password) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hashedPassword), nil
}

func (bcryptPasswordHandler) isCorrect(hashedPassword string, p password) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(p))
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("checking if password is correct: %w", err)
	}
	return true, nil
}
