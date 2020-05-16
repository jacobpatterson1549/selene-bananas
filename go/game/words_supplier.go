package game

import (
	"bufio"
	"io"
)

type (
	// WordsSupplier can be used to retrieve words.
	WordsSupplier struct {
		io.Reader
	}
)

// Words gets distinct, lowercase, words that are separated by spaces or newlines.
func (ws WordsSupplier) Words() map[string]struct{} {
	words := make(map[string]struct{})
	scanner := bufio.NewScanner(ws)
	scanner.Split(scanLowerWords)
	var e struct{}
	for scanner.Scan() {
		rawWord := scanner.Text()
		words[rawWord] = e
	}
	return words
}

// scanLowercaseWords is a bufio.SplitFunc that returns the first only-lowercase word
// derived from bufio.ScanWords, but simplified to only handle ascii
func scanLowerWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start, end := 0, 0
	// Scan until the next all lowercase word is found
	for end < len(data) {
		r := rune(data[end])
		switch {
		case isSpace(r):
			if start < end {
				return end + 1, data[start:end], nil
			}
			end++
			start = end
		case isLower(r):
			end++
		default: // uppercase/symbol
			end++
			// skip until next space found
			for end < len(data) {
				r := rune(data[end])
				end++
				if isSpace(r) {
					start = end
					break
				}
			}
		}
	}
	if atEOF && len(data) > start {
		if end > 0 && !isLower(rune(data[end-1])) {
			return len(data), nil, nil
		}
		// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func isSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ':
		return true
	}
	return false
}
