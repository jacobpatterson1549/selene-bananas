package game

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWords(t *testing.T) {
	var e struct{}
	wordsTests := []struct {
		wordsToRead string
		want        map[string]struct{}
	}{
		{},
		{
			wordsToRead: "   ",
		},
		{
			wordsToRead: "a bad cat",
			want: map[string]struct{}{
				"a":   e,
				"bad": e,
				"cat": e,
			},
		},
		{
			wordsToRead: "A man, a plan, a canal, panama!",
			want:        map[string]struct{}{"a": e},
		},
		{
			wordsToRead: "Abc 'words' they're top-secret not.",
		},
	}
	for i, test := range wordsTests {
		r := strings.NewReader(test.wordsToRead)
		wc, err := NewWordChecker(r)
		switch {
		case err != nil:
			t.Errorf("unexpected error: %v", err)
		case test.want == nil && len(wc.words) != 0 ||
			(test.want != nil && !reflect.DeepEqual(test.want, wc.words)):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, wc.words)
		}
	}
}

func TestWordsReal(t *testing.T) {
	wordsFile := "/usr/share/dict/american-english-small"
	f, err := os.Open(wordsFile)
	if err != nil {
		t.Skipf("could not open wordsFile %v", err)
	}
	wc, err := NewWordChecker(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 40067
	got := len(wc.words)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		t.Errorf("wanted %v words, got %v\n%v", want, got, note)
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
	wc, err := NewWordChecker(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, test := range checkTests {
		got := wc.Check(test.word)
		if test.want != got {
			t.Errorf("Test %v: wanted %v, but got %v for word %v - valid words are %v", i, test.want, got, test.word, wc.words)
		}
	}
}
