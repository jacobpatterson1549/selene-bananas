// Package bcrypt contains password hashing and checking logic for stored passwords.
package bcrypt

import "golang.org/x/crypto/bcrypt"

// PasswordHandler can hash and check passwords
type PasswordHandler struct {
	cost int
}

// NewPasswordHandler creates a password handler with the default cost
func NewPasswordHandler() PasswordHandler {
	bph := PasswordHandler{
		cost: bcrypt.DefaultCost,
	}
	return bph
}

// Hash computes the password hash from the supplied password
func (ph PasswordHandler) Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), ph.cost)
}

// IsCorrect determines if the hashed password matches the supplied password
func (PasswordHandler) IsCorrect(hashedPassword []byte, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	switch {
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
