package server

import (
	"testing"

	"github.com/dgrijalva/jwt-go"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

func TestCreate(t *testing.T) {
	tokenizer := jwtTokenizer{
		method: jwt.SigningMethodHS256,
		key:    []byte("secret"),
	}
	u := db.User{
		Username: "selene",
		Points:   18,
	}
	want := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InNlbGVuZSIsInBvaW50cyI6MTh9.68MAfIB-QQDvY6b7uoOoxsuhd8oi78YZLDBW8kpEi_E"
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
		tokenString   string
		signingMethod jwt.SigningMethod
		want          db.Username
		wantErr       bool
	}{
		{
			tokenString:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InNlbGVuZSIsInBvaW50cyI6MTh9.68MAfIB-QQDvY6b7uoOoxsuhd8oi78YZLDBW8kpEi_E",
			signingMethod: jwt.SigningMethodHS256,
			want:          "selene",
		},
		{
			tokenString:   "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImphY29iIiwicG9pbnRzIjo3fQ.X9ky2F644YutBnsJlokLz2p4tgEO6dxpk3nuLDbGohOMMPk8ix2DgI3E-iXTowKhQJL-cLyRdXIaZVWQXYMFUg",
			signingMethod: jwt.SigningMethodHS512,
			want:          "jacob",
		},
		{
			tokenString:   "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImphY29iIiwicG9pbnRzIjo3fQ.X9ky2F644YutBnsJlokLz2p4tgEO6dxpk3nuLDbGohOMMPk8ix2DgI3E-iXTowKhQJL-cLyRdXIaZVWQXYMFUg",
			signingMethod: jwt.SigningMethodHS256,
			wantErr:       true, // should be SigningMethodES512
		},
		{
			tokenString:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoic2VsZW5lIn0.0hzVY3vo48b6Kjl2-iXfEvSAzlYhg8WotQTD_l1426s",
			signingMethod: jwt.SigningMethodHS256,
			wantErr:       true, // payload is {"user":"selene"}, not a jwtUsernameClaims ("username" is empty)
		},
	}
	for i, test := range readTests {
		var tokenizer Tokenizer
		tokenizer = jwtTokenizer{
			method: test.signingMethod,
			key:    []byte("secret"),
		}
		got, err := tokenizer.Read(test.tokenString)
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
