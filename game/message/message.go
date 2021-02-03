// Package message contains structures to pass between the ui and server.
package message

import (
	"net"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Type represents what the purpose of a message.
	Type int

	// Message contains information to or from a socket for a game/lobby.
	Message struct {
		// Type is the purpose of the message.
		Type Type `json:"type"`
		// Info is a message to show to the player.
		Info string `json:"info,omitempty"`
		// Game is the info for the current game the player is in.
		Game *game.Info `json:"game,omitempty"`
		// Games contains the information about all the available games.
		Games []game.Info `json:"games,omitempty"`
		// PlayerName is the name of the player the message is to/from.
		PlayerName player.Name `json:"-"`
		// Addr is the socket remote address the message is from
		Addr net.Addr `json:"-"`
		// AddSocketRequest contains info about the socket to add for a player.
		AddSocketRequest *AddSocketRequest `json:"-"`
	}

	// AddSocketRequest is used to add players from http requests.
	AddSocketRequest struct {
		http.ResponseWriter
		*http.Request
		Result chan<- Message
	}
)

const (
	_ Type = iota
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
	// Infos is a MessageType that the server sends to report changes in the games in a lobby.
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
	// AddSocket is used to add a socket for a player.
	AddSocket
)
