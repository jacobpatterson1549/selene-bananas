// Package socket handles communication with a player using a websocket connection
package socket

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		runner.Runner
		Conn
		active bool
		Config
	}

	// Config contains commonly shared Socket properties
	Config struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written
		Debug bool
		// Log is used to log errors and other information
		Log *log.Logger
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
		Conn:   conn,
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

// Run writes messages from the connection to the shared "out" channel.
// Run writes messages recieved from the "in" channel to the connection,
// Run writes Socket messages that are recieved to the outbound channel and reads incoming messages onto the inbound channel on separate goroutines.
// The Socket runs until the connection fails for an unexpected reason or the context is cancelled.
func (s *Socket) Run(ctx context.Context, in <-chan message.Message, out chan<- message.Message) error {
	if err := s.Runner.Run(); err != nil {
		return fmt.Errorf("running socket: %v", err)
	}
	pingTicker := time.NewTicker(s.PingPeriod)
	httpPingTicker := time.NewTicker(s.HTTPPingPeriod)
	idleTicker := time.NewTicker(s.IdlePeriod)
	var wg sync.WaitGroup
	go func() {
		wg.Wait()
		pingTicker.Stop()
		httpPingTicker.Stop()
		idleTicker.Stop()
		s.Runner.Finish()
		s.Conn.Close()
	}()
	wg.Add(1)
	go s.readMessages(ctx, out, &wg)
	wg.Add(1)
	go s.writeMessages(ctx, in, &wg, pingTicker, httpPingTicker, idleTicker)
	return nil
}

// readMessages receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessages(ctx context.Context, in chan<- message.Message, wg *sync.WaitGroup) {
	defer wg.Done()
	for { // BLOCKING
		m, err := s.readMessage()
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				if err != errSocketClosed {
					reason := fmt.Sprintf("reading socket messages stopped for player %v: %v", s, err)
					s.Log.Print(reason)
					s.Conn.WriteClose(reason)
				}
				return
			}
		}
		in <- *m
		s.active = true
	}
}

// writeMessages sends messages from the outbound messages channel to the connected socket.
// Messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel.
// The tickers are used to periodically write messages or check for read activity.
func (s *Socket) writeMessages(ctx context.Context, out <-chan message.Message, wg *sync.WaitGroup,
	pingTicker, httpPingTicker, idleTicker *time.Ticker) {
	s.active = false
	var closeReason string
	defer func() {
		s.Conn.WriteClose(closeReason)
		s.Log.Print(closeReason)
		wg.Done()
	}()
	var err error
	for { // BLOCKING
		select {
		case <-ctx.Done():
			closeReason = "server shutting down"
			return
		case m := <-out:
			err = s.writeMessage(m)
		case <-pingTicker.C:
			err = s.Conn.WritePing()
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
			}
			return
		}
	}
}

// readMessage reads the next message from the connection.
func (s *Socket) readMessage() (*message.Message, error) {
	var m message.Message
	if err := s.Conn.ReadJSON(&m); err != nil { // BLOCKING
		if s.Conn.IsUnexpectedCloseError(err) {
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
	if err := s.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("writing socket message: %v", err)
	}
	if m.Type == message.PlayerDelete {
		return fmt.Errorf("player deleted")
	}
	return nil
}
