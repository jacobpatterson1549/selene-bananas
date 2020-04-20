package server

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/dgrijalva/jwt-go"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Tokenizer creates and reads tokens from http traffic.
	Tokenizer interface {
		Create(u db.User) (string, error)
		Read(tokenString string) (db.Username, error)
	}

	jwtRSATokenizer struct {
		signingMethod *jwt.SigningMethodRSA
		key           *rsa.PrivateKey
	}

	jwtUsernameClaims struct {
		Username db.Username `json:"username"`
		Points   int         `json:"points"`
		jwt.StandardClaims
	}
)

const usernameClaimKey = "user"

// NewTokenizer creates a new jwt rsa tokenizer
func NewTokenizer() (Tokenizer, error) {
	return newJwtTokenizer(jwt.SigningMethodRS512, rand.Reader)
}

func newJwtTokenizer(method *jwt.SigningMethodRSA, keyReader io.Reader) (Tokenizer, error) {
	privateKey, err := rsa.GenerateKey(keyReader, 128)
	if err != nil {
		return nil, fmt.Errorf("generating key for Tokenizer: %w", err)
	}
	j := jwtRSATokenizer{
		signingMethod: method,
		key:           privateKey,
	}
	return j, nil
}

func (j jwtRSATokenizer) Create(u db.User) (string, error) {
	claims := &jwtUsernameClaims{
		u.Username,
		u.Points,
		jwt.StandardClaims{
			ExpiresAt: 15000,
			Issuer:    "test",
		},
	}
	token := jwt.NewWithClaims(j.signingMethod, claims)
	return token.SignedString(j.key)
}

func (j jwtRSATokenizer) Read(tokenString string) (db.Username, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtUsernameClaims{}, jwtRSATokenKeyFunc)
	if err != nil {
		return "", err
	}
	jwtUsernameClaims, ok := token.Claims.(jwtUsernameClaims)
	if !ok {
		return "", fmt.Errorf("wanted jwtUsernameClaims, but got %T", token.Claims)
	}
	err = jwtUsernameClaims.Valid()
	if err != nil {
		return "", fmt.Errorf("invalid claims: %w", err)
	}
	return jwtUsernameClaims.Username, nil
}

func jwtRSATokenKeyFunc(t *jwt.Token) (interface{}, error) {
	rsaSigningMethod, ok := t.Method.(*jwt.SigningMethodRSA)
	if !ok {
		return nil, fmt.Errorf("wanted gwt.SigningMethodRSA, but got %T", t.Method)
	}
	print(rsaSigningMethod)

	return nil, nil
}
