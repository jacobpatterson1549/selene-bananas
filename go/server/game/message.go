package game

import (
	"github.com/jacobpatterson1549/selene-bananas/go/server/game/tile"
)

type (
	// MessageType represents what the purpose of a message is
	MessageType int

	// Message contains information to or from a player for a game/lobby
	Message struct {
		Type          MessageType     `json:"type"`
		Info          string          `json:"info,omitempty"`
		Tiles         []tile.Tile     `json:"tiles,omitempty"`
		TilePositions []tile.Position `json:"tilePositions,omitempty"`
		TilesLeft     int             `json:"tilesLeft,omitempty"`
		GameInfos     []Info          `json:"gameInfos,omitempty"`
		GameID        ID              `json:"gameID,omitempty"`
		GameStatus    Status          `json:"gameStatus,omitempty"`
		GamePlayers   []string        `json:"gamePlayers,omitempty"`
		// pointers for inter-goroutine communication:
		PlayerName   PlayerName  `json:"-"`
		Player       Messenger   `json:"-"`
		Game         Messenger   `json:"-"`
		GameInfoChan chan<- Info `json:"-"`
	}

	// Messenger handles messages
	Messenger interface {
		Handle(m Message)
	}
)

// not using iota because MessageTypes are switched on on in javascript
const (
	Create         MessageType = 1
	Join           MessageType = 2
	Leave          MessageType = 3
	Delete         MessageType = 4 // TODO: remove this, add gameState = delete
	StatusChange   MessageType = 5
	Snag           MessageType = 7
	Swap           MessageType = 8
	TilesMoved     MessageType = 9
	TilePositions  MessageType = 10
	Infos          MessageType = 11
	PlayerCreate   MessageType = 12
	PlayerDelete   MessageType = 13
	SocketInfo     MessageType = 14
	SocketError    MessageType = 15
	SocketClosed   MessageType = 16
	SocketHTTPPing MessageType = 17 // the socket should send a http ping every so often  This is  explicitly for heroku, which will shut down the server if 30 minutes passes between http requests
	ChatRecv       MessageType = 18
	ChatSend       MessageType = 19
)
