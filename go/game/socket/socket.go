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
		gameID     game.ID // mutable
	}

	// Config contains commonly shared Socket properties
	Config struct {
		Debug bool
		Log   *log.Logger
	}
)

// TODO: put some of these parameters as env arguments
const (
	pongPeriod     = 20 * time.Second
	pingPeriod     = (pongPeriod * 80) / 100 // should be less than pongPeriod
	idlePeriod     = 15 * time.Minute
	httpPingPeriod = 10 * time.Minute // should be less than 30 minutes to keep heroku alive
)

// NewSocket creates a socket
func (cfg Config) NewSocket(conn *websocket.Conn, playerName game.PlayerName) Socket {
	return Socket{
		debug:      cfg.Debug,
		log:        cfg.Log,
		conn:       conn,
		playerName: playerName,
	}
}

// ReadMessages receives messages from the connected socket and writes the to the messages channel
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel
func (s *Socket) ReadMessages(done <-chan struct{}, messages chan<- game.Message, errs chan<- error) {
	go func() {
		defer func() {
			s.conn.Close()
		}()
		s.conn.SetPongHandler(s.refreshReadDeadline)
		var m game.Message
		for {
			err := s.readMessage(&m)
			select {
			case <-done:
				return
			default:
				if err != nil {
					errs <- err
				}
				messages <- m
			}
		}
	}()
}

// WriteMessages sends messages added to the messages channel to the connected socket
// messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel
func (s *Socket) WriteMessages(done <-chan struct{}, errs chan<- error) chan<- game.Message {
	pingTicker := time.NewTicker(pingPeriod)
	httpPingTicker := time.NewTicker(httpPingPeriod)
	messages := make(chan game.Message)
	go func() {
		defer func() {
			pingTicker.Stop()
			httpPingTicker.Stop()
		}()
		var err error
		for {
			select {
			case <-done:
				return
			case m := <-messages:
				err = s.writeMessage(m)
			case <-pingTicker.C:
				err = s.writePing()
			case <-httpPingTicker.C:
				err = s.writeMessage(game.Message{
					Type: game.SocketHTTPPing,
				})
			}
			if err != nil {
				errs <- err
			}
		}
	}()
	return messages
}

func (s *Socket) readMessage(m *game.Message) error {
	err := s.conn.ReadJSON(m)
	if err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			m = &game.Message{
				Type: game.SocketError,
				Info: err.Error(),
			}
			return nil
		}
		if _, ok := err.(*websocket.CloseError); !ok || websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			return fmt.Errorf("unexpected websocket closure: %v", err)
		}
		return fmt.Errorf("closed")
	}
	if s.debug {
		s.log.Printf("socket reading message with type %v", m.Type)
	}
	m.PlayerName = s.playerName
	if m.Type != game.Join {
		m.GameID = s.gameID
	}
	return nil
}

func (s *Socket) writeMessage(m game.Message) error {
	if s.debug {
		s.log.Printf("socket writing message with type %v", m.Type)
	}
	switch m.Type {
	case game.Join:
		s.gameID = m.GameID
	case game.Delete, game.Leave:
		s.gameID = 0
	}
	err := s.conn.WriteJSON(m)
	if err != nil {
		return fmt.Errorf("writing websocket message: %v", err)
	}
	if m.Type == game.PlayerDelete {
		return fmt.Errorf("player deleted")
	}
	return nil
}

func (s *Socket) writePing() error {
	if err := s.refreshWriteDeadline(); err != nil {
		return err
	}
	if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}
	return nil
}

func (s *Socket) refreshReadDeadline(appData string) error {
	return s.refreshDeadline(s.conn.SetReadDeadline, pongPeriod)
}

func (s *Socket) refreshWriteDeadline() error {
	return s.refreshDeadline(s.conn.SetWriteDeadline, pingPeriod)
}

func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	if err := refreshDeadlineFunc(time.Now().Add(period)); err != nil {
		err := fmt.Errorf("error refreshing ping/pong deadline for %v: %w", s.playerName, err)
		s.log.Print(err)
		return err
	}
	return nil
}
