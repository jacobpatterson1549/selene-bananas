package db

import (
	"testing"
)

type (
	mockPasswordHandler struct {
		hashFunc      func(password string) ([]byte, error)
		isCorrectFunc func(hashedPassword []byte, password string) (bool, error)
	}
)

func (ph mockPasswordHandler) hash(password string) ([]byte, error) {
	return ph.hashFunc(password)
}

func (ph mockPasswordHandler) isCorrect(hashedPassword []byte, password string) (bool, error) {
	return ph.isCorrectFunc(hashedPassword, password)
}

func TestIsValidateUsername(t *testing.T) {
	isValidTests := []struct {
		username string
		want     bool
	}{
		{"", false}, // too short (< 1)
		{"selene", true},
		{"username", true},
		{"username123", false}, // invalid chars (letters)
		{"abcdefghijklmnopqrstuvwxyzabcdef", true},   // 32
		{"abcdefghijklmnopqrstuvwxyzabcdefg", false}, // 33
	}
	for i, test := range isValidTests {
		err := validateUsername(test.username)
		got := err == nil
		if test.want != got {
			t.Errorf("Test %v: wanted username to be valid for '%v' to be %v, but got %v", i, test.username, test.want, got)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	isValidTests := []struct {
		password string
		want     bool
	}{
		{"", false},
		{"selene", false}, // too short
		{"password", true},
		{"password123", true},
		{"abcdefghijklmnopqrstuvwxyzabcdef", true},  // 32
		{"abcdefghijklmnopqrstuvwxyzabcdefg", true}, // 33
	}
	for i, test := range isValidTests {
		err := validatePassword(test.password)
		got := err == nil
		if test.want != got {
			t.Errorf("Test %v: wanted password to be valid for '%v' to be %v, but got %v", i, test.password, test.want, got)
		}
	}
}

func TestNewUser(t *testing.T) {
	newUserTests := []struct {
		username string
		password string
		wantOk   bool
	}{
		{},
		{
			username: "selene",
		},
		{
			password: "top_s3cr3t!",
		},
		{
			username: "selene",
			password: "top_s3cr3t!",
			wantOk:   true,
		},
	}
	for i, test := range newUserTests {
		u, err := NewUser(test.username, test.password)
		switch {
		case err != nil:
			switch {
			case test.wantOk:
				t.Errorf("Test %v: unexpected error: %v", i, err)
			case u != nil:
				t.Errorf("Test %v: expected nil user when error returned", i)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case test.username != string(u.Username):
			t.Errorf("Test %v: wanted user's username to be %v, but was %v", i, test.username, u.Username)
		case test.password != string(u.password):
			t.Errorf("Test %v: wanted user's password to be %v, but was %v", i, test.password, u.password)
		}
	}
}
