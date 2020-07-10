package db

import (
	"golang.org/x/crypto/bcrypt"
)

type (
	// bcryptPasswordHandler implements the passwordHandler interface
	bcryptPasswordHandler struct {
		cost int
	}
)

func newBcryptPasswordHandler() passwordHandler {
	bph := bcryptPasswordHandler{
		cost: bcrypt.DefaultCost,
	}
	return bph
}

func (bph bcryptPasswordHandler) hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bph.cost)
}

func (bcryptPasswordHandler) isCorrect(hashedPassword []byte, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
