package logtest

import (
	"bytes"
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
		l := Logger{&buf}
		l.Printf(test.format, test.v...)
		got := buf.String()
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestLoggerString(t *testing.T) {
	want := "hello"
	buf := bytes.NewBuffer([]byte(want))
	l := Logger{buf}
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
		l := Logger{buf}
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
		l := Logger{buf}
		l.Reset()
		switch {
		case !l.Empty():
			t.Errorf("Test %v: wanted Logger to be empty after reset", i)
		case l.String() != "":
			t.Errorf("Test %v: wanted Logger string to be empty after reset, got %v", i, l.String())
		}
	}
}
