package game

import (
	"fmt"
	"io/ioutil"
	"strings"
)

type (
	// WordsSupplier can be used to retrieve words.
	WordsSupplier interface {
		Words() (map[string]bool, error)
	}

	fileSystemWordsSupplier string
)

const (
	validWordCharacters string = "abcdefghijklmnopqrstuvwxyz"
)

func (f fileSystemWordsSupplier) Words() (map[string]bool, error) {
	wordsFileContents, err := ioutil.ReadFile(string(f))
	if err != nil {
		return nil, fmt.Errorf("reading words from file '%v': %w", f, err)
	}

	rawWords := strings.Fields(string(wordsFileContents))
	words := make(map[string]bool)
	for _, rawWord := range rawWords {
		if hasOnlyLowercaseLetters(rawWord) {
			words[rawWord] = true
		}
	}
	return words, nil
}

func hasOnlyLowercaseLetters(word string) bool {
	for i := 0; i < len(word); i++ {
		if strings.IndexByte(validWordCharacters, word[i]) < 0 {
			return false
		}
	}
	return true
}
