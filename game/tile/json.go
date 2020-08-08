// +build !js !wasm

package tile

import (
	"encoding/json"
	"errors"
)

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
