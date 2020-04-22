package game

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	player struct {
		log        *log.Logger
		username   db.Username
		conn       *websocket.Conn
		game       *game
		lobby      gameLobby
		outMessage chan Message
		// tiles    map[rune]bool
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

func (p player) readMessages() {
	defer p.close()
	p.conn.SetReadLimit(maxReadMessageBytes)
	err := p.refreshReadDeadline()
	if err != nil {
		return
	}
	p.conn.SetPongHandler(func(s string) error {
		p.log.Printf("handling pong for %v: %v", p.username, s)
		return p.refreshReadDeadline()
	})
	for {
		var m Message
		err := p.conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				p.log.Println("unexpected websocket closure", err)
			}
			p.close()
			break
		}
		p.log.Printf("handling messages received from %v: %v", p.username, m)
		handleInMessage(m)
	}
}

func (p player) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()
	defer p.close()
	for {
		var err error
		select {
		case m, ok := <-p.outMessage:
			if !ok {
				return
			}
			p.log.Printf("writing message for %v: %v", p.username, m)
			err = p.conn.WriteJSON(m)
			if err != nil {
				return
			}
		case <-pingTicker.C:
			if err = p.refreshWriteDeadline(); err != nil {
				return
			}
			p.log.Printf("writing ping message for %v", p.username)
			if err = p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (p player) close() {
	p.lobby.RemoveUser(p.username)
	if p.game != nil {
		p.game.Remove(p.username)
	}
	close(p.outMessage)
	p.conn.Close()
}

func (p player) refreshReadDeadline() error {
	return p.refreshDeadline(p.conn.SetReadDeadline, pongPeriod, "read")
}

func (p player) refreshWriteDeadline() error {
	return p.refreshDeadline(p.conn.SetWriteDeadline, pingPeriod, "write")
}

func (p player) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration, name string) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err = fmt.Errorf("refreshing %v deadline: %w", name, err)
		p.log.Println(err)
		return err
	}
	return nil
}

func handleInMessage(m Message) {
	// TODO: notify game/lobby
}

// TODO: move this to channel push in game
func (p player) addTiles(tiles ...tile) {
	// TODO
}
