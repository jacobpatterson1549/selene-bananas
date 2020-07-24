package word

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWords(t *testing.T) {
	wordsTests := []struct {
		wordsToRead string
		wantWords   []string
	}{
		{},
		{
			wordsToRead: "   ",
		},
		{
			wordsToRead: "a bad cat",
			wantWords:   []string{"a", "bad", "cat"},
		},
		{
			wordsToRead: "A man, a plan, a canal, panama!",
			wantWords:   []string{"a"},
		},
		{
			wordsToRead: "Abc 'words' they're top-secret not.",
		},
	}
	for i, test := range wordsTests {
		want := make(map[string]struct{}, len(test.wantWords))
		for _, w := range test.wantWords {
			want[w] = struct{}{}
		}
		r := strings.NewReader(test.wordsToRead)
		c := NewChecker(r)
		lm := c.(lowercaseMap)
		got := map[string]struct{}(lm)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, want, got)
		}
	}
}

func BenchmarkAmericanEnglishLarge(b *testing.B) {
	wordsFile := "/usr/share/dict/american-english-large"
	f, err := os.Open(wordsFile)
	if err != nil {
		b.Fatalf("could not open wordsFile: %v", err)
	}
	lm := NewChecker(f).(lowercaseMap)
	want := 114064
	got := len(lm)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		b.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}

func TestCheck(t *testing.T) {
	checkTests := []struct {
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
	c := NewChecker(r)
	for i, test := range checkTests {
		got := c.Check(test.word)
		if test.want != got {
			t.Errorf("Test %v: wanted %v, but got %v for word %v - valid words are %v", i, test.want, got, test.word, c)
		}
	}
}
