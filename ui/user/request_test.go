//go:build js && wasm

package user

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/http"
)

func TestRequest(t *testing.T) {
	const (
		// many urls are tested multiple times, so they are normalized here
		exampleUserCreateURL         = "http://example.com/user_create"
		exampleUserUpdatePasswordURL = "http://example.com/user_update_password"
		exampleUserDeleteURL         = "http://example.com/user_delete"
		exampleUserLoginURL          = "http://example.com/user_login"
		examplePingURL               = "http://example.com/ping"
	)
	var requestTests = []struct {
		eventURL              string
		hasJWT                bool
		confirmOk             bool
		httpResponse          http.Response
		httpResponseErr       error
		wantErrorLogged       bool
		wantWarningLogged     bool
		wantCredentialsStored bool
		wantLoggedIn          bool
		wantLoggedOut         bool
	}{
		{
			eventURL:        ("bad_form_url"),
			wantErrorLogged: true,
		},
		{
			eventURL:        ("http://unknown_action"),
			wantErrorLogged: true,
		},
		{
			eventURL: (examplePingURL),
			httpResponse: http.Response{
				Code: 401,
			},
			wantWarningLogged: true,
		},
		{
			eventURL: (examplePingURL),
			httpResponse: http.Response{
				Code: 403,
			},
			wantErrorLogged: true,
			wantLoggedOut:   true,
		},
		{
			eventURL:        examplePingURL,
			httpResponseErr: errors.New("httpResponseErr"),
			wantErrorLogged: true,
		},
		// normal cases:
		{
			eventURL:              exampleUserCreateURL,
			wantCredentialsStored: true,
			wantLoggedOut:         true,
		},
		{
			eventURL:              exampleUserUpdatePasswordURL,
			wantCredentialsStored: true,
			wantLoggedOut:         true,
		},
		{
			eventURL: exampleUserDeleteURL,
		},
		{
			eventURL:      exampleUserDeleteURL,
			confirmOk:     true,
			wantLoggedOut: true,
		},
		{
			eventURL:      exampleUserDeleteURL,
			hasJWT:        true,
			confirmOk:     true,
			wantLoggedOut: true,
		},
		{
			eventURL: exampleUserLoginURL,
			httpResponse: http.Response{
				Body: ".login_payload.",
			},
			wantCredentialsStored: true,
			wantLoggedIn:          true,
		},
		{
			eventURL: examplePingURL,
		},
	}
	for i, test := range requestTests {
		gotErrorLogged := false
		gotWarningLogged := false
		gotCredentialsStored := false
		gotLoggedIn := false
		gotLoggedOut := false
		u := User{
			log: &mockLog{
				ErrorFunc: func(text string) {
					gotErrorLogged = true
				},
				WarningFunc: func(text string) {
					gotWarningLogged = true
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
						gotLoggedIn = true
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
					gotCredentialsStored = true
				},
			},
			Socket: &mockSocket{
				CloseFunc: func() {
					gotLoggedOut = true
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
		switch {
		case test.wantErrorLogged != gotErrorLogged:
			t.Errorf("Test %v: ErrorLogged not equal: wanted %v, got %v", i, test.wantErrorLogged, gotErrorLogged)
		case test.wantWarningLogged != gotWarningLogged:
			t.Errorf("Test %v: WarningLogged not equal: wanted %v, got %v", i, test.wantWarningLogged, gotWarningLogged)
		case test.wantCredentialsStored != gotCredentialsStored:
			t.Errorf("Test %v: CredentialsStored not equal: wanted %v, got %v", i, test.wantCredentialsStored, gotCredentialsStored)
		case test.wantLoggedIn != gotLoggedIn:
			t.Errorf("Test %v: LoggedIn not equal: wanted %v, got %v", i, test.wantLoggedIn, gotLoggedIn)
		case test.wantLoggedOut != gotLoggedOut:
			t.Errorf("Test %v: LoggedOut not equal: wanted %v, got %v", i, test.wantLoggedOut, gotLoggedOut)
		}
	}
}
