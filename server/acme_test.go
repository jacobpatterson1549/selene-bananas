package server

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestIsForHTTP01Challenge(t *testing.T) {
	isForTests := []struct {
		path string
		want bool
	}{
		{},
		{
			path: acmeHeader,
		},
		{
			path: "abc",
		},
		{
			path: acmeHeader + "abc",
			want: true,
		},
		{
			path: "/.NOTT-known/acme-challenge/abc",
		},
	}
	for i, test := range isForTests {
		var c Challenge
		got := c.isFor(test.path)
		if test.want != got {
			t.Errorf("test %v: wanted %v when path = %v", i, test.want, test.path)
		}
	}
}

func TestHandleChallenge(t *testing.T) {
	c := Challenge{
		Token: "abc",
		Key:   "s3cr3t",
	}
	r := httptest.NewRequest("", acmeHeader+"abc", nil)
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

func TestHandleBadChallengeToken(t *testing.T) {
	c := Challenge{
		Token: "abc",
		Key:   "s3cr3t",
	}
	invalidPaths := []string{
		"/",
		"/acme-challenge/abc",
		acmeHeader + "othertoken",
	}
	for i, path := range invalidPaths {
		r := httptest.NewRequest("", path, nil)
		w := httptest.NewRecorder()

		if err := c.handle(w, r); err == nil {
			t.Errorf("Test %v: expected error when handling url with incorrect challenge token", i)
		}
	}
}
