package user

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestNewDao(t *testing.T) {
	newDaoTests := []struct {
		backend Backend
		wantOk  bool
	}{
		{},
		{
			backend: new(mockBackend),
			wantOk:  true,
		},
	}
	for i, test := range newDaoTests {
		d, err := NewDao(test.backend)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error creating new dao", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error creating new dao: %v", i, err)
		case d.backend == nil:
			t.Errorf("Test %v: db not set", i)
		}
	}
}

func TestDaoCreate(t *testing.T) {
	createTests := []struct {
		User
		userHashPasswordErr error
		dbExecErr           error
		wantOk              bool
	}{
		{
			User: User{
				Username: "JOHN",
				Password: "Doe12345",
			},
		},
		{
			User: User{
				Username: "john",
				Password: "short",
			},
		},
		{
			User: User{
				Username: "john",
				Password: "Doe12345",
			},
			userHashPasswordErr: fmt.Errorf("problem hashing password"),
		},
		{
			User: User{
				Username: "john",
				Password: "Doe12345",
			},
			dbExecErr: fmt.Errorf("problem executing user create"),
		},
		{
			User: User{
				Username: "john",
				Password: "Doe12345",
			},
			wantOk: true,
		},
	}
	for i, test := range createTests {
		ph := mockPasswordHandler{
			hashFunc: func(password string) ([]byte, error) {
				return []byte(password), test.userHashPasswordErr
			},
		}
		b := &mockBackend{
			createFunc: func(ctx context.Context, u User) error {
				return test.dbExecErr
			},
		}
		d := Dao{
			backend:         b,
			passwordHandler: ph,
		}
		ctx := context.Background()
		err := d.Create(ctx, test.User)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error creating user", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error creating user: %v", i, err)
		}
	}
}

func TestDaoLogin(t *testing.T) {
	loginTests := []struct {
		readErr              error
		incorrectPassword    bool
		isCorrectPasswordErr error
		want                 User
		wantOk               bool
		wantIncorrectLogin   bool
	}{
		{
			readErr: fmt.Errorf("problem reading user row"),
		},
		{
			isCorrectPasswordErr: fmt.Errorf("problem checking password"),
		},
		{
			incorrectPassword:  true,
			wantIncorrectLogin: true,
		},
		{
			wantOk: true,
		},
	}
	for i, test := range loginTests {
		var u User
		ph := mockPasswordHandler{
			isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
				return !test.incorrectPassword, test.isCorrectPasswordErr
			},
		}
		b := mockBackend{
			readFunc: func(ctx context.Context, u User) (*User, error) {
				return &test.want, test.readErr
			},
		}
		d := Dao{
			backend:         b,
			passwordHandler: ph,
		}
		ctx := context.Background()
		got, err := d.Login(ctx, u)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: unwanted error logging in user: %v", i, err)
			}
			if test.wantIncorrectLogin && err != ErrIncorrectLogin {
				t.Errorf("Test %v: errs not equal when the db has no rows: wanted %v, got: %v", i, ErrIncorrectLogin, err)
			}
		case err != nil:
			t.Errorf("Test %v: wanted error logging in user", i)
		case test.want != *got:
			t.Errorf("Test %v: users not equal:\nwanted: %v\ngot:    : %v", i, test.want, got)
		}
	}
	t.Run("NoDatabaseBackend", func(t *testing.T) {
		var b NoDatabaseBackend
		var ph passwordHandler
		d := Dao{
			backend:         b,
			passwordHandler: ph,
		}
		ctx := context.Background()
		u := User{
			Username: "selene",
		}
		got, err := d.Login(ctx, u)
		if err != nil {
			t.Errorf("unwanted error: %v", err)
		}
		if want := &u; !reflect.DeepEqual(want, got) {
			t.Errorf("wanted %v, got %v", want, got)
		}
	})
}

func TestDaoUpdatePassword(t *testing.T) {
	updatePasswordTests := []struct {
		oldP            string
		dbP             string
		newP            string
		hashPasswordErr error
		dbQueryErr      error
		dbExecErr       error
		wantOk          bool
	}{
		{
			newP: "tinyP",
		},
		{
			newP:            "TOP_s3cr3t",
			hashPasswordErr: fmt.Errorf("problem hashing password"),
		},
		{
			oldP: "homer_S!mps0n1",
			dbP:  "el+bart0_rulZ1",
			newP: "TOP_s3cr3t",
		},
		{
			oldP: "homer_S!mps0n2", // ensure the old password is compared to what is in the database
			dbP:  "el+bart0_rulZ2",
			newP: "el+bart0_rulZ2",
		},
		{
			oldP:       "homer_S!mps0n3",
			dbP:        "homer_S!mps0n3",
			newP:       "TOP_s3cr3t",
			dbQueryErr: fmt.Errorf("problem reading user"),
		},
		{
			oldP:       "homer_S!mps0n4",
			dbP:        "homer_S!mps0n4",
			newP:       "TOP_s3cr3t",
			dbQueryErr: ErrIncorrectLogin,
		},
		{
			oldP:      "homer_S!mps0n5",
			dbP:       "homer_S!mps0n5",
			newP:      "TOP_s3cr3t",
			dbExecErr: fmt.Errorf("problem updating password"),
		},
		{
			oldP:   "homer_S!mps0n6",
			dbP:    "homer_S!mps0n6",
			newP:   "TOP_s3cr3t",
			wantOk: true,
		},
	}
	for i, test := range updatePasswordTests {
		u := User{
			Username: "bart",
			Password: test.oldP,
		}
		ph := mockPasswordHandler{
			hashFunc: func(password string) ([]byte, error) {
				if password != test.newP {
					t.Errorf("Test %v: wanted to hash new password %v, got %v", i, test.newP, password)
				}
				return []byte(password), test.hashPasswordErr
			},
			isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
				return reflect.DeepEqual(hashedPassword, []byte(password)), nil
			},
		}
		b := mockBackend{
			readFunc: func(ctx context.Context, u User) (*User, error) {
				u2 := User{
					Password: test.dbP,
				}
				return &u2, test.dbQueryErr
			},
			updatePasswordFunc: func(ctx context.Context, u User) error {
				return test.dbExecErr
			},
		}
		d := Dao{
			backend:         b,
			passwordHandler: ph,
		}
		ctx := context.Background()
		err := d.UpdatePassword(ctx, u, test.newP)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error updating user passwords", i)
			}
			if test.dbQueryErr == ErrIncorrectLogin && err != ErrIncorrectLogin {
				t.Errorf("Test %v: error not passed through:\nwanted: %v\ngot:    %v", i, ErrIncorrectLogin, err)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error updating user passwords: %v", i, err)
		}
	}
}

func TestDaoUpdatePointsIncrement(t *testing.T) {
	updatePointsIncrementTests := []struct {
		usernamePoints map[string]int
		dbExecErr      error
		wantOk         bool
	}{
		{
			dbExecErr: fmt.Errorf("problem updating users' points"),
		},
		{
			usernamePoints: map[string]int{
				"selene": 7,
				"fred":   1,
				"barney": 2,
			},
			wantOk: true,
		},
	}
	for i, test := range updatePointsIncrementTests {
		b := mockBackend{
			updatePointsIncrementFunc: func(ctx context.Context, usernamePoints map[string]int) error {
				if want, got := test.usernamePoints, usernamePoints; !reflect.DeepEqual(want, got) {
					t.Errorf("usernamePoints not passed through exactly")
				}
				return test.dbExecErr
			},
		}
		d := Dao{
			backend: b,
		}
		ctx := context.Background()
		err := d.UpdatePointsIncrement(ctx, test.usernamePoints)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error incrementing user points", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error incrementing user points: %v", i, err)
		}
	}
}

func TestDaoDelete(t *testing.T) {
	deleteTests := []struct {
		dbQueryErr error
		dbExecErr  error
		wantOk     bool
	}{
		{
			dbQueryErr: fmt.Errorf("problem reading user"),
		},
		{
			dbQueryErr: ErrIncorrectLogin,
		},
		{
			dbExecErr: fmt.Errorf("problem deleting user"),
		},
		{
			wantOk: true,
		},
	}
	for i, test := range deleteTests {
		var u User
		ph := mockPasswordHandler{
			isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
				return true, nil
			},
		}
		b := mockBackend{
			readFunc: func(ctx context.Context, u User) (*User, error) {
				u2 := User{
					Password: u.Password,
				}
				return &u2, test.dbQueryErr
			},
			deleteFunc: func(ctx context.Context, u User) error {
				return test.dbExecErr
			},
		}
		d := Dao{
			backend:         b,
			passwordHandler: ph,
		}
		ctx := context.Background()
		err := d.Delete(ctx, u)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error deleting user", i)
			}
			if test.dbQueryErr == ErrIncorrectLogin && err != ErrIncorrectLogin {
				t.Errorf("Test %v: error not passed through:\nwanted: %v\ngot:    %v", i, ErrIncorrectLogin, err)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error deleting user: %v", i, err)
		}
	}
}

func TestDaoBackend(t *testing.T) {
	var b NoDatabaseBackend
	d := Dao{
		backend: b,
	}
	if want, got := b, d.Backend(); !reflect.DeepEqual(want, got) {
		t.Errorf("backends not equal: \n wanted: %v \n got:    %v", want, got)
	}
}
