package game

import (
	"encoding/json"
	"fmt"
)

type (
	// messageType represents what the purpose of a message is
	messageType int
	// message contains information to or from a player for a game/lobby
	message struct {
		Type  messageType `json:"type"`
		Info  string      `json:"message,omitempty"`
		Tiles []tile      `json:"-"`
	}
	// jsonMessage is used to marshal and unmarshal messages
	jsonMessage struct {
		*messageAlias
		StringTiles []string `json:"tiles,omitempty"`
	}
	// messageAlias is used to prevent infinite loops in jsonMessage
	messageAlias message
)

const (
	// not using iota because messageTypes are switched on on in javascript
	gameCreate       messageType = 1
	gameJoin         messageType = 2
	gameRemove       messageType = 3
	gameStart        messageType = 4
	gameSnag         messageType = 5
	gameSwap         messageType = 6
	gameFinish       messageType = 7
	gameClose        messageType = 8
	userTilesChanged messageType = 9
	userMessage      messageType = 10
	userRemove       messageType = 11
	gameInfos        messageType = 12
)

// MarshalJSON has special handling to marshal the tiles to strings
func (m message) MarshalJSON() ([]byte, error) {
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
func (m *message) UnmarshalJSON(b []byte) error {
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
