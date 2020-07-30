package tile

import (
	"encoding/json"
	"errors"
)

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

// MarshalJSON has special handling to marshal the letters to strings.
func (l letter) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(l))
}

// UnmarshalJSON has special handling to unmarshalling tiles from strings.
func (l *letter) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	if len(s) != 1 {
		return errors.New("invalid letter: " + string(s))
	}
	b0 := s[0]
	r := rune(b0)
	*l, err = newLetter(r)
	return err
}
