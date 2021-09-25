//go:build js && wasm

package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

func TestInitDom(t *testing.T) {
	wantJsFuncNames := []string{
		"logout",
		"request",
		"updateConfirmPattern",
	}
	u := User{
		dom: &mockDOM{
			RegisterFuncsFunc: func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
				if want, got := "user", parentName; want != got {
					t.Errorf("wanted parent name to be %v, got %v", want, got)
				}
				switch len(jsFuncs) {
				case len(wantJsFuncNames):
					for _, jsFuncName := range wantJsFuncNames {
						if _, ok := jsFuncs[jsFuncName]; !ok {
							t.Errorf("wanted jsFunc named %q", jsFuncName)
						}
					}
				default:
					t.Errorf("wanted %v jsFuncs, got %v", len(wantJsFuncNames), len(jsFuncs))
				}
				wg.Done()
			},
			NewJsEventFuncFunc: func(fn func(event js.Value)) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
			},
			NewJsEventFuncAsyncFunc: func(fn func(event js.Value), async bool) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
			},
		},
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	u.InitDom(ctx, &wg)
	wg.Wait()
}

func TestGetUser(t *testing.T) {
	atob := func(a string) []byte {
		b, _ := base64.RawURLEncoding.DecodeString(a)
		return b
	}
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
		jwtSet := false
		dom := mockDOM{
			SetValueFunc: func(query, value string) {
				switch {
				case query != ".jwt":
					t.Errorf("Test %v: only expected .jwt to be set, got %v", i, query)
				case value != test.jwt:
					t.Errorf("Test %v: wanted jwt (%v) to be set, got %v", i, test.jwt, value)
				}
				jwtSet = true
			},
			Base64DecodeFunc: atob,
		}
		u := User{
			dom: &dom,
		}
		got, err := u.setInfo(test.jwt)
		switch {
		case jwtSet != test.wantOk:
			t.Errorf("Test %v: wanted jwt to be set only when test is ok", i)
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
	u := New(nil, nil, httpClient)
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

func TestLogin(t *testing.T) {
	tests := []struct {
		jwt string
		log Log
		dom DOM
	}{
		{
			log: &mockLog{
				ErrorFunc: func(text string) {},
			},
		},
		{
			jwt: ".payload.",
			dom: &mockDOM{
				Base64DecodeFunc: func(a string) []byte {
					if want, got := "payload", a; want != got {
						t.Errorf("unexpected payload: wanted %v, got %v", want, got)
					}
					return []byte(`{"points":42}`)
				},
				QuerySelectorFunc:    func(query string) (v js.Value) { return },
				QuerySelectorAllFunc: func(document js.Value, query string) (all []js.Value) { return },
				SetValueFunc: func(query, value string) {
					switch query {
					case ".jwt":
						if want, got := ".payload.", value; want != got {
							t.Errorf("unexpected set of jwt: wanted %v, got %v", want, got)
						}
					case "input.points":
						if want, got := "42", value; want != got {
							t.Errorf("wanted points to be set to %v, got %v", want, got)
						}
					default:
						t.Errorf("unwanted query: %v", query)
					}
				},
				SetCheckedFunc: func(query string, checked bool) {
					if !checked {
						t.Errorf("wanted %v to be checked", query)
					}
				},
			},
		},
	}
	for _, test := range tests {
		u := User{
			dom: test.dom,
			log: test.log,
		}
		u.login(test.jwt)
		// tests are handled by mock objects or should fail with nil refs
	}
}

func TestLogoutButtonClick(t *testing.T) {
	clearCalled := false
	u := User{
		log: &mockLog{
			ClearFunc: func() {
				clearCalled = true
			},
		},
		dom: &mockDOM{
			QuerySelectorFunc:    func(query string) (v js.Value) { return },
			QuerySelectorAllFunc: func(document js.Value, query string) (all []js.Value) { return },
			SetCheckedFunc:       func(query string, checked bool) {},
		},
		Socket: &mockSocket{
			CloseFunc: func() {},
		},
	}
	var event js.Value
	u.logoutButtonClick(event)
	if !clearCalled {
		t.Error("wanted log to be cleared")
	}
}

func TestLogout(t *testing.T) {
	socketClosed := false
	u := User{
		dom: &mockDOM{
			QuerySelectorFunc:    func(query string) (v js.Value) { return },
			QuerySelectorAllFunc: func(document js.Value, query string) (all []js.Value) { return },
			SetCheckedFunc:       func(query string, checked bool) {},
		},
		Socket: &mockSocket{
			CloseFunc: func() {
				socketClosed = true
			},
		},
	}
	u.Logout()
	if !socketClosed {
		t.Error("wanted socket to be closed")
	}
}

func TestJWT(t *testing.T) {
	want := "the.jwt.token"
	u := User{
		dom: &mockDOM{
			ValueFunc: func(query string) string {
				if want, got := ".jwt", query; want != got {
					t.Errorf("wanted to get value of %v, got %v", want, got)
				}
				return want
			},
		},
	}
	got := u.JWT()
	if want != got {
		t.Errorf("jwt values not equal: wanted %v, got %v", want, got)
	}
}

func TestUsername(t *testing.T) {
	tests := []struct {
		jwt  string
		want string
	}{
		{
			jwt:  "BAD",
			want: "",
		},
		{
			jwt:  ".ok.",
			want: "selene",
		},
	}
	for i, test := range tests {
		u := User{
			dom: &mockDOM{
				ValueFunc: func(query string) string {
					switch query {
					case ".jwt":
						return test.jwt
					default:
						t.Errorf("unwanted query: %v", query)
						return ""
					}
				},
				Base64DecodeFunc: func(a string) []byte {
					if want, got := "ok", a; want != got {
						t.Errorf("watned %v, got %v", want, got)
					}
					return []byte(`{"sub":"selene"}`)
				},
				SetValueFunc: func(query, value string) {},
			},
		}
		got := u.Username()
		if test.want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}

func TestUpdateConfirmPassword(t *testing.T) {
	event := js.ValueOf(map[string]interface{}{
		"target": map[string]interface{}{
			"value": "hex", // the password
			"parentElement": map[string]interface{}{
				"nextElementSibling": map[string]interface{}{
					"lastElementChild": map[string]interface{}{
						"pattern": "should be replaced xxx",
					},
				},
			},
		},
	})
	u := User{
		escapeR: strings.NewReplacer("x", "y"),
	}
	u.updateConfirmPassword(event)
	want := "hey"
	got := event.Get("target").
		Get("parentElement").
		Get("nextElementSibling").
		Get("lastElementChild").
		Get("pattern").
		String()
	if want != got {
		t.Errorf("wanted pattern to be set to %v, got %v", want, got)
	}
}

func TestSetUsernamesReadOnly(t *testing.T) {
	// using setAttribute and removeAttribute because the underlying elements need to be changed, not their js copies
	var removeAttributeCalls, setAttributeCalls [][]string
	removeAttribute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		calls := make([]string, len(args))
		for i, a := range args {
			calls[i] = a.String()
		}
		removeAttributeCalls = append(removeAttributeCalls, calls)
		return nil
	})
	defer removeAttribute.Release()
	setAttribute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		calls := make([]string, len(args))
		for i, a := range args {
			calls[i] = a.String()
		}
		setAttributeCalls = append(setAttributeCalls, calls)
		return nil
	})
	defer setAttribute.Release()
	tests := []struct {
		username                 string
		wantReadOnly             bool
		wantRemoveAttributeCalls [][]string
		wantSetAttributeCalls    [][]string
		wantValues               []string
	}{
		{
			wantRemoveAttributeCalls: [][]string{
				{"readonly"},
				{"readonly"},
			},
			wantValues: []string{"jermaine", "murray"}, // values do not need to be set
		},
		{
			username: "bret",
			wantSetAttributeCalls: [][]string{
				{"readonly", "readonly"},
				{"readonly", "readonly"},
			},
			wantValues: []string{"bret", "bret"},
		},
	}
	for i, test := range tests {
		bodyElement := js.ValueOf("__body__")
		usernameInputs := []js.Value{
			js.ValueOf(map[string]interface{}{
				"value":           "jermaine",
				"removeAttribute": removeAttribute,
				"setAttribute":    setAttribute,
			}),
			js.ValueOf(map[string]interface{}{
				"value":           "murray",
				"readonly":        "readonly",
				"removeAttribute": removeAttribute,
				"setAttribute":    setAttribute,
			}),
		}
		setAttributeCalls = nil
		removeAttributeCalls = nil
		u := User{
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					if want, got := "body", query; want != got {
						t.Errorf("Test %v: wanted %v, got %v", i, want, got)
					}
					return bodyElement
				},
				QuerySelectorAllFunc: func(document js.Value, query string) []js.Value {
					if want, got := bodyElement, document; !want.Equal(got) {
						t.Errorf("Test %v: wanted %v, got %v", i, want, got)
					}
					return usernameInputs
				},
			},
		}
		u.setUsernamesReadOnly(test.username)
		if want, got := test.wantRemoveAttributeCalls, removeAttributeCalls; !reflect.DeepEqual(want, got) {
			t.Errorf("Test %v: removeAttributeCalls not equal:\nwanted: %v\ngot:    %v", i, want, got)
		}
		if want, got := test.wantSetAttributeCalls, setAttributeCalls; !reflect.DeepEqual(want, got) {
			t.Errorf("Test %v: setAttributeCalls not equal:\nwanted: %v\ngot:    %v", i, want, got)
		}
		for j, v := range test.wantValues {
			if want, got := v, usernameInputs[j].Get("value").String(); want != got {
				t.Errorf("Test %v, usernameInput %v: values not equal: wanted %v, got %v", i, j, want, got)
			}
		}
	}
}
