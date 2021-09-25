//go:build js && wasm

package user

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

func TestRequest(t *testing.T) {
	event := func(url string) js.Value {
		return js.ValueOf(map[string]interface{}{
			"target": map[string]interface{}{
				"method": "post",
				"action": url,
			},
		})
	}
	type requestResult struct {
		errorLogged       bool
		warningLogged     bool
		credentialsStored bool
		loggedIn          bool
		loggedOut         bool
	}
	tests := []struct {
		event           js.Value
		hasJWT          bool
		confirmOk       bool
		httpResponse    http.Response
		httpResponseErr error
		want            requestResult
	}{
		{
			event: event("bad_form_url"),
			want: requestResult{
				errorLogged: true,
			},
		},
		{
			event: event("http://unknown_action"),
			want: requestResult{
				errorLogged: true,
			},
		},
		{
			event: event("http://example.com/ping"),
			httpResponse: http.Response{
				Code: 401,
			},
			want: requestResult{
				warningLogged: true,
			},
		},
		{
			event: event("http://example.com/ping"),
			httpResponse: http.Response{
				Code: 403,
			},
			want: requestResult{
				errorLogged: true,
				loggedOut:   true,
			},
		},
		{
			event:           event("http://example.com/ping"),
			httpResponseErr: errors.New("httpResponseErr"),
			want: requestResult{
				errorLogged: true,
			},
		},
		// normal cases:
		{
			event: event("http://example.com/user_create"),
			want: requestResult{
				credentialsStored: true,
				loggedOut:         true,
			},
		},
		{
			event: event("http://example.com/user_update_password"),
			want: requestResult{
				credentialsStored: true,
				loggedOut:         true,
			},
		},
		{
			event: event("http://example.com/user_delete"),
		},
		{
			event:     event("http://example.com/user_delete"),
			confirmOk: true,
			want: requestResult{
				loggedOut: true,
			},
		},
		{
			event:     event("http://example.com/user_delete"),
			hasJWT:    true,
			confirmOk: true,
			want: requestResult{
				loggedOut: true,
			},
		},
		{
			event: event("http://example.com/user_login"),
			httpResponse: http.Response{
				Body: ".login_payload.",
			},
			want: requestResult{
				credentialsStored: true,
				loggedIn:          true,
			},
		},
		{
			event: event("http://example.com/ping"),
		},
	}
	for i, test := range tests {
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
						t.Errorf("Test %v: wanted %v, got %v", i, want, got)
					}
					return "browser.jwt.token"
				},
				SetValueFunc: func(query, value string) {
					if query == ".jwt" {
						if want, got := ".login_payload.", value; want != got {
							t.Errorf("Test %v: wanted %v, got %v", i, want, got)
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
						t.Errorf("Test %v: wanted %v, got %v", i, want, got)
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
			httpClient: mockHTTPRequester{
				DoFunc: func(dom http.DOM, req http.Request) (*http.Response, error) {
					if want, got := "post", req.Method; want != got {
						t.Errorf("test %v: wanted %v, got %v", i, want, got)
					}
					if test.hasJWT {
						if _, ok := req.Headers["Authorization"]; !ok {
							t.Errorf("Test %v: wanted authorization header, got %v", i, req.Headers)
						}
					}
					return &test.httpResponse, test.httpResponseErr
				},
			},
		}
		u.request(test.event)
		if want, got := test.want, result; want != got {
			t.Errorf("Test %v: request results not equal:\nwanted %#v\ngot:   %#v", i, want, got)
		}
	}
}
