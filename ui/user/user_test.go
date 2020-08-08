// +build js,wasm

package user

import "testing"

func TestGetUser(t *testing.T) {
	getUserTests := []struct {
		jwt    string
		want   userInfo
		wantOk bool
	}{
		// use jwt alg: HS256, secret: s3cr3t
		{},
		{
			jwt: "onlyTWO.parts",
		},
		{ // invalid base64
			jwt: "a.bad-jwt-!!!.c",
		},
		{ // has key of Sub, not sub
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJTdWIiOiJzZWxlbmUifQ.GN3dIGP0ENeN1SC78ByrW4dmlm2qBP9XVeACAclGhZ8",
		},
		{ // sub is not a string
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJzdWIiOjY1fQ.Uf_R9QoSEJIId-JOqz4UNxI0L1tgcBSBL159UT75nDI",
		},
		{ // has key of Points, not points
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJQb2ludHMiOjE4LCJzdWIiOiJzZWxlbmUifQ.us4MNwC4FTvbmr5ef2piyTUSIL2a0XwZ2tu66jsQDbk",
		},
		{ // points is a string, not a number
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOiIxOCIsInN1YiI6InNlbGVuZSJ9.vfTXyRB6qDI0J7mkso3qxBEIe0RMDsFVz6u97bGC_FE",
		},
		{
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJzdWIiOiJzZWxlbmUifQ.DVKhdVyXfV2cQxHnoNJQdrJUKZ1MuauJdUS8pkcMANE",
			want: userInfo{
				username: "selene",
				points:   18,
			},
			wantOk: true,
		},
	}
	for i, test := range getUserTests {
		var u User
		got, err := u.info(test.jwt)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
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
