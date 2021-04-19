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
	c := make(Validator)
	scanner := bufio.NewScanner(r)
	scanner.Split(scanLowerWords)
	for scanner.Scan() {
		rawWord := scanner.Text()
		c[rawWord] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate determines whether or not the word is valid.
// Words are converted to lowercase before checking.
func (c Validator) Validate(word string) bool {
	lowerWord := strings.ToLower(word)
	_, ok := c[lowerWord]
	return ok
}

// scanLowercaseWords is a bufio.SplitFunc that returns the first only-lowercase word.
// Derived from bufio.ScanWords, but simplified to only handle ASCII.
func scanLowerWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start, end := 0, 0
	skipUntilSpace := false
	// Scan until the next all lowercase word is found
	for end < len(data) {
		r := rune(data[end])
		end++
		switch {
		case unicode.IsSpace(r):
			if !skipUntilSpace && end-start > 1 {
				return end, data[start : end-1], nil
			}
			start = end
			skipUntilSpace = false
		case !unicode.IsLower(r) && !skipUntilSpace: // uppercase/symbol
			skipUntilSpace = true
		}
	}
	if atEOF && len(data) > start {
		if skipUntilSpace {
			return len(data), nil, nil
		}
		// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}
