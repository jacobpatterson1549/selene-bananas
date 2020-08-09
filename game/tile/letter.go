package tile

import "errors"

// Letter is the value of a tile.
type Letter string

// newLetter creates a letter from the rune.
func newLetter(r rune) (*Letter, error) {
	if r < 'A' || 'Z' < r {
		return nil, errors.New("letter must be uppercase and between A and Z: " + string(r))
	}
	l := Letter(r)
	return &l, nil
}
