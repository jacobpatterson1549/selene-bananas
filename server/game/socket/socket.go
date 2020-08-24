// Package socket handles communication with a player using a websocket connection
package socket

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		debug          bool
		log            *log.Logger
		conn           *websocket.Conn
		timeFunc       func() int64
		playerName     player.Name
		gameID         game.ID // mutable
		active         bool
		readWait       time.Duration
		writeWait      time.Duration
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
		// ReadWait is the amout of time that can pass between receiving client messages before timing out.
		ReadWait time.Duration
		// WriteWait is the amout of time that the socket can take to write a message.
		WriteWait time.Duration
		// IdlePeroid is the amount of time that can pass between handling messages that are not pings before the connection is idle and will be disconnected
		IdlePeriod time.Duration
		// HTTPPingPeriod is the amount of time between sending requests for the connection to send a http ping on a different socket
		// Heroku servers shut down if 30 minutes passess between HTTP requests
		HTTPPingPeriod time.Duration
	}
)

var errSocketClosed = fmt.Errorf("socket closed")

// NewSocket creates a socket
func (cfg Config) NewSocket(conn *websocket.Conn, playerName player.Name) (*Socket, error) {
	if err := cfg.validate(conn, playerName); err != nil {
		return nil, fmt.Errorf("creating socket: validation: %w", err)
	}
	pingPeriod := cfg.ReadWait * 9 / 10
	s := Socket{
		debug:          cfg.Debug,
		log:            cfg.Log,
		conn:           conn,
		timeFunc:       cfg.TimeFunc,
		playerName:     playerName,
		readWait:       cfg.ReadWait,
		writeWait:      cfg.WriteWait,
		pingPeriod:     pingPeriod,
		idlePeriod:     cfg.IdlePeriod,
		httpPingPeriod: cfg.HTTPPingPeriod,
	}
	return &s, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(conn *websocket.Conn, playerName player.Name) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case conn == nil:
		return fmt.Errorf("websocket connection required")
	case cfg.TimeFunc == nil:
		return fmt.Errorf("time func required required")
	case len(playerName) == 0:
		return fmt.Errorf("player name required")
	case cfg.ReadWait <= 0:
		return fmt.Errorf("positive read wait period required")
	case cfg.WriteWait <= 0:
		return fmt.Errorf("positive write wait period required")
	case cfg.IdlePeriod <= 0:
		return fmt.Errorf("positive idle period required")
	case cfg.HTTPPingPeriod <= 0:
		return fmt.Errorf("positive http ping period required")
	}
	return nil
}

// Run writes Socket messages to the messages channel and reads incoming messages on separate goroutines.
// The Socket runs until the connection fails for an unexpected reason or a message is received on the "done"< channel.
// Messages the socket receives are sent to the provided channel.
// Messages the socket sends are consumed from the returned channel.
func (s *Socket) Run(ctx context.Context, removeSocketFunc context.CancelFunc, readMessages chan<- game.Message, writeMessages <-chan game.Message) {
	readCtx, readCancelFunc := context.WithCancel(ctx)
	writeCtx, writeCancelFunc := context.WithCancel(ctx)
	go s.readMessages(readCtx, removeSocketFunc, writeCancelFunc, readMessages)
	s.writeMessages(writeCtx, readCancelFunc, writeMessages)
}

// readMessages receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessages(ctx context.Context, removeSocketFunc, writeCancelFunc context.CancelFunc, messages chan<- game.Message) {
	defer func() {
		removeSocketFunc()
		writeCancelFunc()
		s.conn.Close()
	}()
	for { // BLOCKING
		m, err := s.readMessage()
		select {
		case <-ctx.Done():
			CloseConn(s.conn, s.log, "server shut down")
			return
		default:
			if err != nil {
				if err != errSocketClosed {
					reason := fmt.Sprintf("reading socket messages stopped for %v: %v", s.playerName, err)
					s.log.Print(reason)
					CloseConn(s.conn, s.log, reason)
				}
				return
			}
		}
		messages <- *m
		s.active = true
	}
}

// writeMessages sends messages added to the messages channel to the connected socket.
// messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) writeMessages(ctx context.Context, readCancelFunc context.CancelFunc, messages <-chan game.Message) {
	pingTicker := time.NewTicker(s.pingPeriod)
	httpPingTicker := time.NewTicker(s.httpPingPeriod)
	idleTicker := time.NewTicker(s.idlePeriod)
	var closeReason string
	defer func() {
		pingTicker.Stop()
		httpPingTicker.Stop()
		idleTicker.Stop()
		readCancelFunc()
		CloseConn(s.conn, s.log, closeReason)
	}()
	var err error
	for { // BLOCKING
		select {
		case <-ctx.Done():
			closeReason = "server shutting down"
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
				closeReason = "closing socket due to inactivity"
				return
			}
			s.active = false
		}
		if err != nil {
			if err != errSocketClosed {
				closeReason = fmt.Sprintf("writing socket messages stopped for %v: %v", s.playerName, err)
				s.log.Print(closeReason)
			}
			return
		}
	}
}

// readMessage reads the next message from the connection.
func (s *Socket) readMessage() (*game.Message, error) {
	var m game.Message
	if err := s.conn.ReadJSON(&m); err != nil { // BLOCKING
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			return nil, fmt.Errorf("unexpected socket closure: %v", err)
		}
		return nil, errSocketClosed
	}
	if s.debug {
		s.log.Printf("socket reading message with type %v", m.Type)
	}
	m.PlayerName = s.playerName
	if m.Type != game.Join {
		m.GameID = s.gameID
	}
	return &m, nil
}

// writeMessage writes a message to the connection.
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
	if err := s.conn.WriteJSON(m); err != nil {
		return fmt.Errorf("writing socket message: %v", err)
	}
	if m.Type == game.PlayerDelete {
		return fmt.Errorf("player deleted")
	}
	return nil
}

// writePing writes a ping message to the connection.
func (s *Socket) writePing() error {
	if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}
	return nil
}

// refreshDeadline is called when a wait needs to be refreshed.
func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	now := s.timeFunc()
	nowTime := time.Unix(now, 0)
	deadline := nowTime.Add(period)
	if err := refreshDeadlineFunc(deadline); err != nil {
		err = fmt.Errorf("error refreshing ping/pong deadline: %w", err)
		s.log.Print(err)
		return err
	}
	return nil
}

// CloseConn closes the websocket connection without reporting any errors.
func CloseConn(conn *websocket.Conn, log *log.Logger, reason string) {
	data := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	if err := conn.WriteMessage(websocket.CloseMessage, data); err != nil {
		log.Printf("closing connection: writing close message: %v", err)
	}
	conn.Close()
}
