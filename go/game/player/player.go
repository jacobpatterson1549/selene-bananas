package player

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

type (
	// Player reads and writes messages to the browsers
	Player struct {
		log      *log.Logger
		conn     *websocket.Conn
		name     game.PlayerName
		lobby    game.MessageHandler
		messages chan game.Message
		active   bool
		close    chan bool
		gameID   game.ID // mutable
	}

	// Config contains commonly shared player properties
	Config struct {
		Log   *log.Logger
		Lobby game.MessageHandler
	}
)

// TODO: put some of these parameters as env arguments
const (
	writeWait      = 5 * time.Second
	pongPeriod     = 20 * time.Second
	pingPeriod     = (pongPeriod * 80) / 100 // should be less than pongPeriod
	idlePeriod     = 15 * time.Minute
	httpPingPeriod = 10 * time.Minute // should be less than 30 minutes to keep heroku alive
)

var _ game.MessageHandler = &Player{}

// Handle adds a message to the queue
func (p *Player) Handle(m game.Message) {
	p.messages <- m
}

// NewPlayer creates a player and runs it
func (cfg Config) NewPlayer(name game.PlayerName, conn *websocket.Conn) Player {
	p := Player{
		log:      cfg.Log,
		conn:     conn,
		name:     name,
		lobby:    cfg.Lobby,
		messages: make(chan game.Message, 16),
	}
	if conn != nil {
		go p.readMessages()
		go p.writeMessages()
		go func() {
			<-p.close
			p.log.Printf("closing socket connection for %v", p.name)
			p.conn.WriteJSON(game.Message{ // ignore possible error
				Type: game.PlayerDelete,
			})
			p.conn.Close()
		}()
	}
	return p
}

func (p *Player) readMessages() {
	defer func() {
		p.close <- true
	}()
	p.conn.SetPongHandler(p.refreshReadDeadline)
	for {
		var m game.Message
		err := p.conn.ReadJSON(&m)
		if err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				p.messages <- game.Message{
					Type: game.SocketError,
					Info: err.Error(),
				}
				continue
			}
			if _, ok := err.(*websocket.CloseError); !ok || websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				p.log.Print("unexpected websocket closure", err)
			}
			return
		}
		m.PlayerName = p.name
		switch m.Type {
		case game.Join:
			p.gameID = m.GameID
		default:
			m.GameID = p.gameID
		}
		p.lobby.Handle(m)
		p.active = true
	}
}

func (p *Player) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	idleTicker := time.NewTicker(idlePeriod)
	httpPingTicker := time.NewTicker(httpPingPeriod)
	defer func() {
		pingTicker.Stop()
		idleTicker.Stop()
		httpPingTicker.Stop()
		p.close <- true
	}()
	for {
		var err error
		select {
		case m, ok := <-p.messages:
			if !ok {
				return
			}
			switch m.Type {
			case game.Join:
				p.gameID = m.GameID
				continue
			case game.Delete, game.Leave:
				p.gameID = 0
			}
			err = p.conn.WriteJSON(m)
			if err != nil {
				p.log.Printf("error writing websocket message: %v", err)
				return
			}
			if m.Type == game.PlayerDelete {
				return
			}
		case _, ok := <-pingTicker.C:
			if !ok {
				return
			}
			if err = p.refreshWriteDeadline(); err != nil {
				return
			}
			if err = p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case _, ok := <-idleTicker.C:
			if !ok {
				return
			}
			if !p.active {
				p.conn.WriteJSON(game.Message{ // ignore error
					Type: game.PlayerDelete,
					Info: "connection closing due to inactivity",
				})
				return
			}
			p.active = false
		case _, ok := <-httpPingTicker.C:
			if !ok {
				return
			}
			p.messages <- game.Message{
				Type: game.SocketHTTPPing,
			}
		}
	}
}

func (p *Player) refreshReadDeadline(appData string) error {
	return p.refreshDeadline(p.conn.SetReadDeadline, pongPeriod)
}

func (p *Player) refreshWriteDeadline() error {
	return p.refreshDeadline(p.conn.SetWriteDeadline, pingPeriod)
}

func (p *Player) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err := fmt.Errorf("error refreshing ping/pong deadline for %v: %w", p.name, err)
		p.log.Print(err)
		return err
	}
	return nil
}
