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
		db           Database
		readFileFunc func(filename string) ([]byte, error)
		wantErr      bool
	}{
		{
			readFileFunc: func(filename string) ([]byte, error) {
				return nil, fmt.Errorf("mock read file error")
			},
			wantErr: true,
		},
		{
			db: mockDatabase{
				execTransactionFunc: func(ctx context.Context, queries []sqlQuery) error {
					return fmt.Errorf("exec transaction error")
				},
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return []byte("SELECT 1;"), nil
			},
			wantErr: true,
		},
		{
			db: mockDatabase{
				execTransactionFunc: func(ctx context.Context, queries []sqlQuery) error {
					return nil
				},
			},
			readFileFunc: func(filename string) ([]byte, error) {
				return []byte("SELECT 1;"), nil
			},
		},
	}
	for i, test := range setupTests {
		ud := UserDao{
			db:           test.db,
			readFileFunc: test.readFileFunc,
		}
		ctx := context.Background()
		err := ud.Setup(ctx)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		}
	}
}
