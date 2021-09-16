package logtest

import (
	"bytes"
	"sync"
	"testing"
)

func TestLoggerPrintf(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		printfTests := []struct {
			format string
			v      []interface{}
			want   string
		}{
			{},
			{
				format: "Hello, %s",
				v:      []interface{}{"Selene"},
				want:   "Hello, Selene",
			},
			{
				format: "%s, do you have $%d to lend me?  I want to buy a %s.",
				v:      []interface{}{"Dad", 500, "car"},
				want:   "Dad, do you have $500 to lend me?  I want to buy a car.",
			},
			{
				format: "no value",
				want:   "no value",
			},
			{
				format: "this string is longer than the initial 64 character default size abcdefghijklmnopqrstuvwxyz abcdefghijklmnopqrstuvwxyz abcdefghijklmnopqrstuvwxyz",
				want:   "this string is longer than the initial 64 character default size abcdefghijklmnopqrstuvwxyz abcdefghijklmnopqrstuvwxyz abcdefghijklmnopqrstuvwxyz",
			},
		}
		for i, test := range printfTests {
			var l Logger
			l.Printf(test.format, test.v...)
			got := l.buf.String()
			if test.want != got {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	})
	t.Run("async race", func(t *testing.T) {
		var l Logger
		n := 10
		var wg sync.WaitGroup
		wg.Add(n)
		logA := func() {
			l.Printf("a")
			wg.Done()
		}
		for i := 0; i < n; i++ {
			go logA()
		}
		wg.Wait()
		if want, got := "aaaaaaaaaa", l.buf.String(); want != got {
			t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
		}
	})
}

func TestLoggerString(t *testing.T) {
	want := "hello"
	var l Logger
	l.buf = *bytes.NewBuffer([]byte(want))
	got := l.String()
	if want != got {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestLoggerEmpty(t *testing.T) {
	emptyTests := []struct {
		contents string
		want     bool
	}{
		{
			want: true,
		},
		{
			contents: "",
			want:     true,
		},
		{
			contents: "here\nis some text!\n\t[and more]",
		},
	}
	for i, test := range emptyTests {
		var l Logger
		l.buf = *bytes.NewBuffer([]byte(test.contents))
		got := l.Empty()
		if test.want != got {
			t.Errorf("Test %v: empty states not equal: wanted: %v, got: %v", i, test.want, got)
		}
	}
}
