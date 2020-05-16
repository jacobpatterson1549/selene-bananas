package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		debug      bool
		log        *log.Logger
		conn       *websocket.Conn
		playerName game.PlayerName
		lobby      game.MessageHandler
		messages   chan game.Message
		close      chan bool
		gameID     game.ID // mutable
	}

	// Config contains commonly shared Socket properties
	Config struct {
		Debug bool
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

var _ game.MessageHandler = &Socket{}

// Handle adds a message to the queue
func (s *Socket) Handle(m game.Message) {
	s.messages <- m
}

// NewSocket creates a socket and runs it
func (cfg Config) NewSocket(playerName game.PlayerName, conn *websocket.Conn) Socket {
	s := Socket{
		debug:      cfg.Debug,
		log:        cfg.Log,
		conn:       conn,
		playerName: playerName,
		lobby:      cfg.Lobby,
		messages:   make(chan game.Message, 16),
	}
	if conn != nil {
		go s.readMessages()
		go s.writeMessages()
		go func() {
			<-s.close
			s.log.Printf("closing socket connection for %v", s.playerName)
			s.conn.WriteJSON(game.Message{ // ignore possible error
				Type: game.PlayerDelete,
			})
			s.conn.Close()
		}()
	}
	return s
}

func (s *Socket) readMessages() {
	defer func() {
		s.close <- true
	}()
	s.conn.SetPongHandler(s.refreshReadDeadline)
	for {
		var m game.Message
		err := s.conn.ReadJSON(&m)
		if err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				s.Handle(game.Message{
					Type: game.SocketError,
					Info: err.Error(),
				})
				continue
			}
			if _, ok := err.(*websocket.CloseError); !ok || websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.log.Print("unexpected websocket closure", err)
			}
			return
		}
		if s.debug {
			s.log.Printf("socket reading message with type %v", m.Type)
		}
		m.PlayerName = s.playerName
		switch m.Type {
		case game.Join:
			s.gameID = m.GameID
		default:
			m.GameID = s.gameID
		}
		s.lobby.Handle(m)
	}
}

func (s *Socket) writeMessages() {
	pingTicker := time.NewTicker(pingPeriod)
	httpPingTicker := time.NewTicker(httpPingPeriod)
	defer func() {
		pingTicker.Stop()
		httpPingTicker.Stop()
		s.close <- true
	}()
	for {
		var err error
		select {
		case m := <-s.messages:
			if s.debug {
				s.log.Printf("socket writing message with type %v", m.Type)
			}
			switch m.Type {
			case game.Join:
				s.gameID = m.GameID
				continue
			case game.Delete, game.Leave:
				s.gameID = 0
			}
			err = s.conn.WriteJSON(m)
			if err != nil {
				s.log.Printf("error writing websocket message: %v", err)
				return
			}
			if m.Type == game.PlayerDelete {
				return
			}
		case <-pingTicker.C:
			if err = s.refreshWriteDeadline(); err != nil {
				return
			}
			if err = s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-httpPingTicker.C:
			s.Handle(game.Message{
				Type: game.SocketHTTPPing,
			})
		}
	}
}

func (s *Socket) refreshReadDeadline(appData string) error {
	return s.refreshDeadline(s.conn.SetReadDeadline, pongPeriod)
}

func (s *Socket) refreshWriteDeadline() error {
	return s.refreshDeadline(s.conn.SetWriteDeadline, pingPeriod)
}

func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	err := refreshDeadlineFunc(time.Now().Add(period))
	if err != nil {
		err := fmt.Errorf("error refreshing ping/pong deadline for %v: %w", s.playerName, err)
		s.log.Print(err)
		return err
	}
	return nil
}
