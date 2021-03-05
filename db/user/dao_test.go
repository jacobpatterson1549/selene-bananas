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
			db:     mockDatabase{},
			wantOk: true,
		},
	}
	for i, test := range newDaoTests {
		d, err := NewDao(test.db)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
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
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}

func TestDaoRead(t *testing.T) {
	readTests := []struct {
		rowScanErr           error
		incorrectPassword    bool
		isCorrectPasswordErr error
		want                 User
		wantOk               bool
	}{
		{
			rowScanErr: fmt.Errorf("problem reading user row"),
		},
		{
			rowScanErr: db.ErrNoRows,
		},
		{
			isCorrectPasswordErr: fmt.Errorf("problem checking password"),
		},
		{
			incorrectPassword: true,
		},
		{
			wantOk: true,
		},
	}
	for i, test := range readTests {
		u := User{
			ph: mockPasswordHandler{
				isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
					return !test.incorrectPassword, test.isCorrectPasswordErr
				},
			},
		}
		s := mockScanner(func(dest ...interface{}) error {
			return test.rowScanErr
		})
		d := Dao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q db.Query) db.Scanner {
					return s
				},
			},
		}
		ctx := context.Background()
		got, err := d.Read(ctx, u)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
			if test.rowScanErr == db.ErrNoRows && err != ErrIncorrectLogin {
				t.Errorf("Test %v: errs not equal when the db has no rows: wanted %v, got: %v", i, ErrIncorrectLogin, err)
			}
		case err != nil:
			t.Errorf("Test %v: wanted error", i)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted: %v\ngot:    : %v", i, test.want, got)
		}
	}
}

func TestDaoUpdatePassword(t *testing.T) {
	updatePasswordTests := []struct {
		oldP            string
		dbP             string
		newP            string
		hashPasswordErr error
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
			oldP: "homer_S!mps0n",
			dbP:  "el+bart0_rulZ",
			newP: "TOP_s3cr3t",
		},
		{
			oldP: "homer_S!mps0n", // ensure the old password is compared to what is in the database
			dbP:  "el+bart0_rulZ",
			newP: "el+bart0_rulZ",
		},
		{
			oldP:      "homer_S!mps0n",
			dbP:       "homer_S!mps0n",
			newP:      "TOP_s3cr3t",
			dbExecErr: fmt.Errorf("problem updating password"),
		},
		{
			oldP:   "homer_S!mps0n",
			dbP:    "homer_S!mps0n",
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
		s := mockScanner(func(dest ...interface{}) error {
			*dest[1].(*string) = test.dbP
			return nil
		})
		db := mockDatabase{
			queryFunc: func(ctx context.Context, q db.Query) db.Scanner {
				return s
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
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
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
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}

func TestDaoDelete(t *testing.T) {
	deleteTests := []struct {
		readErr   error
		dbExecErr error
		wantOk    bool
	}{
		{
			readErr: fmt.Errorf("problem reading user"),
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
		s := mockScanner(func(dest ...interface{}) error {
			return test.readErr
		})
		d := Dao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q db.Query) db.Scanner {
					return s
				},
				execFunc: func(ctx context.Context, queries ...db.Query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := d.Delete(ctx, u)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
}
