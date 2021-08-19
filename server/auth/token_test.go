package auth

import (
	"reflect"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

func TestCreate(t *testing.T) {
	tokenizer := JwtTokenizer{
		method: jwt.SigningMethodHS256,
		key:    []byte("secret"),
		TokenizerConfig: TokenizerConfig{
			TimeFunc: func() int64 { return 0 },
			ValidSec: 365 * 24 * 60 * 60,
		},
	}
	want := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJleHAiOjMxNTM2MDAwLCJzdWIiOiJzZWxlbmUifQ.X95h7cYKsUvmIT3yuhnxB0QjUNIgFKUNz-lD-d5GYhg"
	got, err := tokenizer.Create("selene", 18)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
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
		creationTokenizer := JwtTokenizer{
			method: test.creationSigningMethod,
			key:    []byte("secret"),
			TokenizerConfig: TokenizerConfig{
				TimeFunc: epochSecondsSupplier,
			},
		}
		tokenString, err := creationTokenizer.Create(test.username, 0)
		if err != nil {
			t.Errorf("Test %v: unwanted error: %v", i, err)
			continue
		}
		var readTokenizer = JwtTokenizer{
			method: test.readSigningMethod,
			key:    []byte("secret"),
			TokenizerConfig: TokenizerConfig{
				TimeFunc: epochSecondsSupplier,
			},
		}
		got, err := readTokenizer.ReadUsername(tokenString)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
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
		var tokenizer = JwtTokenizer{
			method: jwt.SigningMethodHS256,
			key:    []byte("secret"),
			TokenizerConfig: TokenizerConfig{
				TimeFunc: epochSecondsSupplier,
				ValidSec: validSecs,
			},
		}
		jwt.TimeFunc = func() time.Time {
			now := epochSecondsSupplier()
			return time.Unix(now, 0)
		}
		want := "selene"
		tokenString, err := tokenizer.Create(want, 32)
		if err != nil {
			t.Errorf("unwanted error: %v", err)
		}
		got, err := tokenizer.ReadUsername(tokenString)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case want != got:
			t.Errorf("Test %v: wanted %v, got %v", i, want, got)
		}
	}
}

func TestNewTokenizer(t *testing.T) {
	secretKey := []byte("secret")
	timeFunc := func() int64 { return 20 }
	newTokenizerTests := []struct {
		TokenizerConfig
		key    interface{}
		wantOk bool
		want   *JwtTokenizer
	}{
		{}, // no key
		{ // no time func
			key: secretKey,
		},
		{ // bad valid sec
			key: secretKey,
			TokenizerConfig: TokenizerConfig{
				TimeFunc: timeFunc,
			},
		},
		{ // ok
			key: secretKey,
			TokenizerConfig: TokenizerConfig{
				TimeFunc: timeFunc,
				ValidSec: 39,
			},
			wantOk: true,
			want: &JwtTokenizer{
				method: jwt.SigningMethodHS256,
				key:    secretKey,
				TokenizerConfig: TokenizerConfig{
					ValidSec: 39,
				},
			},
		},
	}
	for i, test := range newTokenizerTests {
		got, err := test.TokenizerConfig.NewTokenizer(test.key)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case got.TimeFunc == nil:
			t.Errorf("Test %v: time func not set", i)
		default:
			got.TimeFunc = nil
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	}
}
