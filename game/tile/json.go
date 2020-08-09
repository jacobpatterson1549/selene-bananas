// +build !js !wasm

package tile

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON has special handling to unmarshalling tiles from strings.
func (l *Letter) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	if len(s) != 1 {
		return errors.New("letter longer than 1 character: " + s)
	}
	b0 := s[0]
	r := rune(b0)
	l2, err := newLetter(r)
	if err != nil {
		return err
	}
	*l = *l2
	return nil
}
