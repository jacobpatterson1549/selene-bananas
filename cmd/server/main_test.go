package main

import (
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/word"
)

// BenchmarkNewChecker loads the words which are expected to be from /usr/share/dict/american-english-large, version 2018.04.16-1.
func BenchmarkEmbeddedWords(b *testing.B) {
	r := strings.NewReader(embeddedWords)
	c := word.NewChecker(r)
	want := 114064
	got := len(*c)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		b.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}
