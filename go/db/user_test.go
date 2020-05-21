package db

import (
	"testing"
)

func TestIsValidUsername(t *testing.T) {
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
		_, err := newUsername(test.username)
		got := err == nil
		if test.want != got {
			t.Errorf("Test %v: wanted username to be valid for '%v' to be %v, but got %v", i, test.username, test.want, got)
		}
	}
}

func TestIsValid(t *testing.T) {
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
		_, err := newPassword(test.password)
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
		wantErr  bool
	}{
		{
			wantErr: true,
		},
		{
			username: "selene",
			wantErr:  true,
		},
		{
			password: "top_s3cr3t!",
			wantErr:  true,
		},
		{
			username: "selene",
			password: "top_s3cr3t!",
		},
	}
	for i, test := range newUserTests {
		u, err := NewUser(test.username, test.password)
		switch {
		case err != nil:
			switch {
			case !test.wantErr:
				t.Errorf("Test %v: unexpected error: %v", i, err)
			case u != nil:
				t.Errorf("Test %v: expected nil user when error returned", i)
			}
		case test.username != string(u.Username):
			t.Errorf("Test %v: wanted user's username to be %v, but was %v", i, test.username, u.Username)
		case test.password != string(u.password):
			t.Errorf("Test %v: wanted user's password to be %v, but was %v", i, test.password, u.password)
		}
	}
}
