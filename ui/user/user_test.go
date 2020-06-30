// +build js,wasm

package user

import (
	"testing"
)

func TestGetUser(t *testing.T) {
	getUserTests := []struct {
		jwt     string
		wantErr bool
		want    userInfo
	}{
		{
			jwt:     "onlyTWO.parts",
			wantErr: true,
		},
		{
			jwt:     "a.bad-jwt-!!!.c",
			wantErr: true,
		},
		{
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJleHAiOjMxNTM2MDAwLCJzdWIiOiJzZWxlbmUifQ.C4w9IKwB3k3Db40R5FYdeqfg66gYCF7l5s821WlLuJY",
			want: userInfo{
				username: "selene",
				points:   18,
			},
		},
	}
	for i, test := range getUserTests {
		j := jwt(test.jwt)
		got, err := j.getUser()
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case got == nil:
			t.Errorf("Test %v: expected result to not be nil", i)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}

func TestEscapePassword(t *testing.T) {
	init := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \^$*+?.()|[]{} but snowman should be unescaped: ☃`
	want := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \\\^\$\*\+\?\.\(\)\|\[\]\{\} but snowman should be unescaped: ☃`
	u := New(nil)
	got := u.escapePassword(init)
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}