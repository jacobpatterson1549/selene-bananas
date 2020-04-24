package game

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type (
	socket struct {
		log      *log.Logger
		conn     *websocket.Conn
		player   *player
		messages chan message
	}
)

// TODO: put some of these parameters as env arguments
// derived from gorilla websocket example chat client
const (
	writeWait           = 5 * time.Second
	pongPeriod          = 20 * time.Second
	pingPeriod          = (pongPeriod * 80) / 100 // should be less than pongPeriod
	maxReadMessageBytes = 100
)

func (s socket) readMessages() {
	defer s.close()
	s.conn.SetReadLimit(maxReadMessageBytes)
	err := s.refreshReadDeadline()
	if err != nil {
		return
	}
	s.conn.SetPongHandler(func(pong string) error {
		s.log.Printf("handling pong for %v: %v", s.player.username, pong)
		return s.refreshReadDeadline()
	})
	for {
		var m message
		err := s.conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.log.Print("unexpected websocket closure", err)
			}
			s.close()
			break
		}
		s.log.Printf("receiving messages from %v: %v", s.player.username, m)
		m.Player = s.player
		switch m.Type {
		case gameCreate, gameJoin, gameLeave, gameDelete, gameInfos, playerDelete:
			s.player.lobby.messages <- m
		case gameStart, gameFinish, gameSnag, gameSwap, gameTileMoved:
			s.player.messages <- m
		default:
			s.log.Printf("player does not know how to handle a messageType of %v", m.Type)
		}
	}
}

func (s socket) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()
	defer s.close()
	for {
		var err error
		select {
		case m, ok := <-s.messages:
			if !ok {
				return
			}
			s.log.Printf("writing message for %v: %v", s.player.username, m)
			err = s.conn.WriteJSON(m)
			if err != nil {
				s.log.Printf("error writing websocket message: %v", err)
				return
			}
		case _, ok := <-pingTicker.C:
			if !ok {
				return
			}
			if err = s.refreshWriteDeadline(); err != nil {
				return
			}
			s.log.Printf("writing ping message for %v", s.player.username)
			if err = s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.log.Printf("error writing websocket ping message: %v", err)
				return
			}
		}
	}
}

func (s socket) close() {
	close(s.player.messages)
	s.conn.Close()
}

func (s socket) refreshReadDeadline() error {
	return s.refreshDeadline(s.conn.SetReadDeadline, pongPeriod, "read")
}

func (s socket) refreshWriteDeadline() error {
	return s.refreshDeadline(s.conn.SetWriteDeadline, pingPeriod, "write")
}

func (s socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration, name string) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err := fmt.Errorf("error refreshing %v deadline: %w", name, err)
		s.log.Print(err)
		return err
	}
	return nil
}
