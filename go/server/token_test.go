package server

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jacobpatterson1549/selene-bananas/go/db"
)

func TestCreate(t *testing.T) {
	tokenizer := jwtTokenizer{
		method: jwt.SigningMethodHS256,
		key:    []byte("secret"),
		ess:    func() int64 { return 0 },
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

func TestRead(t *testing.T) {
	readTests := []struct {
		user                  db.User
		creationSigningMethod jwt.SigningMethod
		readSigningMethod     jwt.SigningMethod
		want                  db.Username
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
			method: test.creationSigningMethod,
			key:    []byte("secret"),
			ess:    epochSecondsSupplier,
		}
		tokenString, err := creationTokenizer.Create(test.user)
		if err != nil {
			t.Errorf("Test %v: unexpected error: %v", i, err)
			continue
		}
		readTokenizer := jwtTokenizer{
			method: test.readSigningMethod,
			key:    []byte("secret"),
			ess:    epochSecondsSupplier,
		}
		got, err := readTokenizer.Read(tokenString)
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

func TestCreateRead_time(t *testing.T) {
	// TODO: make these long test structures vars above the function declaration.  this will help the complexity
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
			readTime:     99 + tokenValidDurationSec,
			wantErr:      false,
		},
		// not working: https://github.com/dgrijalva/jwt-go/issues/340
		// {
		// 	creationTime: 100,
		// 	readTime:     100 + tokenValidDurationSec,
		// 	wantErr:      true,
		// },
		{
			creationTime: 100,
			readTime:     101 + tokenValidDurationSec,
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
			method: jwt.SigningMethodHS256,
			key:    []byte("secret"),
			ess:    epochSecondsSupplier,
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
		_, got := tokenizer.Read(tokenString)
		gotErr := got != nil
		if test.wantErr != gotErr {
			t.Errorf("Test %v: wanted error: %v, but got %v (%v)", i, test.wantErr, gotErr, got)
		}
	}
}
