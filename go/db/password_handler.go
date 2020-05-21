package db

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type (
	passwordHandler interface {
		hashPassword(p password) (string, error)
		isCorrect(hashedPassword, p password) (bool, error)
	}

	bcryptPasswordHandler struct{}
	hashedPassword        []byte
)

func (p password) hash() (hashedPassword, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	return hashedPassword, nil
}

func (p password) isCorrect(hp hashedPassword) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hp, []byte(p))
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("checking if password is correct: %w", err)
	}
	return true, nil
}
