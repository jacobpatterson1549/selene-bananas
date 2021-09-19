//go:build !js || !wasm

package message

import (
	"math/rand"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

// Socket is used by the server lobby to ask the socket runner to change sockets.
type Socket struct {
	Type       Type
	PlayerName player.Name
	Result     chan<- error
	http.ResponseWriter
	*http.Request
}

// Send is a utility function for sending messages. out on.
// When debugging, it prints a message before and after the message is sent to help identify deadlocks
func Send(m Message, out chan<- Message, debug bool, log log.Logger) {
	if debug {
		id := rand.Int()
		log.Printf("[id: %v] sending message: %v", id, m)
		defer log.Printf("[id: %v] message sent", id)
	}
	out <- m
}
