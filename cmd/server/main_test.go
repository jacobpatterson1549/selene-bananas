package main

import (
	"strings"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/word"
)

// BenchmarkNewValidator loads the embedded words, which should be the dump of the aspell en_US dictionary, Debian: aspell-en2018.04.16-0-1,Alpine: aspell-en=2020.12.07-r0
func BenchmarkNewWordValidator(b *testing.B) {
	r := strings.NewReader(embeddedWords)
	c := word.NewValidator(r)
	want := 77808
	got := len(*c)
	if want != got {
		note := "NOTE: this might be flaky, but it ensures that a large number of words can be loaded."
		b.Errorf("wanted %v words, got %v\n%v", want, got, note)
	}
}
