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
		deleted  bool
	}
)

func (p *player) run() {
	// TODO: start ticker to periodically get gameTilePositions (like socket idleTicker)
	for m := range p.messages {
		switch m.Type {
		case gameJoin:
			p.game = m.Game
		case gameLeave:
			p.game = nil
			p.socket.messages <- m
		case gameDelete:
			p.lobby.messages <- message{
				Type:   gameDelete,
				GameID: p.game.id,
				Info:   fmt.Sprintf("%v deleted the game", p.username),
			}
		case socketInfo, socketError, gameInfos:
			p.socket.messages <- m
		case gameStateChange, gameSnag, gameSwap, gameTileMoved, gameTilePositions:
			if p.game == nil {
				p.socket.messages <- message{
					Type: socketError,
					Info: fmt.Sprintf("no game to handle messageType %v", m.Type),
				}
				continue
			}
			m.Player = p
			p.game.messages <- m
		case playerDelete:
			if !p.deleted {
				m.Player = p
				p.lobby.messages <- m
			}
			p.deleted = true
			break
		default:
			p.log.Printf("player %v does not know how to handle messageType %v", p.username, m.Type)
		}
	}
	if p.game != nil {
		p.game.messages <- message{Type: playerDelete, Player: p}
		p.game = nil
	}
	if p.socket != nil {
		close(p.socket.messages)
		p.socket = nil
	}
	p.log.Printf("player %v closed", p.username)
}
