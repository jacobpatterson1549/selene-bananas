package user

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/sql"
)



func TestNewDao(t *testing.T) {
	var mockDB mockDatabase
	var sqlDB sql.Database
	mockReadFileFunc := func(filename string) ([]byte, error) {
		return nil, nil
	}
	newDaoTests := []struct {
		db           db.Database
		readFileFunc func(filename string) ([]byte, error)
		wantOk       bool
	}{
		{
			readFileFunc: mockReadFileFunc,
		},
		{
			db: mockDB,
		},
		{
			db:           mockDB,
			readFileFunc: mockReadFileFunc,
		},
		{
			db:           sqlDB,
			readFileFunc: mockReadFileFunc,
			wantOk:       true,
		},
	}
	for i, test := range newDaoTests {
		cfg := DaoConfig{
			DB:           test.db,
			ReadFileFunc: test.readFileFunc,
		}
		d, err := cfg.NewDao()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(d.db, test.db):
			t.Errorf("Test %v: db not set", i)
		case d.readFileFunc == nil:
			t.Errorf("Test %v: readFileFunc not set", i)
		}
	}
}

func TestDaoSetup(t *testing.T) {
	setupTests := []struct {
		readFileErr error
		execFuncErr error
		wantOk      bool
	}{
		{
			readFileErr: fmt.Errorf("mock read file error"),
		},
		{
			execFuncErr: fmt.Errorf("exec transaction error"),
		},
		{
			wantOk: true,
		},
	}
	for i, test := range setupTests {
		d := Dao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...db.Query) error {
					return test.execFuncErr
				},
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return []byte("SETUP;"), test.readFileErr
			},
		}
		ctx := context.Background()
		err := d.Setup(ctx)
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
		dbExecErr            error
		incorrectPassword    bool
		isCorrectPasswordErr error
		want                 User
		wantOk               bool
	}{
		{
			rowScanErr: fmt.Errorf("problem reading user row"),
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
		s := mockScanner{
			scanFunc: func(dest ...interface{}) error {
				return test.rowScanErr
			},
		}
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
		got, err := d.Read(ctx, u)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
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
		s := mockScanner{
			scanFunc: func(dest ...interface{}) error {
				if len(dest) == 3 {
					if d, ok := dest[1].(*string); ok {
						*d = test.dbP
					}
				}
				return nil
			},
		}
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
	checkUpdateQueries := func(usernamePoints map[string]int, queries []db.Query) error {
		updatedUsernames := make(map[string]struct{}, len(usernamePoints))
		for i, q := range queries {
			args := q.Args()
			u, ok := args[0].(string)
			if !ok {
				return fmt.Errorf("query %v: arg0 was not a string", i)
			}
			p2, ok := args[1].(int)
			if !ok {
				return fmt.Errorf("query %v: arg1 was not an int", i)
			}
			p1, ok := usernamePoints[u]
			switch {
			case !ok:
				return fmt.Errorf("query %v: unwanted username: %v", i, u)
			case p1 != p2:
				return fmt.Errorf("query %v: wanted to update points for %v to %v, got: %v ", i, u, p1, p2)
			}
			updatedUsernames[u] = struct{}{}
		}
		if len(usernamePoints) != len(updatedUsernames) {
			return fmt.Errorf("wanted to update %v users, got %v", len(usernamePoints), len(updatedUsernames))
		}
		return nil
	}
	for i, test := range updatePointsIncrementTests {
		d := Dao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...db.Query) error {
					if test.dbExecErr != nil {
						return test.dbExecErr
					}
					return checkUpdateQueries(test.usernamePoints, queries)
				},
			},
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
		s := mockScanner{
			scanFunc: func(dest ...interface{}) error {
				return test.readErr
			},
		}
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
