// +build js,wasm

package socket

import (
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom/json"
)

// parseMessageJSON converts the text into a game message.
func parseMessageJSON(text string) (game.Message, error) {
	return game.Message{}, nil // TODO
}

// messageToJSON converts the value to a JSON string.
func messageToJSON(m game.Message) (string, error) {
	return json.Stringify(m)
}
