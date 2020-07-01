package server

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestIsForHTTP01Challenge(t *testing.T) {
	isForTests := []struct {
		token string
		path  string
		want  bool
	}{
		{},
		{
			token: "abc",
			path:  "abc",
		},
		{
			token: "abc",
			path:  "/.well-known/acme-challenge/abc",
			want:  true,
		},
		{
			token: "abc",
			path:  "/.NOTT-known/acme-challenge/abc",
		},
		{
			token: "AbC",
			path:  "/.well-known/acme-challenge/abc",
		},
	}
	for i, test := range isForTests {
		c := Challenge{
			Token: test.token,
		}
		got := c.isFor(test.path)
		if test.want != got {
			t.Errorf("test %v: wanted %v when token = %v, path = %v", i, test.want, test.token, test.path)
		}
	}
}

func TestHandleChallenge(t *testing.T) {
	c := Challenge{
		Token: "abc",
		Key:   "s3cr3t",
	}
	r := httptest.NewRequest("", "/any-url", nil)
	w := httptest.NewRecorder()

	c.handle(w, r)

	resp := w.Result()
	b, _ := ioutil.ReadAll(resp.Body)
	want := "abc.s3cr3t"
	got := string(b)
	if want != got {
		t.Errorf("different body:\nwanted: %v\ngot:    %v", want, got)
	}
}
