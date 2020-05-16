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
		ws := WordsSupplier{r}
		got := ws.Words()
		if test.want == nil && len(got) != 0 ||
			(test.want != nil && !reflect.DeepEqual(test.want, got)) {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestWordsReal(t *testing.T) {
	wordsFile := "/usr/share/dict/american-english-small"
	f, err := os.Open(wordsFile)
	if err != nil {
		t.Skipf("could not open wordsFile %v", err)
	}
	ws := WordsSupplier{f}
	words := ws.Words()
	want := 40067
	got := len(words)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		t.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}
