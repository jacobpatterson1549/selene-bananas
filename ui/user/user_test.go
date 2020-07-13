// +build js,wasm

package user

import (
	"testing"
)

func TestGetUser(t *testing.T) {
	getUserTests := []struct {
		jwt    string
		want   userInfo
		wantOk bool
	}{
		{
			jwt: "onlyTWO.parts",
		},
		{
			jwt: "a.bad-jwt-!!!.c",
		},
		{
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJleHAiOjMxNTM2MDAwLCJzdWIiOiJzZWxlbmUifQ.C4w9IKwB3k3Db40R5FYdeqfg66gYCF7l5s821WlLuJY",
			want: userInfo{
				username: "selene",
				points:   18,
			},
			wantOk: true,
		},
	}
	for i, test := range getUserTests {
		j := jwt(test.jwt)
		got, err := j.getUser()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}

func TestEscapePassword(t *testing.T) {
	init := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \^$*+?.()|[]{} but snowman should be unescaped: ☃`
	want := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \\\^\$\*\+\?\.\(\)\|\[\]\{\} but snowman should be unescaped: ☃`
	var cfg Config
	u := cfg.New(nil)
	got := u.escapePassword(init)
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}
