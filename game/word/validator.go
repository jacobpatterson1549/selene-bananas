// Package word handles checking words in the game.
package word

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
)

// Validator determines if words are valid.
type Validator map[string]struct{}

// NewValidator consumes the lower case words in the reader to use for validating.
func NewValidator(r io.Reader) (*Validator, error) {
	if r == nil {
		return nil, errors.New("reader required to initialize word validator from")
	}
	v := make(Validator)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		rawWord := scanner.Text()
		v[rawWord] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for w := range v {
		for _, r := range w {
			if !unicode.IsLower(r) {
				return nil, errors.New("wanted only lower case words, got " + w)
			}
		}
	}
	return &v, nil
}

// Validate determines whether or not the word is valid.
// Words are converted to lowercase before checking.
func (v Validator) Validate(word string) bool {
	lowerWord := strings.ToLower(word)
	_, ok := v[lowerWord]
	return ok
}
