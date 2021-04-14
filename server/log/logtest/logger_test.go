package logtest

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewLogger(t *testing.T) {
	l := NewLogger()
	switch {
	case l == nil:
		t.Errorf("wanted non-nil Logger")
	case l.buf == nil:
		t.Errorf("wanted non-nil internal buffer")
	}
}

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
		}
		for i, test := range printfTests {
			var buf bytes.Buffer
			var l Logger
			l.buf = &buf
			l.Printf(test.format, test.v...)
			got := buf.String()
			if test.want != got {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	})
	t.Run("async race", func(t *testing.T) {
		var buf bytes.Buffer
		var l Logger
		l.buf = &buf
		n := 10
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				l.Printf("a")
				wg.Done()
			}()
		}
		wg.Wait()
		if want, got := "aaaaaaaaaa", buf.String(); want != got {
			t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
		}
	})
}

func TestLoggerString(t *testing.T) {
	want := "hello"
	buf := bytes.NewBuffer([]byte(want))
	var l Logger
	l.buf = buf
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
		buf := bytes.NewBuffer([]byte(test.contents))
		var l Logger
		l.buf = buf
		got := l.Empty()
		if test.want != got {
			t.Errorf("Test %v: empty states not equal: wanted: %v, got: %v", i, test.want, got)
		}
	}
}

func TestLoggerReset(t *testing.T) {
	contents := []string{
		"",
		"stuff",
		"1. there\n2. may be\n3. a TOOOOOOOOOOOOOOOOOON\n4. of stuff",
	}
	for i, data := range contents {
		buf := bytes.NewBuffer([]byte(data))
		var l Logger
		l.buf = buf
		l.Reset()
		switch {
		case !l.Empty():
			t.Errorf("Test %v: wanted Logger to be empty after reset", i)
		case l.String() != "":
			t.Errorf("Test %v: wanted Logger string to be empty after reset, got %v", i, l.String())
		}
	}
}
