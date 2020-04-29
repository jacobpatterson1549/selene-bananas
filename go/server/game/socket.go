package game

import (
	"encoding/json"
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
		active   bool
	}
)

// TODO: put some of these parameters as env arguments
// derived from gorilla websocket example chat client
const (
	writeWait      = 5 * time.Second
	pongPeriod     = 20 * time.Second
	pingPeriod     = (pongPeriod * 80) / 100 // should be less than pongPeriod
	idlePeriod     = 5 * time.Minute
	httpPingPeriod = 10 * time.Minute // should be less than 30 minutes to keep heroku alive
)

func (s *socket) readMessages() {
	defer s.close()
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
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				s.messages <- message{Type: socketError, Info: err.Error()}
				continue
			}
			if _, ok := err.(*websocket.CloseError); !ok || websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.log.Print("unexpected websocket closure", err)
			}
			s.close()
			return
		}
		s.log.Printf("receiving messages from %v: %v", s.player.username, m)
		m.Player = s.player
		switch m.Type {
		case gameCreate, gameJoin, gameLeave, gameInfos, playerDelete: // TODO: this a bit of a hack.  It would be nice if the socket only interfaced with the player
			s.player.lobby.messages <- m
		case gameStateChange, gameSnag, gameSwap, gameTileMoved:
			s.player.messages <- m
		case gameDelete:
			if s.player.game != nil {
				m.GameID = s.player.game.id
				s.player.lobby.messages <- m
			}
		default:
			s.log.Printf("player does not know how to handle a messageType of %v", m.Type)
		}
		s.active = true // (mutates the socket)
	}
}

func (s *socket) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()
	idleTicker := time.NewTicker(idlePeriod)
	defer idleTicker.Stop()
	httpPingTicker := time.NewTicker(httpPingPeriod)
	defer httpPingTicker.Stop()
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
			if m.Type == playerDelete {
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
		case _, ok := <-idleTicker.C:
			if !ok {
				return
			}
			if !s.active {
				err := s.conn.WriteJSON(message{Type: socketClosed, Info: "connection closing due to inactivity"})
				if err != nil {
					s.log.Printf("error writing websocket message: %v", err)
				}
				return
			}
			s.active = false
		case _, ok := <-httpPingTicker.C:
			if !ok {
				return
			}
			s.messages <- message{Type: socketHTTPPing}
		}
	}
}

func (s socket) close() {
	s.log.Printf("closing socket connection for %v", s.player.username)
	s.player.messages <- message{Type: playerDelete}
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
