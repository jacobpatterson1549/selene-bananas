package postgres

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/db/sql"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

func TestUserBackendRead(t *testing.T) {
	tests := []struct {
		QueryErr error
		wantOk   bool
	}{
		{
			wantOk: true,
		},
		{
			QueryErr: fmt.Errorf("could not read user from mock"),
		},
	}
	for i, test := range tests {
		u := user.User{
			Username: "Billy",
			Password: "B0b",
		}
		want := &user.User{
			Username: "Billy",
			Password: "B0b",
			Points:   1955,
		}
		d := mockDatabase{
			QueryFunc: func(ctx context.Context, q sql.Query, dest ...interface{}) error {
				wantCmd := "SELECT username, password, points FROM user_read($1)"
				wantArgs := []interface{}{u.Username}
				switch {
				case !reflect.DeepEqual(wantCmd, q.Cmd()):
					t.Errorf("Test %v: query commands not equal: \n wanted: %q \n got:    %q", i, wantCmd, q.Cmd())
				case !reflect.DeepEqual(wantArgs, q.Args()):
					t.Errorf("Test %v: query commands not equal: \n wanted: %q \n got:    %q", i, wantArgs, q.Args())
				}
				*dest[0].(*string) = want.Username
				*dest[1].(*string) = want.Password
				*dest[2].(*int) = want.Points
				return test.QueryErr
			},
		}
		ub := UserBackend{
			Database: d,
		}
		ctx := context.Background()
		got, err := ub.Read(ctx, u)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(want, got):
			t.Errorf("Test %v: users not equal: \n wanted: %v \n got:    %v", i, want, got)
		}
	}
}

func TestUserBackendExecUser(t *testing.T) {
	tests := []struct {
		execErr error
		wantOk  bool
	}{
		{
			wantOk: true,
		},
		{
			execErr: fmt.Errorf("could not update password of user in mock"),
		},
	}
	type wantQuery struct {
		cmd  string
		args []interface{}
	}
	funcs := []struct {
		name        string
		f           func(ub UserBackend, ctx context.Context) error
		wantQueries []wantQuery
	}{
		{
			name: "Create",
			f: func(ub UserBackend, ctx context.Context) error {
				u := user.User{
					Username: "billy",
					Password: "B0b",
				}
				return ub.Create(ctx, u)
			},
			wantQueries: []wantQuery{
				{"SELECT user_create($1, $2)", []interface{}{"billy", "B0b"}},
			},
		},
		{
			name: "Update Password",
			f: func(ub UserBackend, ctx context.Context) error {
				u := user.User{
					Username: "billy",
					Password: "B0b",
				}
				return ub.UpdatePassword(ctx, u)
			},
			wantQueries: []wantQuery{
				{"SELECT user_update_password($1, $2)", []interface{}{"billy", "B0b"}},
			},
		},
		{
			name: "Update UserPoints",
			f: func(ub UserBackend, ctx context.Context) error {
				usernamePoints := map[string]int{
					"charlie": 7,
					"alice":   3,
					"billy":   1,
				}
				return ub.UpdatePointsIncrement(ctx, usernamePoints)
			},
			wantQueries: []wantQuery{
				{"SELECT user_update_points_increment($1, $2)", []interface{}{"alice", 3}},
				{"SELECT user_update_points_increment($1, $2)", []interface{}{"billy", 1}},
				{"SELECT user_update_points_increment($1, $2)", []interface{}{"charlie", 7}},
			},
		},
		{
			name: "Delete",
			f: func(ub UserBackend, ctx context.Context) error {
				u := user.User{
					Username: "billy",
					Password: "B0b",
				}
				return ub.Delete(ctx, u)
			},
			wantQueries: []wantQuery{
				{"SELECT user_delete($1)", []interface{}{"billy"}},
			},
		},
	}
	for _, f := range funcs {
		t.Run(f.name, func(t *testing.T) {
			for i, test := range tests {
				d := mockDatabase{
					ExecFunc: func(ctx context.Context, queries ...sql.Query) error {
						gotQueries := make([]wantQuery, len(queries))
						for i, q := range queries {
							gotQueries[i].cmd = q.Cmd()
							gotQueries[i].args = q.Args()
						}
						if !reflect.DeepEqual(f.wantQueries, gotQueries) {
							t.Errorf("Test %v: queries not equal: \n wanted: %q \n got:    %q", i, f.wantQueries, gotQueries)
						}
						return test.execErr
					},
				}
				ub := UserBackend{
					Database: d,
				}
				ctx := context.Background()
				err := f.f(ub, ctx)
				switch {
				case !test.wantOk:
					if err == nil {
						t.Errorf("Test %v: wanted error", i)
					}
				case err != nil:
					t.Errorf("Test %v: unwanted error: %v", i, err)
				}
			}
		})
	}
}
