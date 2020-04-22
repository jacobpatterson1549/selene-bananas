package game

import (
	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	player struct {
		username db.Username
		conn     *websocket.Conn
		game     *Game
		tiles    map[rune]bool
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

func (p player) addTiles(tiles ...rune) {
	// TODO
}
