package game

import (
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// MessageType represents what the purpose of a message is
	MessageType int

	// Message contains information to or from a socket for a game/lobby
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
		PlayerName    PlayerName      `json:"-"`
		GameInfoChan  chan<- Info     `json:"-"`
	}
)

// not using iota because MessageTypes are used in javascript // TODO: use iota+1 now that javascript is gone
const (
	Create         MessageType = 1
	Join           MessageType = 2
	Leave          MessageType = 3
	Delete         MessageType = 4
	StatusChange   MessageType = 5
	Snag           MessageType = 7
	Swap           MessageType = 8
	TilesMoved     MessageType = 9
	Infos          MessageType = 11
	PlayerDelete   MessageType = 13
	SocketError    MessageType = 15
	SocketHTTPPing MessageType = 17
	Chat           MessageType = 18
	GetInfos       MessageType = 20
	SocketWarning  MessageType = 21
	TilesChange    MessageType = 22
)
