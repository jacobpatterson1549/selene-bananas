// Package message contains structures to pass between the ui and server.
package message

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"

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
	// CreateGame is a MessageType that users send to open a new game.
	CreateGame
	// JoinGame is a MessageType that users send to join a game or the server sends to have the user load a game.
	JoinGame
	// LeaveGame is a MessageType that users and servers send to indicate that a user can to longer be in the current game.
	LeaveGame
	// DeleteGame is a MessageType that users send to remove a game from the server.
	DeleteGame
	// GameChat is a MessageType that users send to communicate with ether players through the server.
	GameChat
	// RefreshGameBoard refreshes the board size for the current game by reading the NumCols and NumRows fields.
	RefreshGameBoard
	// ChangeGameStatus is a MessageType that users and servers send to request or inform of a game status change.
	ChangeGameStatus
	// ChangeGameTiles is a MessageType that the server sends to users to indicate that tiles have been changed for any player.
	ChangeGameTiles
	// SnagGameTile is a MessageType that users send to the server to request a new tile when they have none left to use.
	SnagGameTile
	// SwapGameTile is a MessageType that users send to the server to exchange a tile for three new ones.
	SwapGameTile
	// MoveGameTile is a MessageType that users send to the server whenever they change the state of their boards.
	MoveGameTile
	// GameInfos is a MessageType that the server sends to report changes in the games in a lobby.
	GameInfos
	// SocketWarning is a MessageType that servers send to inform users that a request is invalid.
	SocketWarning
	// SocketError is a MessageType that servers send to users to report an unexpected state.
	SocketError
	// SocketHTTPPing is a MessageType the server sends to the user to request a http request to the site to keep it active.  Some environments shut down after a period of HTTP inactivity has passed.
	SocketHTTPPing
	// SocketAdd is used to add a socket for a player.
	SocketAdd
	// SocketClose is sent when the socket is closed
	SocketClose
	// PlayerRemove is a MessageType that gets sent from the lobby to inform that all sockets should be removed.
	PlayerRemove
)

// Send is a unility function for sending messages. out on.
// When debugging, it prints a message before and after the message is sent to help identify deadlocks
func Send(m Message, out chan<- Message, debug bool, log *log.Logger) {
	if debug {
		id := rand.Int()
		log.Printf("[id: %v] sending message: %v", id, m)
		defer log.Printf("[id: %v] message sent", id)
	}
	out <- m
}

// String prints the Type as a debug-useful string.
func (t Type) String() string {
	var s string
	switch t {
	case CreateGame:
		s = "CreateGame"
	case JoinGame:
		s = "JoinGame"
	case LeaveGame:
		s = "LeaveGame"
	case DeleteGame:
		s = "DeleteGame"
	case GameChat:
		s = "GameChat"
	case RefreshGameBoard:
		s = "RefreshGameBoard"
	case ChangeGameStatus:
		s = "ChangeGameStatus"
	case ChangeGameTiles:
		s = "ChangeGameTiles"
	case SnagGameTile:
		s = "SnagGameTile"
	case SwapGameTile:
		s = "SwapGameTile"
	case MoveGameTile:
		s = "MoveGameTile"
	case GameInfos:
		s = "GameInfos"
	case SocketWarning:
		s = "SocketWarning"
	case SocketError:
		s = "SocketError"
	case SocketHTTPPing:
		s = "SocketHTTPPing"
	case SocketAdd:
		s = "SocketAdd"
	case SocketClose:
		s = "SocketClose"
	case PlayerRemove:
		s = "PlayerRemove"
	}
	return s + "(" + strconv.Itoa(int(t)) + ")"
}
