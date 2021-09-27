//go:build js && wasm

package user

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

func TestRequest(t *testing.T) {
	type requestResult struct {
		errorLogged       bool
		warningLogged     bool
		credentialsStored bool
		loggedIn          bool
		loggedOut         bool
	}
	const (
		// many urls are tested multiple times, so they are normalized here
		exampleUserCreateURL         = "http://example.com/user_create"
		exampleUserUpdatePasswordURL = "http://example.com/user_update_password"
		exampleUserDeleteURL         = "http://example.com/user_delete"
		exampleUserLoginURL          = "http://example.com/user_login"
		examplePingURL               = "http://example.com/ping"
	)
	var requestTests = []struct {
		eventURL        string
		hasJWT          bool
		confirmOk       bool
		httpResponse    http.Response
		httpResponseErr error
		want            requestResult
	}{
		{
			eventURL: ("bad_form_url"),
			want: requestResult{
				errorLogged: true,
			},
		},
		{
			eventURL: ("http://unknown_action"),
			want: requestResult{
				errorLogged: true,
			},
		},
		{
			eventURL: (examplePingURL),
			httpResponse: http.Response{
				Code: 401,
			},
			want: requestResult{
				warningLogged: true,
			},
		},
		{
			eventURL: (examplePingURL),
			httpResponse: http.Response{
				Code: 403,
			},
			want: requestResult{
				errorLogged: true,
				loggedOut:   true,
			},
		},
		{
			eventURL:        examplePingURL,
			httpResponseErr: errors.New("httpResponseErr"),
			want: requestResult{
				errorLogged: true,
			},
		},
		// normal cases:
		{
			eventURL: exampleUserCreateURL,
			want: requestResult{
				credentialsStored: true,
				loggedOut:         true,
			},
		},
		{
			eventURL: exampleUserUpdatePasswordURL,
			want: requestResult{
				credentialsStored: true,
				loggedOut:         true,
			},
		},
		{
			eventURL: exampleUserDeleteURL,
		},
		{
			eventURL:  exampleUserDeleteURL,
			confirmOk: true,
			want: requestResult{
				loggedOut: true,
			},
		},
		{
			eventURL:  exampleUserDeleteURL,
			hasJWT:    true,
			confirmOk: true,
			want: requestResult{
				loggedOut: true,
			},
		},
		{
			eventURL: exampleUserLoginURL,
			httpResponse: http.Response{
				Body: ".login_payload.",
			},
			want: requestResult{
				credentialsStored: true,
				loggedIn:          true,
			},
		},
		{
			eventURL: examplePingURL,
		},
	}
	for i, test := range requestTests {
		var result requestResult
		u := User{
			log: &mockLog{
				ErrorFunc: func(text string) {
					result.errorLogged = true
				},
				WarningFunc: func(text string) {
					result.warningLogged = true
				},
			},
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) (v js.Value) {
					return // setUsernamesReadOnly
				},
				QuerySelectorAllFunc: func(document js.Value, query string) (all []js.Value) {
					return // no inputs on form, setUsernamesReadOnly
				},
				ValueFunc: func(query string) string {
					if want, got := ".jwt", query; want != got {
						t.Errorf("Test %v: queries not equal: wanted %v, got %v", i, want, got)
					}
					return "browser.jwt.token"
				},
				SetValueFunc: func(query, value string) {
					if query == ".jwt" {
						if want, got := ".login_payload.", value; want != got {
							t.Errorf("Test %v: values set not equal: wanted %v, got %v", i, want, got)
						}
						result.loggedIn = true
					}
				},
				CheckedFunc: func(query string) bool {
					if query != "#has-login" {
						t.Errorf("Test %v: unwanted call checked: %v", i, query)
					}
					return test.hasJWT
				},
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
				ConfirmFunc: func(message string) bool {
					return test.confirmOk
				},
				Base64DecodeFunc: func(a string) []byte {
					if want, got := "login_payload", a; want != got {
						t.Errorf("Test %v: encoded strings not equal: wanted %v, got %v", i, want, got)
					}
					return []byte(`{}`)
				},
				StoreCredentialsFunc: func(form js.Value) {
					result.credentialsStored = true
				},
			},
			Socket: &mockSocket{
				CloseFunc: func() {
					result.loggedOut = true
				},
			},
			httpClient: &mockHTTPRequester{
				DoFunc: func(dom http.DOM, req http.Request) (*http.Response, error) {
					if want, got := "post", req.Method; want != got {
						t.Errorf("Test %v: methods not equal: wanted %v, got %v", i, want, got)
					}
					if _, ok := req.Headers["Authorization"]; test.hasJWT && !ok {
						t.Errorf("Test %v: wanted authorization header, got %v", i, req.Headers)
					}
					return &test.httpResponse, test.httpResponseErr
				},
			},
		}
		event := js.ValueOf(map[string]interface{}{
			"target": map[string]interface{}{
				"method": "post",
				"action": test.eventURL,
			},
		})
		u.request(event)
		if want, got := test.want, result; want != got {
			t.Errorf("Test %v: request results not equal:\nwanted %#v\ngot:   %#v", i, want, got)
		}
	}
}
