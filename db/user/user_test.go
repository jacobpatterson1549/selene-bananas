package user

import "testing"

func TestValidateUsername(t *testing.T) {
	isValidTests := []struct {
		username string
		isOauth2 bool
		want     bool
	}{
		{"", false, false}, // too short (< 1)
		{"", true, false},  // too short (< 1)
		{"selene", false, true},
		{"username", false, true},
		{"username-user", false, false},                     // invalid chars (numbers)
		{"oauth2-user", true, true},                         // invalid chars (numbers)
		{"abcdefghijklmnopqrstuvwxyzabcdef", false, true},   // 32
		{"abcdefghijklmnopqrstuvwxyzabcdefg", false, false}, // 33
		{"abcdefghijklmnopqrstuvwxyzabcdefg", true, false},  // 33
	}
	for i, test := range isValidTests {
		u := User{
			Username: test.username,
			IsOauth2: test.isOauth2,
		}
		err := u.validateUsername()
		got := err == nil
		if test.want != got {
			t.Errorf("Test %v: wanted username to be valid for '%+v' to be %v, but got %v", i, u, test.want, got)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	isValidTests := []struct {
		password string
		isOauth2 bool
		want     bool
	}{
		{"", false, false},
		{"", true, true},
		{"selene", false, false}, // too short
		{"password", false, true},
		{"password123", false, true},
		{"abcdefghijklmnopqrstuvwxyzabcdef", false, true},  // 32
		{"abcdefghijklmnopqrstuvwxyzabcdefg", false, true}, // 33
		{"abcdefghijklmnopqrstuvwxyzabcdefg", true, true},  // 33
	}
	for i, test := range isValidTests {
		u := User{
			Password: test.password,
			IsOauth2: test.isOauth2,
		}
		err := u.validatePassword()
		got := err == nil
		if test.want != got {
			t.Errorf("Test %v: wanted password to be valid for '%+v' to be %v, but got %v", i, u, test.want, got)
		}
	}
}

func TestUserValidate(t *testing.T) {
	newUserTests := []struct {
		username string
		password string
		isOauth2 bool
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
		{
			username: "oauth2-user",
			password: "",
			isOauth2: true,
			wantOk:   true,
		},
	}
	for i, test := range newUserTests {
		u := User{
			Username: test.username,
			Password: test.password,
			IsOauth2: test.isOauth2,
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
		case test.isOauth2 != u.IsOauth2:
			t.Errorf("Test %v: wanted user's isOauth2 setting to be %v, but was %v", i, test.isOauth2, u.IsOauth2)
		}
	}
}
