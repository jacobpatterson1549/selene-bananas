package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestNewUserDao(t *testing.T) {
	newUserDaoTests := []struct {
		db          Database
		queryPeriod time.Duration
		wantErr     bool
	}{
		{
			queryPeriod: 100,
			wantErr:     true,
		},
		{
			db:      mockDatabase{},
			wantErr: true,
		},
		{
			db:          mockDatabase{},
			queryPeriod: -100,
			wantErr:     true,
		},
		{
			db:          mockDatabase{},
			queryPeriod: 100,
		},
	}
	for i, test := range newUserDaoTests {
		cfg := UserDaoConfig{
			DB:          test.db,
			QueryPeriod: test.queryPeriod,
		}
		ud, err := cfg.NewUserDao()
		switch {
		case err != nil, ud == nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !reflect.DeepEqual(ud.db, test.db):
			t.Errorf("Test %v: db not set", i)
		case ud.queryPeriod != test.queryPeriod:
			t.Errorf("Test %v: queryPeriod not set", i)
		case ud.readFileFunc == nil:
			t.Errorf("Test %v: readFileFunc not set", i)
		}
	}
}

func TestUserDaoSetup(t *testing.T) {
	setupTests := []struct {
		readFileErr error
		execFuncErr error
		wantErr     bool
	}{
		{
			readFileErr: fmt.Errorf("mock read file error"),
			wantErr:     true,
		},
		{
			execFuncErr: fmt.Errorf("exec transaction error"),
			wantErr:     true,
		},
		{
			// [all ok]
		},
	}
	for i, test := range setupTests {
		ud := UserDao{
			db: mockDatabase{
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
					return test.execFuncErr
				},
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return []byte("SELECT 1;"), test.readFileErr
			},
		}
		ctx := context.Background()
		err := ud.Setup(ctx)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoCreate(t *testing.T) {
	createTests := []struct {
		userHashPasswordErr error
		dbExecErr           error
		wantErr             bool
	}{
		{
			userHashPasswordErr: fmt.Errorf("problem hashing password"),
			wantErr:             true,
		},
		{

			dbExecErr: fmt.Errorf("problem executing user create"),
			wantErr:   true,
		},
		{
			// [all ok]
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
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.Create(ctx, u)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
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
		wantErr              bool
	}{
		{
			rowScanErr: fmt.Errorf("problem reading user row"),
			wantErr:    true,
		},
		{
			isCorrectPasswordErr: fmt.Errorf("problem checking password"),
			wantErr:              true,
		},
		{
			incorrectPassword: true,
			wantErr:           true,
		},
		{
			// [all ok]
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
		r := mockRow{
			ScanFunc: func(dest ...interface{}) error {
				return test.rowScanErr
			},
		}
		ud := UserDao{
			db: mockDatabase{
				queryRowFunc: func(ctx context.Context, q sqlQuery) row {
					return r
				},
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		got, err := ud.Read(ctx, u)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		case test.want != got:
			t.Errorf("Test %v:\nwanted: %v\ngot:    : %v", i, test.want, got)
		}
	}
}

func TestUserDaoUpdatePassword(t *testing.T) {
	updatePasswordTests := []struct {
		oldP            string
		dbP             string
		newP            string // TODO: mock password validation
		hashPasswordErr error
		dbExecErr       error
		wantErr         bool
	}{
		{
			newP:    "tinyP",
			wantErr: true,
		},
		{
			newP:            "TOP_s3cr3t",
			hashPasswordErr: fmt.Errorf("problem hashing password"),
			wantErr:         true,
		},
		{
			oldP:    "homer_S!mps0n",
			dbP:     "el+bart0_rulZ",
			newP:    "TOP_s3cr3t",
			wantErr: true,
		},
		{
			oldP:    "homer_S!mps0n", // ensure the old password is compared to what is in the database
			dbP:     "el+bart0_rulZ",
			newP:    "el+bart0_rulZ",
			wantErr: true,
		},
		{
			oldP:      "homer_S!mps0n",
			dbP:       "homer_S!mps0n",
			newP:      "TOP_s3cr3t",
			dbExecErr: fmt.Errorf("problem updating password"),
			wantErr:   true,
		},
		{
			oldP: "homer_S!mps0n",
			dbP:  "homer_S!mps0n",
			newP: "TOP_s3cr3t",
			// [all ok]
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
		r := mockRow{
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
				queryRowFunc: func(ctx context.Context, q sqlQuery) row {
					return r
				},
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.UpdatePassword(ctx, u, test.newP)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoUpdatePointsIncrement(t *testing.T) {
	updatePointsIncrementTests := []struct {
		usernamePoints map[string]int
		dbExecErr      error
		wantErr        bool
	}{
		{
			dbExecErr: fmt.Errorf("problem updating users' points"),
			wantErr:   true,
		},
		{
			usernamePoints: map[string]int{
				"selene": 7,
				"fred":   1,
				"barney": 2,
			},
			// [all ok]
		},
	}
	checkUpdateQueries := func(usernamePoints map[string]int, queries []sqlQuery) error {
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
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
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
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		}
	}
}

func TestUserDaoDelete(t *testing.T) {
	deleteTests := []struct {
		readErr   error
		dbExecErr error
		wantErr   bool
	}{
		{
			readErr: fmt.Errorf("problem reading user"),
			wantErr: true,
		},
		{
			dbExecErr: fmt.Errorf("problem deleting user"),
			wantErr:   true,
		},
		{
			// [all ok]
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
		r := mockRow{
			ScanFunc: func(dest ...interface{}) error {
				return test.readErr
			},
		}
		ud := UserDao{
			db: mockDatabase{
				queryRowFunc: func(ctx context.Context, q sqlQuery) row {
					return r
				},
				execFunc: func(ctx context.Context, queries ...sqlQuery) error {
					return test.dbExecErr
				},
			},
		}
		ctx := context.Background()
		err := ud.Delete(ctx, u)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: expected error", i)
		}
	}
}
