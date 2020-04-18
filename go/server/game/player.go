package game

import (
	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	player struct {
		user db.User
		conn *websocket.Conn
	}
)

func (p player) readMessages() {
	for {

	}
	// TODO
}

func (p player) writeMessages() {
	// TODO
}
