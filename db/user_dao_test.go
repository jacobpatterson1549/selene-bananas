package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestNewUserDao(t *testing.T) {
	var mockDatabase mockDatabase
	mockReadFileFunc := func(filename string) ([]byte, error) {
		return nil, nil
	}
	newUserDaoTests := []struct {
		db           Database
		readFileFunc func(filename string) ([]byte, error)
		wantOk       bool
	}{
		{
			readFileFunc: mockReadFileFunc,
		},
		{
			db: mockDatabase,
		},
		{
			db:           mockDatabase,
			readFileFunc: mockReadFileFunc,
			wantOk:       true,
		},
	}
	for i, test := range newUserDaoTests {
		cfg := UserDaoConfig{
			DB:           test.db,
			ReadFileFunc: test.readFileFunc,
		}
		ud, err := cfg.NewUserDao()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case !reflect.DeepEqual(ud.db, test.db):
			t.Errorf("Test %v: db not set", i)
		case ud.readFileFunc == nil:
			t.Errorf("Test %v: readFileFunc not set", i)
		}
	}
}

func TestUserDaoSetup(t *testing.T) {
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
		ud := UserDao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...query) error {
					return test.execFuncErr
				},
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return []byte("SETUP;"), test.readFileErr
			},
		}
		ctx := context.Background()
		err := ud.Setup(ctx)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoCreate(t *testing.T) {
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
		ud := UserDao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.Create(ctx, u)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoRead(t *testing.T) {
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
			ScanFunc: func(dest ...interface{}) error {
				return test.rowScanErr
			},
		}
		ud := UserDao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q query) scanner {
					return s
				},
				execFunc: func(ctx context.Context, queries ...query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		got, err := ud.Read(ctx, u)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted: %v\ngot:    : %v", i, test.want, got)
		}
	}
}

func TestUserDaoUpdatePassword(t *testing.T) {
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
					return []byte(password), test.hashPasswordErr
				},
				isCorrectFunc: func(hashedPassword []byte, password string) (bool, error) {
					return reflect.DeepEqual(hashedPassword, []byte(password)), nil
				},
			},
		}
		s := mockScanner{
			ScanFunc: func(dest ...interface{}) error {
				if len(dest) == 3 {
					if d, ok := dest[1].(*string); ok {
						*d = test.dbP
					}
				}
				return nil
			},
		}
		ud := UserDao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q query) scanner {
					return s
				},
				execFunc: func(ctx context.Context, queries ...query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.UpdatePassword(ctx, u, test.newP)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoUpdatePointsIncrement(t *testing.T) {
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
	checkUpdateQueries := func(usernamePoints map[string]int, queries []query) error {
		updatedUsernames := make(map[string]struct{}, len(usernamePoints))
		for i, q := range queries {
			args := q.args()
			u, ok := args[0].(string)
			if !ok {
				return fmt.Errorf("Test %v: arg0 was not a string", i)
			}
			p2, ok := args[1].(int)
			if !ok {
				return fmt.Errorf("query %v: arg1 was not an int", i)
			}
			p1, ok := usernamePoints[u]
			switch {
			case !ok:
				return fmt.Errorf("query %v: unexpected username: %v", i, u)
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
		ud := UserDao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...query) error {
					if test.dbExecErr != nil {
						return test.dbExecErr
					}
					return checkUpdateQueries(test.usernamePoints, queries)
				},
			},
		}
		ctx := context.Background()
		usernames := make([]string, 0, len(test.usernamePoints))
		for u := range test.usernamePoints {
			usernames = append(usernames, u)
		}
		f := func(username string) int {
			return test.usernamePoints[username]
		}
		err := ud.UpdatePointsIncrement(ctx, usernames, f)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoDelete(t *testing.T) {
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
			ScanFunc: func(dest ...interface{}) error {
				return test.readErr
			},
		}
		ud := UserDao{
			db: mockDatabase{
				queryFunc: func(ctx context.Context, q query) scanner {
					return s
				},
				execFunc: func(ctx context.Context, queries ...query) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.Delete(ctx, u)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		}
	}
}
