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
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		runner.Runner
		Conn
		readActive bool
		Config
		PlayerName player.Name
		net.Addr
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
		// ActivityPeroid is the amount of time to check for read activity.
		// Also sends a HTTP ping on a different socket, as Heroku servers shut down if 30 minutes passess between HTTP requests.
		ActivityCheckPeriod time.Duration
	}

	// Conn is the connection than backs the socket
	Conn interface {
		// ReadJSON reads the next json message from the connection.
		ReadJSON(m *message.Message) error
		// WriteJSON writes the message as json to the connection.
		WriteJSON(m message.Message) error
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
func (cfg Config) NewSocket(pn player.Name, conn Conn) (*Socket, error) {
	a, err := cfg.validate(pn, conn)
	if err != nil {
		return nil, fmt.Errorf("creating socket: validation: %w", err)
	}
	s := Socket{
		Conn:       conn,
		Config:     cfg,
		PlayerName: pn,
		Addr:       a,
	}
	return &s, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(pn player.Name, conn Conn) (net.Addr, error) {
	switch {
	case len(pn) == 0:
		return nil, fmt.Errorf("player name rquired")
	case conn == nil:
		return nil, fmt.Errorf("websocket connection required")
	}
	a := conn.RemoteAddr()
	switch {
	case a == nil:
		return nil, fmt.Errorf("remote address of connection required")
	case cfg.Log == nil:
		return nil, fmt.Errorf("log required")
	case cfg.ReadWait <= 0:
		return nil, fmt.Errorf("positive read wait period required")
	case cfg.WriteWait <= 0:
		return nil, fmt.Errorf("positive write wait period required")
	case cfg.PingPeriod <= 0:
		return nil, fmt.Errorf("positive ping period required")
	case cfg.ActivityCheckPeriod <= 0:
		return nil, fmt.Errorf("positive activity check period required")
	case cfg.PingPeriod >= cfg.ReadWait:
		return nil, fmt.Errorf("ping period should be less than read wait")
	}
	return a, nil
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
	activityCheckTicker := time.NewTicker(s.ActivityCheckPeriod)
	var wg sync.WaitGroup
	go func() {
		wg.Wait()
		pingTicker.Stop()
		activityCheckTicker.Stop()
		s.Runner.Finish()
		s.Conn.Close()
		// TODO: send playerDelete message, test this
	}()
	s.readActive = false
	wg.Add(1)
	go s.readMessages(ctx, out, &wg)
	wg.Add(1)
	go s.writeMessages(ctx, in, &wg, pingTicker, activityCheckTicker)
	return nil
}

// readMessages receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessages(ctx context.Context, out chan<- message.Message, wg *sync.WaitGroup) {
	defer wg.Done()
	for { // BLOCKING
		m, err := s.readMessage()
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				var reason string
				if err != errSocketClosed {
					reason = fmt.Sprintf("reading socket messages stopped for player %v: %v", s, err)
					s.Log.Print(reason)
					s.Conn.WriteClose(reason)
				}
				return
			}
		}
		out <- *m
		s.readActive = true
	}
}

// writeMessages sends messages from the outbound messages channel to the connected socket.
// Messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel.
// The tickers are used to periodically write messages or check for read activity.
func (s *Socket) writeMessages(ctx context.Context, out <-chan message.Message, wg *sync.WaitGroup,
	pingTicker, activityTicker *time.Ticker) {
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
		case m, ok := <-out:
			if !ok {
				closeReason = "server not reading messages"
				return
			}
			err = s.writeMessage(m)
		case <-pingTicker.C:
			err = s.Conn.WritePing()
		case <-activityTicker.C:
			err = s.handleActivityCheck()
		}
		if err != nil {
			closeReason = fmt.Sprintf("writing socket messages stopped for player %v: %v", s, err)
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
	if m.Game == nil {
		return nil, fmt.Errorf("received message not relating to game")
	}
	// Add the player name and address so subscribers of the socket can know who to send responses to because the out channel is shared.
	m.PlayerName = s.PlayerName
	m.Addr = s.Addr
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

// handleActivityCheck ensures the socket has read a message recently.  If so, it writes an HTTPPing message.
func (s *Socket) handleActivityCheck() error {
	if !s.readActive {
		return fmt.Errorf("closing socket due to inactivity")
	}
	s.readActive = false
	m := message.Message{
		Type: message.SocketHTTPPing,
	}
	return s.writeMessage(m)
}
