package server

import (
	"fmt"
	"math/rand"

	"github.com/dgrijalva/jwt-go"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(u db.User) (string, error)
		Read(tokenString string) (db.Username, error)
	}

	jwtTokenizer struct {
		method jwt.SigningMethod
		key    interface{}
	}

	jwtUsernameClaims struct {
		Username db.Username `json:"username"`
		Points   int         `json:"points"`
		jwt.StandardClaims
	}
)

const usernameClaimKey = "user"

func newTokenizer(rand *rand.Rand) (Tokenizer, error) {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	t := jwtTokenizer{
		method: jwt.SigningMethodHS256,
		key:    key,
	}
	return t, nil
}

func (j jwtTokenizer) Create(u db.User) (string, error) {
	claims := &jwtUsernameClaims{
		u.Username,
		u.Points,
		jwt.StandardClaims{
			//ExpiresAt: 0, // TODO: add token expiry, issuer,
		},
	}
	token := jwt.NewWithClaims(j.method, claims)
	return token.SignedString(j.key)
}

func (j jwtTokenizer) Read(tokenString string) (db.Username, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtUsernameClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != j.method {
			return nil, fmt.Errorf("incorrect authorization signing method")
		}
		return j.key, nil
	})
	if err != nil {
		return "", err
	}
	jwtUsernameClaims, ok := token.Claims.(*jwtUsernameClaims)
	if !ok {
		return "", fmt.Errorf("wanted *jwtUsernameClaims, but got %T", token.Claims)
	}
	err = jwtUsernameClaims.Valid()
	if err != nil {
		return "", fmt.Errorf("invalid claims: %w", err)
	}
	return jwtUsernameClaims.Username, nil
}
