package word

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	newValidatorTests := []struct {
		words     string
		wantWords []string
	}{
		{},
		{
			words: "   ",
		},
		{
			words:     "a bad cat",
			wantWords: []string{"a", "bad", "cat"},
		},
		{
			words:     "A man, a plan, a canal, panama!",
			wantWords: []string{"a"},
		},
		{
			words: "Abc 'words' they're top-secret not.",
		},
	}
	for i, test := range newValidatorTests {
		want := Validator(make(map[string]struct{}, len(test.wantWords)))
		for _, w := range test.wantWords {
			want[w] = struct{}{}
		}
		r := strings.NewReader(test.words)
		c := NewValidator(r)
		got := *c
		if !reflect.DeepEqual(want, got) {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, want, got)
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
	r := strings.NewReader("apple bat car")
	c := NewValidator(r)
	for i, test := range validateTests {
		got := c.Validate(test.word)
		if test.want != got {
			t.Errorf("Test %v: wanted %v, but got %v for word %v - valid words are %v", i, test.want, got, test.word, c)
		}
	}
}
