package game

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

type (
	// WordChecker can be used to check if words are valid.
	WordChecker struct {
		words map[string]struct{}
	}
)

// NewWordChecker consumes the lower case words in the reader to use for checking and creates a new WordChecker.
func NewWordChecker(r io.Reader) *WordChecker {
	words := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	scanner.Split(scanLowerWords)
	for scanner.Scan() {
		rawWord := scanner.Text()
		words[rawWord] = struct{}{}
	}
	wc := WordChecker{
		words: words,
	}
	return &wc
}

// Check determines whether or not the word is valid.
func (wc WordChecker) Check(word string) bool {
	lowerWord := strings.ToLower(word)
	_, ok := wc.words[lowerWord]
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
