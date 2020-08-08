package tile

import "errors"

// letter is the value of a tile.
type letter rune

// newLetter creates a letter from the rune.
func newLetter(r rune) (letter, error) {
	if r < 'A' || 'Z' < r {
		return 0, errors.New("letter must be uppercase and between A and Z: " + string(r))
	}
	return letter(r), nil
}

// String returns the letter as a string.
func (l letter) String() string {
	return string(l)
}
