package certificate

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestChallengeIsFor(t *testing.T) {
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
			path: "/not" + acmeHeader + "abc",
		},
		{
			path: acmeHeader + "abc",
			want: true,
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

func TestChallengeHandle(t *testing.T) {
	token := "abc"
	key := "s3cr3t"
	want := "abc.s3cr3t"
	handleTests := []struct {
		path     string
		writeErr error
		wantOk   bool
	}{
		{
			path: "/",
		},
		{
			path: "/acme-challenge/" + token,
		},
		{
			path: acmeHeader + "other" + token,
		},
		{
			path:     acmeHeader + token,
			writeErr: fmt.Errorf("write error"),
		},
		{
			path:   acmeHeader + token,
			wantOk: true,
		},
	}
	c := Challenge{
		Token: token,
		Key:   key,
	}
	for i, test := range handleTests {
		var b bytes.Buffer
		w := errorWriter{
			writeErr: test.writeErr,
			Writer:   &b,
		}
		err := c.Handle(&w, test.path)
		got := b.String()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		case want != got:
			t.Errorf("different body:\nwanted: %v\ngot:    %v", want, got)
		}
	}
}

type errorWriter struct {
	writeErr error
	io.Writer
}

func (w errorWriter) Write(p []byte) (n int, err error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.Writer.Write(p)
}
