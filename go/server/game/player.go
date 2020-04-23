package game

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	player interface {
		sendMessage(m messager)
		username() db.Username
	}
	playerImpl struct {
		log   *log.Logger
		u     db.Username
		conn  *websocket.Conn
		game  game
		lobby Lobby
		send  chan message
	}
)

// TODO: put some of these parameters as env arguments
// derived from gorilla websocket example chat client
const (
	writeWait           = 5 * time.Second
	pongPeriod          = 45 * time.Second
	pingPeriod          = (pongPeriod * 80) / 100 // should be less than pongPeriod
	maxReadMessageBytes = 100
)

func newPlayer(log *log.Logger, l Lobby, u db.Username, conn *websocket.Conn) player {
	p := playerImpl{
		log:   log,
		u:     u,
		conn:  conn,
		lobby: l,
		send:  make(chan message, 16),
	}
	go p.readMessages()
	go p.writeMessages()
	return p
}

func (p playerImpl) username() db.Username {
	return p.u
}

func (p playerImpl) readMessages() {
	defer p.close()
	p.conn.SetReadLimit(maxReadMessageBytes)
	err := p.refreshReadDeadline()
	if err != nil {
		return
	}
	p.conn.SetPongHandler(func(s string) error {
		p.log.Printf("handling pong for %v: %v", p.u, s)
		return p.refreshReadDeadline()
	})
	for {
		var m message
		err := p.conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				p.log.Println("unexpected websocket closure", err)
			}
			p.close()
			break
		}
		p.log.Printf("handling messages received from %v: %v", p.u, m)
		p.handle(m)
	}
}

func (p playerImpl) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()
	defer p.close()
	for {
		var err error
		select {
		case m, ok := <-p.send:
			if !ok {
				return
			}
			p.log.Printf("writing message for %v: %v", p.u, m)
			err = p.conn.WriteJSON(m)
			if err != nil {
				return
			}
		case <-pingTicker.C:
			if err = p.refreshWriteDeadline(); err != nil {
				return
			}
			p.log.Printf("writing ping message for %v", p.u)
			if err = p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (p playerImpl) close() {
	p.lobby.RemoveUser(p.username())
	if p.game != nil {
		m, err := infoMessage{Type: userRemove, Username: p.username()}.message()
		if err != nil {
			p.log.Printf("unexpected error trying to send message to remove player from game: %v", err)
		}
		p.game.handleRequest(m)
	}
	close(p.send)
	p.conn.Close()
}

func (p playerImpl) refreshReadDeadline() error {
	return p.refreshDeadline(p.conn.SetReadDeadline, pongPeriod, "read")
}

func (p playerImpl) refreshWriteDeadline() error {
	return p.refreshDeadline(p.conn.SetWriteDeadline, pingPeriod, "write")
}

func (p playerImpl) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration, name string) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err = fmt.Errorf("refreshing %v deadline: %w", name, err)
		p.log.Println(err)
		return err
	}
	return nil
}

func (p playerImpl) handle(m message) {
	// TODO: notify game/lobby
	switch m.Type {
	case gameInfos:
		p.sendGameInfos()
	default:
		p.sendError(fmt.Sprintf("unknown messageType: %v", m.Type))
	}
}

func (p playerImpl) sendMessage(m messager) {
	message, err := m.message()
	if err != nil {
		p.log.Printf("player message error: %v", err)
		return
	}
	p.send <- message
}

func (p playerImpl) sendError(m string) {
	p.sendMessage(infoMessage{Info: m})
}

func (p playerImpl) sendGameInfos() {
	p.lobby.getGameInfos(p.u)
}
