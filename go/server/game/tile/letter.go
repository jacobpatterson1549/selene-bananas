package tile

import (
	"encoding/json"
	"fmt"
)

// letter is the value of a tile
type letter rune

func newLetter(r rune) (letter, error) {
	if r < 'A' || 'Z' < r {
		return 0, fmt.Errorf("letter must be uppercase and between A and Z")
	}
	return letter(r), nil
}

func (l letter) String() string {
	return string(l)
}

// MarshalJSON has special handling to marshal the letters to strings
func (l letter) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(l))
}

// UnmarshalJSON has special handling to unmarshalling tiles from strings
func (l *letter) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	if len(s) != 1 {
		return fmt.Errorf("invalid letter: %v", s)
	}
	ch := s[0]
	if ch < 'A' || ch > 'Z' {
		return fmt.Errorf("invalid letter: %v, must be [A-Z]", s)
	}
	*l = letter(ch)
	return nil
}
