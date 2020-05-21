package db

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type (
	hashedPassword []byte
)

func (p password) bytes() []byte {
	return []byte(p)
}

func (p password) hash() (hashedPassword, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(p.bytes(), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	return hashedPassword, nil
}

func (p password) isCorrect(hp hashedPassword) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hp, p.bytes())
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("checking if password is correct: %w", err)
	}
	return true, nil
}
