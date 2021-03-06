// +build js,wasm

package user

import (
	"encoding/json"
	"reflect"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

func TestGetUser(t *testing.T) {
	global := js.Global()
	atobNodeJS := func(this js.Value, args []js.Value) interface{} {
		encodedData := args[0].String()
		buffer := global.Get("Buffer")
		buf := buffer.Call("from", encodedData, "base64")
		decodedData := buf.Call("toString")
		return decodedData
	}
	atob := js.FuncOf(atobNodeJS)
	global.Set("atob", atob)
	defer atob.Release()
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
		{ // bad json
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJTdWIiOiJzZWxlbmUifg.GN3dIGP0ENeN1SC78ByrW4dmlm2qBP9XVeACAclGhZ8",
		},
		{
			jwt: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwb2ludHMiOjE4LCJzdWIiOiJzZWxlbmUifQ.DVKhdVyXfV2cQxHnoNJQdrJUKZ1MuauJdUS8pkcMANE",
			want: userInfo{
				Name:   "selene",
				Points: 18,
			},
			wantOk: true,
		},
	}
	for i, test := range getUserTests {
		var u User
		got, err := u.info(test.jwt)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.want != *got:
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}

func TestEscapePassword(t *testing.T) {
	init := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \^$*+?.()|[]{} but snowman should be unescaped: ☃`
	want := `ok characters are: ` + "`" + `'"<>%&_:;/, escaped are  \\\^\$\*\+\?\.\(\)\|\[\]\{\} but snowman should be unescaped: ☃`
	var httpClient http.Client
	u := New(nil, httpClient)
	got := u.escapePassword(init)
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestParseUserInfoJSON(t *testing.T) {
	parseUserInfoJSONTests := []struct {
		json   string
		wantOk bool
		want   *userInfo
	}{
		{},
		{
			json: `{"sub":18,"points":18}`, // bad name type
		},
		{
			json: `{"sub":"selene","points":"18"}`, // bad points type
		},
		{
			json:   `{"sub":"selene","points":18}`,
			wantOk: true,
			want: &userInfo{
				Name:   "selene",
				Points: 18,
			},
		},
	}
	for i, test := range parseUserInfoJSONTests {
		var got userInfo
		err := json.Unmarshal([]byte(test.json), &got)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(*test.want, got):
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, *test.want, got)
		}
	}
}
