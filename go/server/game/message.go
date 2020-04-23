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
		Type     messageType     `json:"type"`
		Username db.Username     `json:"-"`
		Content  json.RawMessage `json:"body,omitempty"`
	}

	// messager is used to know what should handle inbound and outbound messages between the lobby, game, players (websockets)
	messager interface {
		message() (message, error)
	}

	infoMessage struct {
		Type     messageType `json:"-"`
		Username db.Username `json:"-"`
		Info     string      `json:"info,omitempty"`
	}

	// gameSnag/gameSwap messageType
	tilesMessage struct {
		Type     messageType `json:"-"`
		Username db.Username `json:"-"`
		Info     string      `json:"info,omitempty"`
		Tiles    []tile      `json:"tiles"`
	}

	tilePosition struct {
		Tile tile `json:"tile"`
		X    int  `json:"x"`
		Y    int  `json:"y"`
	}

	// gameTilePositions messageType
	tilePositionsMessage struct {
		Username      db.Username    `json:"-"`
		TilePositions []tilePosition `json:"tilePositions"`
	}

	gameInfo struct {
		Username  db.Username   `json:"-"`
		Players   []db.Username `json:"players"`
		CanJoin   bool          `json:"canJoin"`
		CreatedAt string        `json:"createdAt"`
	}

	// gameInfos messageType
	gameInfosMessage []gameInfo
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

func (im infoMessage) message() (message, error) {
	content, err := json.Marshal(im.Info)
	return message{
		Type:     im.Type,
		Username: im.Username,
		Content:  content,
	}, err
}

func (tm tilesMessage) message() (message, error) {
	content, err := json.Marshal(tm)
	return message{
		Type:     tm.Type,
		Username: tm.Username,
		Content:  content,
	}, err
}

func (tpm tilePositionsMessage) message() (message, error) {
	content, err := json.Marshal(tpm)
	return message{
		Type:     gameTilePositions,
		Username: tpm.Username,
		Content:  content,
	}, err
}

func (gim gameInfosMessage) message() (message, error) {
	content, err := json.Marshal(gim)
	return message{
		Type:    gameInfos,
		Content: content,
	}, err
}
