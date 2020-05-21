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
		Read(tokenString string) (string, error) // TODO: rename to GetUsername
	}

	jwtTokenizer struct {
		method            jwt.SigningMethod
		key               interface{}
		usersTokenStrings map[string]string // TODO: rename to userTokens
		ess               epochSecondsSupplier
	}

	jwtUserClaims struct {
		Points             int `json:"points"`
		jwt.StandardClaims     // username stored in Subject ("sub") field
	}

	epochSecondsSupplier func() int64
)

const (
	tokenValidDurationSec int64 = 365 * 24 * 60 * 60 // 1 year
)

// NewTokenizer creates a Tokenizer that users the random number generator to generate tokens
func NewTokenizer(rand *rand.Rand) (Tokenizer, error) {
	ess := func() int64 {
		return time.Now().Unix()
	}
	return newTokenizer(rand, ess)
}

func newTokenizer(rand *rand.Rand, ess epochSecondsSupplier) (Tokenizer, error) {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("generating Tokenizer key: %w", err)
	}
	t := jwtTokenizer{
		method: jwt.SigningMethodHS256,
		key:    key,
		ess:    ess,
	}
	return t, nil
}

func (j jwtTokenizer) Create(u db.User) (string, error) {
	now := j.ess()
	expiresAt := now + tokenValidDurationSec
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

func (j jwtTokenizer) Read(tokenString string) (string, error) {
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
