package game

import (
	"bufio"
	"io"
	"strings"
)

type (
	// WordsSupplier can be used to retrieve words.
	WordsSupplier struct {
		io.Reader
	}
)

const (
	validWordCharacters string = "abcdefghijklmnopqrstuvwxyz"
)

// Words gets distinct, lowercase, words that are separated by spaces or newlines.
func (ws WordsSupplier) Words() map[string]bool {
	words := make(map[string]bool)
	scanner := bufio.NewScanner(ws)
	scanner.Split(bufio.ScanWords) // TODO: split over lowercase characters
	for scanner.Scan() {
		rawWord := scanner.Text()
		if hasOnlyLowercaseLetters(rawWord) {
			words[rawWord] = true
		}
	}
	return words
}

func hasOnlyLowercaseLetters(word string) bool {
	for i := 0; i < len(word); i++ {
		if strings.IndexByte(validWordCharacters, word[i]) < 0 {
			return false
		}
	}
	return true
}
