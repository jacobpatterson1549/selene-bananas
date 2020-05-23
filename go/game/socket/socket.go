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
		debug          bool
		log            *log.Logger
		conn           *websocket.Conn
		timeFunc       func() int64
		playerName     game.PlayerName
		gameID         game.ID // mutable
		active         bool
		pongPeriod     time.Duration
		pingPeriod     time.Duration
		idlePeriod     time.Duration
		httpPingPeriod time.Duration
	}

	// Config contains commonly shared Socket properties
	Config struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written
		Debug bool
		// Log is used to log errors and other information
		Log *log.Logger
		// TimeFunc is a function which should supply the current time since the unix epoch.
		// Used to set ping/pong deadlines
		TimeFunc func() int64
		// PongPeriod is the amount of time that between messages that can bass before the connection is invalid
		PongPeriod time.Duration
		// PingPeriod is the amount of time between sending ping messages to the connection to keep it active
		// Should be less than pongPeriod
		PingPeriod time.Duration
		// IdlePeroid is the amount of time that can pass between handling messages that are not pings before the connection is idle and will be disconnected
		IdlePeriod time.Duration
		// HTTPPingPeriod is the amount of time between sending requests for the connection to send a http ping on a different socket
		// Heroku servers shut down if 30 minutes passess between HTTP requests
		HTTPPingPeriod time.Duration
	}
)

// NewSocket creates a socket
func (cfg Config) NewSocket(conn *websocket.Conn, playerName game.PlayerName) (*Socket, error) {
	if err := cfg.validate(conn, playerName); err != nil {
		return nil, err
	}
	s := Socket{
		debug:          cfg.Debug,
		log:            cfg.Log,
		conn:           conn,
		timeFunc:       cfg.TimeFunc,
		playerName:     playerName,
		pongPeriod:     cfg.PongPeriod,
		pingPeriod:     cfg.PingPeriod,
		idlePeriod:     cfg.IdlePeriod,
		httpPingPeriod: cfg.HTTPPingPeriod,
	}
	return &s, nil
}

func (cfg Config) validate(conn *websocket.Conn, playerName game.PlayerName) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case conn == nil:
		return fmt.Errorf("websocket connection required")
	case len(playerName) == 0:
		return fmt.Errorf("player name required")
	case cfg.PongPeriod <= 0:
		return fmt.Errorf("positive pong period required")
	case cfg.PingPeriod <= 0:
		return fmt.Errorf("positive ping period required")
	case cfg.IdlePeriod <= 0:
		return fmt.Errorf("positive idle period required")
	case cfg.HTTPPingPeriod <= 0:
		return fmt.Errorf("positive http ping period required")
	case cfg.PingPeriod >= cfg.PongPeriod:
		return fmt.Errorf("ping period must be less than pong period")
	}
	return nil
}

// Run writes Socket messages to the messages channel and reads incoming messages on a separate goroutine
func (s *Socket) Run(done <-chan struct{}, messages chan<- game.Message) chan<- game.Message {
	s.readMessages(done, messages)
	return s.writeMessages(done)
}

// readMessages receives messages from the connected socket and writes the to the messages channel
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel
func (s *Socket) readMessages(done <-chan struct{}, messages chan<- game.Message) {
	go func() {
		defer func() {
			s.conn.Close()
			messages <- game.Message{
				Type:       game.PlayerDelete,
				PlayerName: s.playerName,
			}
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
					s.log.Printf("reading socket messages stopped for %v: %v", s.playerName, err)
					return
				}
			}
			messages <- m
			s.active = true
		}
	}()
}

// writeMessages sends messages added to the messages channel to the connected socket
// messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel
func (s *Socket) writeMessages(done <-chan struct{}) chan<- game.Message {
	pingTicker := time.NewTicker(s.pingPeriod)
	httpPingTicker := time.NewTicker(s.httpPingPeriod)
	idleTicker := time.NewTicker(s.idlePeriod)
	messages := make(chan game.Message)
	go func() {
		defer func() {
			pingTicker.Stop()
			httpPingTicker.Stop()
			idleTicker.Stop()
			close(messages)
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
			case <-idleTicker.C:
				if !s.active {
					s.writeMessage(game.Message{
						Type: game.SocketWarning,
						Info: "closing socket due to inactivity",
					})
					return
				}
				s.active = false
			}
			if err != nil {
				s.log.Printf("writing socket messages stopped for %v: %v", s.playerName, err)
				return
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
			return fmt.Errorf("unexpected socket closure: %v", err)
		}
		return fmt.Errorf("socket closed")
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
		return fmt.Errorf("writing socket message: %v", err)
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
	return s.refreshDeadline(s.conn.SetReadDeadline, s.pongPeriod)
}

func (s *Socket) refreshWriteDeadline() error {
	return s.refreshDeadline(s.conn.SetWriteDeadline, s.pingPeriod)
}

func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	now := s.timeFunc()
	nowTime := time.Unix(now, 0)
	deadline := nowTime.Add(period)
	if err := refreshDeadlineFunc(deadline); err != nil {
		err := fmt.Errorf("error refreshing ping/pong deadline: %w", err)
		s.log.Print(err)
		return err
	}
	return nil
}
