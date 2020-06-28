// +build js,wasm

package socket

import (
	"net/url"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
)

type (
	mockUser struct {
		jwt string
	}
)

func TestReleaseWebSocketJsFuncs(t *testing.T) {
	var s Socket
	// it should be ok to release the functions multiple times, even if they are undefined/null
	s.releaseWebSocketJsFuncs()
	s.releaseWebSocketJsFuncs()
}

func TestGetWebSocketURL(t *testing.T) {
	getWebSocketURLTests := []struct {
		url  string
		jwt  string
		want string
	}{
		{
			url:  "http://127.0.0.1:8000/user_join_lobby",
			jwt:  "a.jwt.token",
			want: "ws://127.0.0.1:8000/user_join_lobby?access_token=a.jwt.token",
		},
		{
			url:  "https://example.com",
			jwt:  "XYZ",
			want: "wss://example.com?access_token=XYZ",
		},
	}
	for i, test := range getWebSocketURLTests {
		url, err := url.Parse(test.url)
		if err != nil {
			t.Errorf("Test %v: %v", i, err)
			continue
		}
		f := dom.Form{
			URL:    *url,
			Params: make(map[string][]string, 1),
		}
		u := mockUser{
			jwt: test.jwt,
		}
		s := Socket{
			User: &u,
		}
		got := s.getWebSocketURL(f)
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func (u mockUser) JWT() string {
	return u.jwt
}

func (u mockUser) Username() string {
	return ""
}

func (u *mockUser) Logout() {
	// NOOP
}
