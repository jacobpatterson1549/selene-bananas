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
