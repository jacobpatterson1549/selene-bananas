package game

import (
	"fmt"
	"log"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	player struct {
		log      *log.Logger
		username db.Username
		lobby    *lobby
		game     *game // possibly nil
		socket   *socket
		messages chan message
	}
)

func (p player) run() {
	for m := range p.messages {
		switch m.Type {
		case gameJoin:
			p.game = m.Game
			p.game.messages <- message{Type: gameTilePositions}
		case gameLeave, gameDelete:
			p.game = nil
			p.socket.messages <- m
		case gameStart, gameSnag, gameSwap, gameFinish, gameTilePositions:
			if p.game == nil {
				err := fmt.Errorf("no game to handle messageType %v", m.Type)
				p.socket.messages <- message{Type: socketError, Info: err.Error()}
				return
			}
			p.game.messages <- m
		case playerDelete:
			p.game = nil
			close(p.socket.messages)
		default:
			p.log.Printf("player does not know how to handle messageType %v", m.Type)
		}
	}
}
