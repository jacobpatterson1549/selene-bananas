package tile

import (
	"encoding/json"
	"errors"
)

// MarshalJSON implements the encoding/json.Marshaler interface to marshal letters into strings.
func (l Letter) MarshalJSON() ([]byte, error) {
	letterString := string(l)
	return json.Marshal(letterString)
}

// UnmarshalJSON implements the encoding/json.UnMarshaler interface to unmarshal letters from strings.
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
