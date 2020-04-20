package server

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/dgrijalva/jwt-go"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

func TestNewJwtTokenizer(t *testing.T) {
	tokenizer, err := newJwtTokenizer(jwt.SigningMethodRS512, mockReader{})
	if err != nil {
		t.Fatal(err)
	}
	u := db.User{
		Username: "selene",
		Points:   54,
	}

	if j, ok := tokenizer.(jwtRSATokenizer); ok {
		fmt.Printf("privateKey: %v\n", j.key.N)
		fmt.Printf("publicKey:  %v\n", j.key.PublicKey.N)
	} else {
		t.Errorf("tokenizer is not jwtRSATokenizer, it is %T", j)
	}

	token, err := tokenizer.Create(u)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %v", err)
	case len(token) == 0:
		t.Error("expected non-empty token")
	default:
		fmt.Printf("token is: %v\n", token)
	}
}

// {"username":"selene","points":54}
func TestRead(t *testing.T) {
	readTests := []struct {
		tokenString string
		keyReader   io.Reader
		wantUser    db.User
		wantErr     bool
	}{
		{
			keyReader: mockReader{
				readErr: errors.New("read error"),
			},
			wantErr: true,
		},
		// {
		// 	tokenString
		// }
	}

	for i, test := range readTests {
		tokenizer, err := newJwtTokenizer(jwt.SigningMethodRS512, test.keyReader)
		if err != nil {
			t.Errorf("Test %v: unexpected error: %v", i, err)
		}

		gotUser, gotErr := tokenizer.Read(test.tokenString)
		switch {
		case gotErr != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, gotErr)
			}
		case !reflect.DeepEqual(test.wantUser, gotUser):
			t.Errorf("Test %v: wanted %v, got %v", i, test.wantUser, gotUser)
		}
	}
}

type mockReader struct {
	readErr error
}

func (r mockReader) Read(p []byte) (n int, err error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	for i := 0; i < len(p); i++ {
		p[i] = 'x'
	}
	return len(p), nil
}
