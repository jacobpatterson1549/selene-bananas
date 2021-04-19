package word

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func reader(words string) io.Reader {
	return strings.NewReader(words)
}

func TestNewValidator(t *testing.T) {
	wantWords := func(words ...string) *Validator {
		validator := Validator(make(map[string]struct{}, len(words)))
		for _, w := range words {
			validator[w] = struct{}{}
		}
		return &validator
	}
	newValidatorTests := []struct {
		wantOk bool
		words  io.Reader
		want   *Validator
	}{
		{},
		{
			words: iotest.ErrReader(errors.New("cannot read words")),
		},
		{
			wantOk: true,
			words:  reader("   "),
			want:   wantWords(),
		},
		{
			wantOk: true,
			words:  reader("a bad cat"),
			want:   wantWords("a", "bad", "cat"),
		},
		{
			wantOk: true,
			words:  reader("a bad cat"),
			want:   wantWords("a", "bad", "cat"),
		},
		{
			wantOk: true,
			words:  reader("A man, a plan, a canal, panama!"),
			want:   wantWords("a"),
		},
		{
			wantOk: true,
			words:  reader("Abc 'words' they're top-secret not."),
			want:   wantWords(),
		},
	}
	for i, test := range newValidatorTests {
		got, err := NewValidator(test.words)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestValidate(t *testing.T) {
	validateTests := []struct {
		word string
		want bool
	}{
		{},
		{
			word: "bat",
			want: true,
		},
		{
			word: "BAT",
			want: true,
		},
		{
			word: "BAT ",
		},
		{
			word: "'BAT'",
		},
		{
			word: "care",
		},
	}
	for i, test := range validateTests {
		r := reader("apple bat car")
		validator, err := NewValidator(r)
		if err != nil {
			t.Errorf("Test %v: unwanted error: %v", i, err)
			continue
		}
		got := validator.Validate(test.word)
		if test.want != got {
			t.Errorf("Test %v: wanted %v, but got %v for word %v - valid words are %v", i, test.want, got, test.word, validator)
		}
	}
}
