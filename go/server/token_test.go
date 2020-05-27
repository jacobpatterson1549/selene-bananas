package server

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jacobpatterson1549/selene-bananas/go/db"
)

func TestCreate(t *testing.T) {
	tokenizer := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      []byte("secret"),
		timeFunc: func() int64 { return 0 },
		validSec: 365 * 24 * 60 * 60,
	}
	u := db.User{
		Username: "selene",
		Points:   18,
	}
	want := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJleHAiOjMxNTM2MDAwLCJzdWIiOiJzZWxlbmUifQ.X95h7cYKsUvmIT3yuhnxB0QjUNIgFKUNz-lD-d5GYhg"
	got, err := tokenizer.Create(u)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %v", err)
	case want != got:
		t.Errorf("could not create using simple key\nwanted %v\ngot    %v", want, got)
	}
}

func TestReadUsername(t *testing.T) {
	readTests := []struct {
		user                  db.User
		creationSigningMethod jwt.SigningMethod
		readSigningMethod     jwt.SigningMethod
		want                  string
		wantErr               bool
	}{
		{
			user:                  db.User{Username: "selene"},
			creationSigningMethod: jwt.SigningMethodHS256,
			readSigningMethod:     jwt.SigningMethodHS256,
			want:                  "selene",
		},
		{
			user:                  db.User{Username: "jacob"},
			creationSigningMethod: jwt.SigningMethodHS512,
			readSigningMethod:     jwt.SigningMethodHS512,
			want:                  "jacob",
		},
		{
			user:                  db.User{Username: "selene"},
			creationSigningMethod: jwt.SigningMethodHS512,
			readSigningMethod:     jwt.SigningMethodHS256,
			wantErr:               true,
		},
	}
	jwt.TimeFunc = func() time.Time { return time.Unix(0, 0) }
	epochSecondsSupplier := func() int64 { return 0 }
	for i, test := range readTests {
		creationTokenizer := jwtTokenizer{
			method:   test.creationSigningMethod,
			key:      []byte("secret"),
			timeFunc: epochSecondsSupplier,
		}
		tokenString, err := creationTokenizer.Create(test.user)
		if err != nil {
			t.Errorf("Test %v: unexpected error: %v", i, err)
			continue
		}
		readTokenizer := jwtTokenizer{
			method:   test.readSigningMethod,
			key:      []byte("secret"),
			timeFunc: epochSecondsSupplier,
		}
		got, err := readTokenizer.ReadUsername(tokenString)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}

func TestCreateReadWithTime(t *testing.T) {
	const validSecs int64 = 1000
	readTests := []struct {
		creationTime int64 // not before
		readTime     int64 // not equal or after
		wantErr      bool
	}{
		{
			creationTime: 1,
			readTime:     0,
			wantErr:      true,
		},
		{
			creationTime: 2,
			readTime:     2,
			wantErr:      false,
		},
		{
			creationTime: 3,
			readTime:     5,
			wantErr:      false,
		},
		{
			creationTime: 100,
			readTime:     99 + validSecs,
			wantErr:      false,
		},
		// not working: https://github.com/dgrijalva/jwt-go/issues/340
		// {
		// 	creationTime: 100,
		// 	readTime:     100 + validSecs,
		// 	wantErr:      true,
		// },
		{
			creationTime: 100,
			readTime:     101 + validSecs,
			wantErr:      true,
		},
	}
	for i, test := range readTests {
		j := 0
		epochSecondsSupplier := func() int64 {
			j++
			switch j {
			case 1:
				return test.creationTime
			case 2:
				return test.readTime
			default:
				return -1
			}
		}
		var tokenizer Tokenizer
		tokenizer = jwtTokenizer{
			method:   jwt.SigningMethodHS256,
			key:      []byte("secret"),
			timeFunc: epochSecondsSupplier,
			validSec: validSecs,
		}
		jwt.TimeFunc = func() time.Time {
			now := epochSecondsSupplier()
			return time.Unix(now, 0)
		}
		u := db.User{
			Username: "selene",
			Points:   5,
		}
		tokenString, err := tokenizer.Create(u)
		if err != nil {
			t.Errorf("Test %v: unexpected error: %v", i, err)
			continue
		}
		_, got := tokenizer.ReadUsername(tokenString)
		gotErr := got != nil
		if test.wantErr != gotErr {
			t.Errorf("Test %v: wanted error: %v, but got %v (%v)", i, test.wantErr, gotErr, got)
		}
	}
}

func TestNewTokenizer(t *testing.T) {
	src := rand.NewSource(0) // make the key predictable
	rand := rand.New(src)
	timeFunc := func() int64 { return 20 }
	validSec := int64(3600)
	cfg := TokenizerConfig{
		Rand:     rand,
		TimeFunc: timeFunc,
		ValidSec: validSec,
	}
	wantKey := []byte{1, 148, 253, 194, 250, 47, 252, 192, 65, 211, 255, 18, 4, 91, 115, 200, 110, 79, 249, 95, 246, 98, 165, 238, 232, 42, 189, 244, 74, 45, 11, 117, 251, 24, 13, 175, 72, 167, 158, 224, 177, 13, 57, 70, 81, 133, 15, 212, 161, 120, 137, 46, 226, 133, 236, 225, 81, 20, 85, 120, 8, 117, 214, 78}
	want := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      wantKey,
		timeFunc: timeFunc,
		validSec: validSec,
	}
	got, err := cfg.NewTokenizer()
	gotJWT, ok := got.(jwtTokenizer)
	switch {
	case err != nil:
		t.Errorf("unexpected error")
	case !ok:
		t.Errorf("expected jwtTokenizer, got %T", got)
	case want.method != gotJWT.method,
		!reflect.DeepEqual(want.key, gotJWT.key),
		gotJWT.timeFunc == nil, // '!=' and reflect.DeepEquals do not work with non-nil functions
		want.validSec != gotJWT.validSec:
		t.Errorf("not equal:\nwanted %v\ngot    %v", want, got)
	}
}
