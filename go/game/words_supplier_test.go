package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestWords(t *testing.T) {
	wordsTests := []struct {
		wordsToRead string
		want        map[string]bool
	}{
		{},
		{
			wordsToRead: "a bad cat",
			want:        map[string]bool{"a": true, "bad": true, "cat": true},
		},
		{
			wordsToRead: "A man, a plan, a canal, panama!",
			want:        map[string]bool{"a": true},
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
