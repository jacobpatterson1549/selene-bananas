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

func TestSetup(t *testing.T) {
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

func TestCreate(t *testing.T) {
	createTests := []struct {
		userHashPasswordErr error
		dbExecErr error
		wantErr bool
	}{
		{
			userHashPasswordErr: fmt.Errorf("problem hashing password"),
			wantErr: true,
		},
		{
			
			dbExecErr: fmt.Errorf("problem executing user create"),
			wantErr: true,
		},
		{
			// [all ok] 
		},
	}
	for i, test := range createTests {
		u:= User{
			ph: mockPasswordHandler{
				hashFunc: func() ([]byte, error) {
					return []byte("hashed password"), test.userHashPasswordErr
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
