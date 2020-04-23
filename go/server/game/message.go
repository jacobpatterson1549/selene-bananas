package game

import (
	"encoding/json"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// messageType represents what the purpose of a message is
	messageType int

	// message contains information to or from a player for a game/lobby
	message struct {
		Type    messageType     `json:"type"`
		Content json.RawMessage `json:"body,omitempty"`
	}

	messager interface {
		message() (message, error)
	}

	infoMessage struct {
		Type messageType `json:"-"`
		Info string      `json:"info,omitempty"`
	}

	tilesMessage struct {
		// gameSnag/gameSwap messageType
		Type  messageType `json:"-"`
		Info  string      `json:"info,omitempty"`
		Tiles []tile      `json:"tiles"`
	}

	tilePosition struct {
		Tile tile `json:"tile"`
		X    int  `json:"x"`
		Y    int  `json:"y"`
	}

	// gameTilePositions messageType
	tilePositionsMessage []tilePosition

	gameInfo struct {
		Players   []db.Username `json:"players"`
		CanJoin   bool          `json:"canJoin"`
		CreatedAt string        `json:"createdAt"`
	}

	// gameInfos messageType
	gameInfosMessage []gameInfo

	// userRemoveMessageType
	userRemoveMessage db.Username
)

const (
	// not using iota because messageTypes are switched on on in javascript
	gameCreate        messageType = 1
	gameJoin          messageType = 2
	gameRemove        messageType = 3
	gameStart         messageType = 4
	gameSnag          messageType = 5 // tilesMessage
	gameSwap          messageType = 6 // tilesMessage
	gameFinish        messageType = 7
	gameClose         messageType = 8
	gameTilePositions messageType = 9  // tilePositionsMessage
	gameInfos         messageType = 10 // gameInfoMessage
	userRemove        messageType = 11 // userMessage
)

func (mt messageType) message() (message, error) {
	return message{Type: mt}, nil
}

func (im infoMessage) message() (message, error) {
	content, err := json.Marshal(im.Info)
	return message{
		Type:    im.Type,
		Content: content,
	}, err
}

func (tm tilesMessage) message() (message, error) {
	content, err := json.Marshal(tm)
	return message{
		Type:    tm.Type,
		Content: content,
	}, err
}

func (tpm tilePositionsMessage) message() (message, error) {
	content, err := json.Marshal(tpm)
	return message{
		Type:    gameTilePositions,
		Content: content,
	}, err
}

func (gim gameInfosMessage) message() (message, error) {
	content, err := json.Marshal(gim)
	return message{
		Type:    gameInfos,
		Content: content,
	}, err
}

func (urm userRemoveMessage) message() (message, error) {
	content, err := json.Marshal(urm)
	return message{
		Type:    userRemove,
		Content: content,
	}, err
}
