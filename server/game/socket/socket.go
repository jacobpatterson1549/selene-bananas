// Package socket handles communication with a player using a websocket connection
package socket

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		runner.Runner
		conn   Conn
		active bool
		Config
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
		// PingPeriod is how often ping messages should be sent.  Should be less than ReadWait.
		PingPeriod time.Duration
		// IdlePeroid is the amount of time that can pass between handling messages that are not pings before the connection is idle and will be disconnected
		IdlePeriod time.Duration
		// HTTPPingPeriod is the amount of time between sending requests for the connection to send a http ping on a different socket
		// Heroku servers shut down if 30 minutes passess between HTTP requests
		HTTPPingPeriod time.Duration
	}

	// Conn is the connection than backs the socket
	Conn interface {
		// ReadJSON reads the next json message from the connection.
		ReadJSON(v interface{}) error
		// WriteJSON writes the message as json to the connection.
		WriteJSON(v interface{}) error
		// Close closes the connection.
		Close() error
		// WritePing writes a ping message on the connection.
		WritePing() error
		// WriteClose writes a close message on the connection and always closes it.
		WriteClose(reason string) error
		// IsUnexpectedCloseError determines if the error message is an unexpected close error.
		IsUnexpectedCloseError(err error) bool
		// RemoteAddr gets the remote network address of the connection.
		RemoteAddr() net.Addr
	}
)

var errSocketClosed = fmt.Errorf("socket closed")

// NewSocket creates a socket
func (cfg Config) NewSocket(conn Conn) (*Socket, error) {
	if err := cfg.validate(conn); err != nil {
		return nil, fmt.Errorf("creating socket: validation: %w", err)
	}
	s := Socket{
		conn:   conn,
		Config: cfg,
	}
	return &s, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(conn Conn) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case conn == nil:
		return fmt.Errorf("websocket connection required")
	case cfg.TimeFunc == nil:
		return fmt.Errorf("time func required required")
	case cfg.ReadWait <= 0:
		return fmt.Errorf("positive read wait period required")
	case cfg.WriteWait <= 0:
		return fmt.Errorf("positive write wait period required")
	case cfg.PingPeriod <= 0:
		return fmt.Errorf("positive ping period required")
	case cfg.IdlePeriod <= 0:
		return fmt.Errorf("positive idle period required")
	case cfg.HTTPPingPeriod <= 0:
		return fmt.Errorf("positive http ping period required")
	case cfg.PingPeriod >= cfg.ReadWait:
		return fmt.Errorf("ping period should be less than read wait")
	}
	return nil
}

// Run writes Socket messages to the messages channel and reads incoming messages on separate goroutines.
// The Socket runs until the connection fails for an unexpected reason or the context is cancelled
func (s *Socket) Run(ctx context.Context, in chan<- message.Message, out <-chan message.Message) error {
	if err := s.Runner.Run(); err != nil {
		return fmt.Errorf("running socket: %v", err)
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	go s.readMessages(ctx, in)
	go s.writeMessages(ctx, cancelFunc, out)
	return nil
}

// String implements the fmtStringer interface, uniquely identifying the socket by its address
func (s *Socket) String() string {
	a := s.conn.RemoteAddr()
	return fmt.Sprintf("socket on %v at %v", a.Network(), a.String())
}

// readMessages receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessages(ctx context.Context, messages chan<- message.Message) {
	for { // BLOCKING
		m, err := s.readMessage()
		select {
		case <-ctx.Done():
			s.closeConn("server shut down")
			return
		default:
			if err != nil {
				if err != errSocketClosed {
					reason := fmt.Sprintf("reading socket messages stopped for player %v: %v", s, err)
					s.Log.Print(reason)
					s.closeConn(reason)
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
func (s *Socket) writeMessages(ctx context.Context, cancelFunc context.CancelFunc, messages <-chan message.Message) {
	s.active = false
	pingTicker := time.NewTicker(s.PingPeriod)
	httpPingTicker := time.NewTicker(s.HTTPPingPeriod)
	idleTicker := time.NewTicker(s.IdlePeriod)
	var closeReason string
	defer func() {
		pingTicker.Stop()
		httpPingTicker.Stop()
		idleTicker.Stop()
		cancelFunc()
		s.closeConn(closeReason)
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
			err = s.conn.WritePing()
		case <-httpPingTicker.C:
			err = s.writeMessage(message.Message{
				Type: message.SocketHTTPPing,
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
				closeReason = fmt.Sprintf("writing socket messages stopped for player %v: %v", s, err)
				s.Log.Print(closeReason)
			}
			return
		}
	}
}

// readMessage reads the next message from the connection.
func (s *Socket) readMessage() (*message.Message, error) {
	var m message.Message
	if err := s.conn.ReadJSON(&m); err != nil { // BLOCKING
		if s.conn.IsUnexpectedCloseError(err) {
			return nil, fmt.Errorf("unexpected socket closure: %v", err)
		}
		return nil, errSocketClosed
	}
	if s.Debug {
		s.Log.Printf("socket reading message with type %v", m.Type)
	}
	// m.PlayerName = s.playerName // TODO: ensure socket manager or lobby adds player name to message
	if m.Game == nil {
		return nil, fmt.Errorf("received message not relating to game")
	}
	return &m, nil
}

// writeMessage writes a message to the connection.
func (s *Socket) writeMessage(m message.Message) error {
	if s.Debug {
		s.Log.Printf("socket writing message with type %v", m.Type)
	}
	if err := s.conn.WriteJSON(m); err != nil {
		return fmt.Errorf("writing socket message: %v", err)
	}
	if m.Type == message.PlayerDelete {
		return fmt.Errorf("player deleted")
	}
	return nil
}

// refreshDeadline is called when a wait needs to be refreshed.
func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	now := s.TimeFunc()
	nowTime := time.Unix(now, 0)
	deadline := nowTime.Add(period)
	if err := refreshDeadlineFunc(deadline); err != nil {
		err = fmt.Errorf("error refreshing ping/pong deadline: %w", err)
		s.Log.Print(err)
		return err
	}
	return nil
}

// closeConn closes the websocket connection without reporting any errors.
func (s *Socket) closeConn(reason string) {
	s.conn.WriteClose(reason)
	s.conn.Close()
	s.Runner.Finish()
}
