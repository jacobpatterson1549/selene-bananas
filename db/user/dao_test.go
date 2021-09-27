package user

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db"
)

func TestNewDao(t *testing.T) {
	newDaoTests := []struct {
		db     Database
		wantOk bool
	}{
		{},
		{
			db:     new(mockDatabase),
			wantOk: true,
		},
	}
	for i, test := range newDaoTests {
		d, err := NewDao(test.db)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error creating new dao", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error creating new dao: %v", i, err)
		case d.db == nil:
			t.Errorf("Test %v: db not set", i)
		}
	}
}

func TestDaoCreate(t *testing.T) {
	createTests := []struct {
		userHashPasswordErr error
		dbExecErr           error
		wantOk              bool
	}{
		{
			userHashPasswordErr: fmt.Errorf("problem hashing password"),
		},
		{

			dbExecErr: fmt.Errorf("problem executing user create"),
		},
		{
			wantOk: true,
		},
	}
	for i, test := range createTests {
		u := User{
			ph: mockPasswordHandler{
				hashFunc: func(password string) ([]byte, error) {
					return []byte(password), test.userHashPasswordErr
				},
			},
		}
		d := Dao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...db.Query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := d.Create(ctx, u)
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
		rowScanErr           error
		incorrectPassword    bool
		isCorrectPasswordErr error
		want                 User
		wantOk               bool
		wantIncorrectLogin   bool
	}{
		{
			rowScanErr: fmt.Errorf("problem reading user row"),
		},
		{
			rowScanErr:         db.ErrNoRows,
			wantIncorrectLogin: true,
		},
		{
			rowScanErr:         fmt.Errorf("wrapped db.ErrNoRows: %w", db.ErrNoRows),
			wantIncorrectLogin: true,
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
		u := User{
			ph: mockPasswordHandler{
				isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
					return !test.incorrectPassword, test.isCorrectPasswordErr
				},
			},
		}
		d := Dao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q db.Query, dest ...interface{}) error {
					return test.rowScanErr
				},
			},
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
			password: test.oldP,
			ph: mockPasswordHandler{
				hashFunc: func(password string) ([]byte, error) {
					if password != test.newP {
						t.Errorf("Test %v: wanted to hash new password %v, got %v", i, test.newP, password)
					}
					return []byte(password), test.hashPasswordErr
				},
				isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
					return reflect.DeepEqual(hashedPassword, []byte(password)), nil
				},
			},
		}
		db := mockDatabase{
			queryFunc: func(ctx context.Context, q db.Query, dest ...interface{}) error {
				*dest[1].(*string) = test.dbP
				return test.dbQueryErr
			},
			execFunc: func(ctx context.Context, queries ...db.Query) error {
				return test.dbExecErr
			},
		}
		d := Dao{
			db: db,
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
		db := mockDatabase{
			execFunc: func(ctx context.Context, queries ...db.Query) error {
				if test.dbExecErr != nil {
					return test.dbExecErr
				}
				updatedUsernamePoints := make(map[string]int, len(queries))
				for _, q := range queries {
					u := q.Args()[0].(string)
					p := q.Args()[1].(int)
					updatedUsernamePoints[u] = p
				}
				if !reflect.DeepEqual(test.usernamePoints, updatedUsernamePoints) {
					return fmt.Errorf("Test %v: update usernamePoints not equal:\nwanted: %v\ngot:    %v", i, test.usernamePoints, updatedUsernamePoints)
				}
				return nil
			},
		}
		d := Dao{
			db: db,
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
		u := User{
			ph: mockPasswordHandler{
				isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
					return true, nil
				},
			},
		}
		d := Dao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q db.Query, dest ...interface{}) error {
					return test.dbQueryErr
				},
				execFunc: func(ctx context.Context, queries ...db.Query) error {
					return test.dbExecErr
				},
			},
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
