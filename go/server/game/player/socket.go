package player

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/server/game"
)

type (
	// Socket reads and sends messages for the player
	socket struct {
		log      *log.Logger
		conn     *websocket.Conn
		player   Player
		messages chan game.Message
		active   bool
	}

	player struct {
		name  game.PlayerName
		lobby game.Messenger
		game  game.Messenger // possibly nil
		game.Messenger
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

// Handle adds a message to the queue
func (s *socket) Handle(m game.Message) {
	s.messages <- m
}

func (s *socket) readMessages() {
	defer s.close()
	err := s.refreshReadDeadline()
	if err != nil {
		return
	}
	s.conn.SetPongHandler(func(pong string) error {
		return s.refreshReadDeadline()
	})
	for {
		var m game.Message
		err := s.conn.ReadJSON(&m)
		if err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				s.messages <- game.Message{
					Type: game.SocketError,
					Info: err.Error(),
				}
				continue
			}
			if _, ok := err.(*websocket.CloseError); !ok || websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.log.Print("unexpected websocket closure", err)
			}
			s.close()
			return
		}
		m.PlayerName = s.player.name
		m.Player = &s.player
		switch m.Type {
		case game.Create, game.Join, game.Infos, game.PlayerDelete: // TODO: this a bit of a hack.  It would be nice if the socket only interfaced with the player
			s.player.lobby.Handle(m)
		case game.StatusChange, game.Snag, game.Swap, game.TilesMoved, game.Delete, game.ChatRecv:
			s.player.Handle(m)
		default:
			s.log.Printf("player does not know how to handle a messageType of %v", m.Type)
		}
		s.active = true
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
			err = s.conn.WriteJSON(m)
			if err != nil {
				s.log.Printf("error writing websocket message: %v", err)
				return
			}
			if m.Type == game.PlayerDelete {
				return
			}
		case _, ok := <-pingTicker.C:
			if !ok {
				return
			}
			if err = s.refreshWriteDeadline(); err != nil {
				return
			}
			if err = s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.log.Printf("error writing websocket ping message: %v", err)
				return
			}
		case _, ok := <-idleTicker.C:
			if !ok {
				return
			}
			if !s.active {
				err := s.conn.WriteJSON(game.Message{
					Type: game.SocketClosed,
					Info: "connection closing due to inactivity",
				})
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
			s.messages <- game.Message{
				Type: game.SocketHTTPPing,
			}
		}
	}
}

func (s socket) close() {
	s.log.Printf("closing socket connection for %v", s.player.name)
	s.player.Handle(game.Message{
		Type: game.PlayerDelete,
	})
	s.conn.Close()
}

func (s *socket) refreshReadDeadline() error {
	return s.refreshDeadline(s.conn.SetReadDeadline, pongPeriod)
}

func (s *socket) refreshWriteDeadline() error {
	return s.refreshDeadline(s.conn.SetWriteDeadline, pingPeriod)
}

func (s *socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err := fmt.Errorf("error refreshing ping/pong deadline for %v: %w", s.player.name, err)
		s.log.Print(err)
		return err
	}
	return nil
}
