package auth

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func TestCreate(t *testing.T) {
	tokenizer := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      []byte("secret"),
		timeFunc: func() int64 { return 0 },
		validSec: 365 * 24 * 60 * 60,
	}
	want := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJleHAiOjMxNTM2MDAwLCJzdWIiOiJzZWxlbmUifQ.X95h7cYKsUvmIT3yuhnxB0QjUNIgFKUNz-lD-d5GYhg"
	got, err := tokenizer.Create("selene", 18)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %v", err)
	case want != got:
		t.Errorf("could not create using simple key\nwanted %v\ngot    %v", want, got)
	}
}

func TestReadUsername(t *testing.T) {
	readTests := []struct {
		username              string
		creationSigningMethod jwt.SigningMethod
		readSigningMethod     jwt.SigningMethod
		want                  string
		wantOk                bool
	}{
		{
			username:              "selene",
			creationSigningMethod: jwt.SigningMethodHS256,
			readSigningMethod:     jwt.SigningMethodHS256,
			want:                  "selene",
			wantOk:                true,
		},
		{
			username:              "jacob",
			creationSigningMethod: jwt.SigningMethodHS512,
			readSigningMethod:     jwt.SigningMethodHS512,
			want:                  "jacob",
			wantOk:                true,
		},
		{
			username:              "selene",
			creationSigningMethod: jwt.SigningMethodHS512,
			readSigningMethod:     jwt.SigningMethodHS256,
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
		tokenString, err := creationTokenizer.Create(test.username, 0)
		if err != nil {
			t.Errorf("Test %v: unexpected error: %v", i, err)
			continue
		}
		var readTokenizer Tokenizer = jwtTokenizer{
			method:   test.readSigningMethod,
			key:      []byte("secret"),
			timeFunc: epochSecondsSupplier,
		}
		got, err := readTokenizer.ReadUsername(tokenString)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
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
		wantOk       bool
	}{
		{
			creationTime: 1,
			readTime:     0,
		},
		{
			creationTime: 2,
			readTime:     2,
			wantOk:       true,
		},
		{
			creationTime: 3,
			readTime:     5,
			wantOk:       true,
		},
		{
			creationTime: 100,
			readTime:     99 + validSecs,
			wantOk:       true,
		},
		// not working: https://github.com/dgrijalva/jwt-go/issues/340
		// {
		// 	creationTime: 100,
		// 	readTime:     100 + validSecs,
		// },
		{
			creationTime: 100,
			readTime:     101 + validSecs,
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
		var tokenizer Tokenizer = jwtTokenizer{
			method:   jwt.SigningMethodHS256,
			key:      []byte("secret"),
			timeFunc: epochSecondsSupplier,
			validSec: validSecs,
		}
		jwt.TimeFunc = func() time.Time {
			now := epochSecondsSupplier()
			return time.Unix(now, 0)
		}
		want := "selene"
		tokenString, _ := tokenizer.Create(want, 32)
		got, err := tokenizer.ReadUsername(tokenString)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestNewTokenizer(t *testing.T) {
	badCfg := TokenizerConfig{
		KeyReader: errorReader{fmt.Errorf("problem reading key")},
	}
	if _, err := badCfg.NewTokenizer(); err == nil {
		t.Errorf("expected error creating tokenizer with bad key reader")
	}
	key := []byte("secret")
	keyReader := bytes.NewReader(key)
	timeFunc := func() int64 { return 20 }
	validSec := int64(3600)
	cfg := TokenizerConfig{
		KeyReader: keyReader,
		TimeFunc:  timeFunc,
		ValidSec:  validSec,
	}
	want := jwtTokenizer{
		method:   jwt.SigningMethodHS256,
		key:      key,
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
		want.key == nil,
		gotJWT.timeFunc == nil,
		want.validSec != gotJWT.validSec:
		t.Errorf("not equal:\nwanted %v\ngot    %v", want, got)
	}
}

type errorReader struct {
	readErr error
}

func (r errorReader) Read(p []byte) (n int, err error) {
	return 0, r.readErr
}
