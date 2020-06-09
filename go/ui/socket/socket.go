// +build js

// Package socket contains the logic to communicate with the server for the game via websocket communication
package socket

import (
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

// OnMessage is called when the websocket opens.
func OnOpen() {
	js.SetChecked("has-websocket", true)
}

// OnMessage is called when the websocket is closing.
func OnClose() {
	js.SetChecked("has-websocket", false)
	js.SetChecked("has-game", false)
	js.SetChecked("tab-4", true) // lobby tab
}

// OnMessage is called when the websocket encounters an unexpected error.
func OnError() {
	log.Error("lobby closed")
	user.Logout()
}

// OnMessage is called when the websocket receives a message.
func OnMessage(m game.Message) {
	js.OnMessage(m)
}
