// Package auth contains code to ensure users are authorized to use the server after they have logged in.
package auth

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

type (
	// TokenizerConfig contains fields which describe a Tokenizer.
	TokenizerConfig struct {
		// TimeFunc is a function which should supply the current time since the unix epoch.
		// This is used for setting token lifespan.
		TimeFunc func() int64
		// ValidSec is the length of time the token is valid from the issuing time, in seconds.
		ValidSec int64
	}

	// JwtTokenizer creates java web tokens.
	JwtTokenizer struct {
		method jwt.SigningMethod
		key    interface{}
		TokenizerConfig
	}

	// jwtUserClaims appends user points to the standard jwt claims.
	jwtUserClaims struct {
		Points             int `json:"points"`
		jwt.StandardClaims     // username stored in Subject ("sub") field
	}
)

// NewTokenizer creates a Tokenizer that users the random number generator to generate tokens.
func (cfg TokenizerConfig) NewTokenizer(key interface{}) (*JwtTokenizer, error) {
	if err := cfg.validate(key); err != nil {
		return nil, fmt.Errorf("creating tokenizer: validation: %w", err)
	}
	t := JwtTokenizer{
		method:          jwt.SigningMethodHS256,
		key:             key,
		TokenizerConfig: cfg,
	}
	return &t, nil
}

// validate ensures the configuration has no errors.
func (cfg TokenizerConfig) validate(key interface{}) error {
	switch {
	case key == nil:
		return fmt.Errorf("log required")
	case cfg.TimeFunc == nil:
		return fmt.Errorf("time func required")
	case cfg.ValidSec <= 0:
		return fmt.Errorf("non-negative valid seconds required")
	}
	return nil
}

// Create converts a user to a token string.
func (j *JwtTokenizer) Create(username string, points int) (string, error) {
	now := j.TimeFunc()
	expiresAt := now + j.ValidSec
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

// ReadUsername extracts the username from the token string.
func (j *JwtTokenizer) ReadUsername(tokenString string) (string, error) {
	var claims jwtUserClaims
	if _, err := jwt.ParseWithClaims(tokenString, &claims, j.keyFunc); err != nil {
		return "", err
	}
	return claims.Subject, nil
}

// keyFunc ensures the key type (method) of the token is correct before returning the key.
func (j *JwtTokenizer) keyFunc(t *jwt.Token) (interface{}, error) {
	if t.Method != j.method {
		return nil, fmt.Errorf("incorrect authorization signing method")
	}
	return j.key, nil
}
