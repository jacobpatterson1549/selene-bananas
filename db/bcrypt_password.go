package db

import (
	"golang.org/x/crypto/bcrypt"
)

type (
	// bcryptPassword implements the passwordHandler interface
	bcryptPassword string
)

func (p bcryptPassword) hash() ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
}

func (p bcryptPassword) isCorrect(hashedPassword []byte) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(p))
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
