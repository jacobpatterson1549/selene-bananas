package game

import (
	"fmt"
	"bufio"
	"io"
	"strings"
	"unicode"
)

type (
	// WordChecker can be used to check if words are valid
	WordChecker struct {
		words map[string]struct{}
	}
)

// NewWordChecker consumes the lower case words in the reader to use for checking and creates a new WordChecker
func NewWordChecker(r io.Reader) (*WordChecker, error) {
	if r == nil {
		return nil, fmt.Errorf("reader required")
	}
	words := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	scanner.Split(scanLowerWords)
	var e struct{}
	for scanner.Scan() {
		rawWord := scanner.Text()
		words[rawWord] = e
	}
	wc := WordChecker{
		words: words,
	}
	return &wc, nil
}

// Check determines whether or not the word is valid
func (wc WordChecker) Check(word string) bool {
	lowerWord := strings.ToLower(word)
	_, ok := wc.words[lowerWord]
	return ok
}

// scanLowercaseWords is a bufio.SplitFunc that returns the first only-lowercase word
// derived from bufio.ScanWords, but simplified to only handle ascii
func scanLowerWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start, end := 0, 0
	// Scan until the next all lowercase word is found
	for end < len(data) {
		r := rune(data[end])
		end++
		switch {
		case unicode.IsSpace(r):
			if start+1 < end {
				return end, data[start:end-1], nil
			}
			start = end
		case !unicode.IsLower(r): // uppercase/symbol
			// skip until next space found
			for end < len(data) {
				r := rune(data[end])
				end++
				if unicode.IsSpace(r) {
					start = end
					break
				}
			}
		}
	}
	if atEOF && len(data) > start {
		if end > 0 && !unicode.IsLower(rune(data[end-1])) {
			return len(data), nil, nil
		}
		// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}
