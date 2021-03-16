// +build !js !wasm

package message

import (
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

// Socket is used by the server lobby to ask the socket runner to change sockets.
type Socket struct {
	Type       Type
	PlayerName player.Name
	Result     chan<- error
	http.ResponseWriter
	*http.Request
}
