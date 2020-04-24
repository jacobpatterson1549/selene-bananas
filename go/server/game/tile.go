package game

import (
	"encoding/json"
	"fmt"
)

type (
	// Tile is a piece in the game
	// TODO: make int a tileID type
	tile struct {
		ID int    `json:"id"`
		Ch letter `json:"ch"`
	}

	tilePosition struct {
		Tile tile `json:"tile"`
		X    int  `json:"x"`
		Y    int  `json:"y"`
	}

	// Letter is the value of a tile
	letter rune
)

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
