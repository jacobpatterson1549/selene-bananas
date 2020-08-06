package game

import (
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

type (
	// MessageType represents what the purpose of a message.
	MessageType int

	// Message contains information to or from a socket for a game/lobby.
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
		NumCols       int             `json:"c,omitempty"`
		NumRows       int             `json:"r,omitempty"`
		PlayerName    player.Name     `json:"-"`
	}
)

const (
	_ MessageType = iota
	// Create is a MessageType that users send to open a new game.
	Create
	// Join is a MessageType that users send to join a game or the server sends to have the user load a game.
	Join
	// Leave is a MessageType that servers send to indicate that a user can to longer be in the current game.
	Leave
	// Delete is a MessageType that users send to remove a game from the server.
	Delete
	// StatusChange is a MessageType that users and servers send to request or inform of a game status change.
	StatusChange
	// TilesChange is a MessageType that the server sends to users to indicate that tiles have been changed for any player.
	TilesChange
	// Snag is a MessageType that users send to the server to request a new tile when they have none left to use.
	Snag
	// Swap is a MessageType that users send to the server to exchange a tile for three new ones.
	Swap
	// TilesMoved is a MessageType that users send to the server whenever they change the state of their boards.
	TilesMoved
	// Infos is a MessageType that users/servers send to request/report changes in the games in a lobby.
	Infos
	// PlayerDelete is a MessageType that gets sent to inform the game that a player's account has been deleted.
	PlayerDelete
	// SocketWarning is a MessageType that servers send to inform users that a request is invalid.
	SocketWarning
	// SocketError is a MessageType that servers send to users to report an unexpected state.
	SocketError
	// SocketHTTPPing is a MessageType the server sends to the user to request a http request to the site to keep it active.  Some environments shut down after a period of HTTP inactivity has passed.
	SocketHTTPPing
	// Chat is a MessageType that users send to communicate with ether players through the server.
	Chat
	// BoardSize refreshes the board size for the current game by reading the NumCols and NumRows fields.
	BoardSize
)
