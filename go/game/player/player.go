package player

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

type (
	// Player can connect to a Lobby and play in a Game
	Player struct {
		log      *log.Logger
		name     game.PlayerName
		lobby    game.Messenger
		game     game.Messenger // possibly nil
		gameID   game.ID
		socket   game.Messenger
		messages chan game.Message
		deleted  bool
	}

	// Config contains commonly shared player properties
	Config struct {
		Log   *log.Logger
		Lobby game.Messenger
	}
)

// New creates a player from the config and starts it
func (cfg Config) New(name game.PlayerName, conn *websocket.Conn) Player {
	s := socket{
		log:      cfg.Log,
		conn:     conn,
		messages: make(chan game.Message, 16),
	}
	p := Player{
		log:      cfg.Log,
		name:     name,
		lobby:    cfg.Lobby,
		socket:   &s,
		messages: make(chan game.Message, 16),
	}
	s.player = p
	go s.readMessages()
	go s.writeMessages()
	go p.run()
	return p
}

// Handle adds a message to the queue
func (p *Player) Handle(m game.Message) {
	p.messages <- m
}

func (p *Player) run() {
	// TODO: start ticker to periodically get gameTilePositions (like socket idleTicker)
	for m := range p.messages {
		switch m.Type {
		case game.Join:
			p.game = m.Game
			p.gameID = m.GameID
		case game.Leave:
			p.game = nil
			p.socket.Handle(m)
		case game.Delete:
			p.lobby.Handle(game.Message{
				Type:   game.Delete,
				GameID: p.gameID,
				Info:   fmt.Sprintf("%v deleted the game", p.name),
			})
		case game.SocketInfo, game.SocketError, game.Infos, game.ChatSend, game.TilePositions:
			if !p.deleted {
				p.socket.Handle(m)
			}
		case game.StatusChange, game.Snag, game.Swap, game.TilesMoved, game.ChatRecv:
			if p.game == nil {
				p.socket.Handle(game.Message{
					Type: game.SocketError,
					Info: fmt.Sprintf("no game to handle messageType %v", m.Type),
				})
				continue
			}
			m.Player = p
			p.game.Handle(m)
		case game.PlayerDelete:
			if !p.deleted {
				m.Player = p
				p.lobby.Handle(m)
			}
			p.deleted = true
			break
		default:
			p.log.Printf("player %v does not know how to handle messageType %v", p.name, m.Type)
		}
	}
	if p.game != nil {
		p.game.Handle(game.Message{
			Type:   game.PlayerDelete,
			Player: p,
		})
		p.game = nil
	}
	if p.socket != nil {
		// close(p.socketMessages)
		p.socket = nil
	}
	p.log.Printf("player %v closed", p.name)
}
