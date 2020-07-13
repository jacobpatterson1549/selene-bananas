package certificate

import (
	"bytes"
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
			path: "/not" + acmeHeader + "abc",
		},
	}
	for i, test := range isForTests {
		var c Challenge
		got := c.IsFor(test.path)
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
	var w bytes.Buffer
	path := acmeHeader + "abc"
	c.Handle(&w, path)
	want := "abc.s3cr3t"
	got := string(w.Bytes())
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
		var w bytes.Buffer
		if err := c.Handle(&w, path); err == nil {
			t.Errorf("Test %v: expected error when handling url with incorrect challenge token", i)
		}
	}
}
