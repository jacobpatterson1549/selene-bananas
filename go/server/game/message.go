package game

import (
	"encoding/json"
	"fmt"
)

type (
	// MessageType represents what the purpose of a message is
	MessageType int
	// Message contains information to or from a player for a game/lobby
	Message struct {
		Type    MessageType `json:"type"`
		Message string      `json:"message,omitempty"`
		Tiles   []tile      `json:"-"`
	}
	// jsonMessage is used to marshal and unmarshal messages
	jsonMessage struct {
		*messageAlias
		StringTiles []string `json:"tiles,omitempty"`
	}
	// messageAlias is used to prevent infinite loops in jsonMessage
	messageAlias Message
)

const (
	// not using iota because emssageTypes are switched on on in javascript
	gameCreate       = 1
	gameJoin         = 2
	gameRemove       = 3
	gameStart        = 4
	gameSnag         = 5
	gameSwap         = 6
	gameFinish       = 7
	gameClose        = 8
	userTilesChanged = 9
	userMessage      = 10
)

// MarshalJSON has special handling to marshal the tiles to strings
func (m Message) MarshalJSON() ([]byte, error) {
	stringTiles := make([]string, len(m.Tiles))
	for i, t := range m.Tiles {
		stringTiles[i] = string(rune(t))
	}
	jm := jsonMessage{
		(*messageAlias)(&m),
		stringTiles,
	}
	return json.Marshal(jm)
}

// UnmarshalJSON has special handling to unmarshalling tiles from strings
func (m *Message) UnmarshalJSON(b []byte) error {
	jm := &jsonMessage{messageAlias: (*messageAlias)(m)}
	err := json.Unmarshal(b, &jm)
	if err != nil {
		return err
	}
	if len(jm.StringTiles) > 0 {
		tiles := make([]tile, len(jm.StringTiles))
		for i, s := range jm.StringTiles {
			if len(s) != 1 {
				return fmt.Errorf("invalid tile: %v", s)
			}
			tiles[i] = tile(s[0])
		}
		m.Tiles = tiles
	}
	return nil
}
