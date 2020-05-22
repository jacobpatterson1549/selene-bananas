package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jacobpatterson1549/selene-bananas/go/db"
)

type (
	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(u db.User) (string, error)
		ReadUsername(tokenString string) (string, error)
	}

	// TokenizerConfig contains fields which describe a Tokenizer
	TokenizerConfig struct {
		// Rand is used to generate token keys
		Rand *rand.Rand
		// ValidSec is the length of time the token is valid from the issuing time, in seconds
		ValidSec int64
	}

	jwtTokenizer struct {
		method   jwt.SigningMethod
		key      interface{}
		ess      epochSecondsSupplier
		validSec int64
	}

	jwtUserClaims struct {
		Points             int `json:"points"`
		jwt.StandardClaims     // username stored in Subject ("sub") field
	}

	epochSecondsSupplier func() int64
)

// NewTokenizer creates a Tokenizer that users the random number generator to generate tokens
func (cfg TokenizerConfig) NewTokenizer() (Tokenizer, error) {
	ess := func() int64 {
		return time.Now().Unix()
	}
	key := make([]byte, 64)
	_, err := cfg.Rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	t := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      key,
		ess:      ess,
		validSec: cfg.ValidSec,
	}
	return t, nil
}

// Create converts a user to a token string
func (j jwtTokenizer) Create(u db.User) (string, error) {
	now := j.ess()
	expiresAt := now + j.validSec
	claims := &jwtUserClaims{
		u.Points,
		jwt.StandardClaims{
			Subject:   string(u.Username),
			NotBefore: now,
			ExpiresAt: expiresAt,
		},
	}
	token := jwt.NewWithClaims(j.method, claims)
	return token.SignedString(j.key)
}

// Read extracts the username from the token string
func (j jwtTokenizer) ReadUsername(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtUserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != j.method {
			return nil, fmt.Errorf("incorrect authorization signing method")
		}
		return j.key, nil
	})
	if err != nil {
		return "", err
	}
	jwtUserClaims, ok := token.Claims.(*jwtUserClaims)
	if !ok {
		return "", fmt.Errorf("wanted *jwtUserClaims, but got %T", token.Claims)
	}
	return jwtUserClaims.Subject, nil
}
