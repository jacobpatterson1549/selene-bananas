// Package auth contains code to ensure users are authorized to use the server after they have logged in.
package auth

import (
	"fmt"
	"io"

	"github.com/dgrijalva/jwt-go"
)

type (
	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(username string, points int) (string, error)
		ReadUsername(tokenString string) (string, error)
	}

	// TokenizerConfig contains fields which describe a Tokenizer
	TokenizerConfig struct {
		// KeyReader is used to generate token keys
		KeyReader io.Reader
		// TimeFunc is a function which should supply the current time since the unix epoch.
		// Used to set the the length of time the token is valid
		TimeFunc func() int64
		// ValidSec is the length of time the token is valid from the issuing time, in seconds
		ValidSec int64
	}

	jwtTokenizer struct {
		method   jwt.SigningMethod
		key      interface{}
		timeFunc func() int64
		validSec int64
	}

	jwtUserClaims struct {
		Points             int `json:"points"`
		jwt.StandardClaims     // username stored in Subject ("sub") field
	}
)

// NewTokenizer creates a Tokenizer that users the random number generator to generate tokens
func (cfg TokenizerConfig) NewTokenizer() (Tokenizer, error) {
	key := make([]byte, 64)
	if _, err := cfg.KeyReader.Read(key); err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	t := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      key,
		timeFunc: cfg.TimeFunc,
		validSec: cfg.ValidSec,
	}
	return t, nil
}

// Create converts a user to a token string
func (j jwtTokenizer) Create(username string, points int) (string, error) {
	now := j.timeFunc()
	expiresAt := now + j.validSec
	stdClaims := jwt.StandardClaims{
		Subject:   username,
		NotBefore: now,
		ExpiresAt: expiresAt,
	}
	claims := jwtUserClaims{
		Points:         points,
		StandardClaims: stdClaims,
	}
	token := jwt.NewWithClaims(j.method, claims)
	return token.SignedString(j.key)
}

// Read extracts the username from the token string
func (j jwtTokenizer) ReadUsername(tokenString string) (string, error) {
	var claims jwtUserClaims
	if _, err := jwt.ParseWithClaims(tokenString, &claims, j.keyFunc); err != nil {
		return "", err
	}
	return claims.Subject, nil
}

// keyFunc ensures the key type (method) of the token is correct before returning the key.
func (j jwtTokenizer) keyFunc(t *jwt.Token) (interface{}, error) {
	if t.Method != j.method {
		return nil, fmt.Errorf("incorrect authorization signing method")
	}
	return j.key, nil
}
