package user

import "testing"

func TestValidateUsername(t *testing.T) {
	isValidTests := []struct {
		username string
		want     bool
	}{
		{"", false}, // too short (< 1)
		{"selene", true},
		{"username", true},
		{"username123", false}, // invalid chars (numbers)
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

func TestUserValidate(t *testing.T) {
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
		u := User{
			Username: test.username,
			Password: test.password,
		}
		err := u.Validate()
		switch {
		case err != nil:
			switch {
			case test.wantOk:
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case test.username != u.Username:
			t.Errorf("Test %v: wanted user's username to be %v, but was %v", i, test.username, u.Username)
		case test.password != u.Password:
			t.Errorf("Test %v: wanted user's password to be %v, but was %v", i, test.password, u.Password)
		}
	}
}
