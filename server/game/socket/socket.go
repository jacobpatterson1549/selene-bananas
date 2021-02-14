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
)

type (
	// Socket reads and writes messages to the browsers
	Socket struct {
		log  *log.Logger
		Conn Conn
		// The reason the read failed.  If a read fails, it is sent as the close message when the socket closes.
		readCloseReason string
		// The reason the writing by by the connection is stopping.  Is sent as the connection close reason if there is no read close reason.
		writeCloseReason string
		Config
		PlayerName player.Name
		net.Addr
	}

	// Config contains commonly shared Socket properties
	Config struct {
		// Debug is a flag that causes the socket to log the types non-ping/pong messages that are read/written
		Debug bool
		// ReadWait is the amout of time that can pass between receiving client messages before timing out.
		ReadWait time.Duration
		// WriteWait is the amout of time that the socket can take to write a message.
		WriteWait time.Duration
		// PingPeriod is how often ping messages should be sent.  Should be less than WriteWait.
		PingPeriod time.Duration
		// HTTPPingPeriod is how frequently to ask the client to send an HTTP request, as Heroku servers shut down if 30 minutes passess between HTTP requests.
		HTTPPingPeriod time.Duration
		// TimeFunc is a function which should supply the current time since the unix epoch.
		// Used to update the read deadline.
		TimeFunc func() int64
	}

	// Conn is the connection than backs the socket
	Conn interface {
		// ReadJSON reads the next message from the connection.
		ReadMessage(m *message.Message) error
		// WriteJSON writes the message to the connection.
		WriteMessage(m message.Message) error
		// SetReadDeadline sets how long a read can take before it returns an error.
		SetReadDeadline(t time.Time) error
		// SetWriteDeadline sets how long a read can take before it returns an error.
		SetWriteDeadline(t time.Time) error
		// SetPongHandler is triggered when the server recieves a pong response from a previous ping
		SetPongHandler(h func(appData string) error)
		// Close closes the connection.
		Close() error
		// WritePing writes a ping message on the connection.
		WritePing() error
		// WriteClose writes a close message on the connection.  The connestion is NOT closed.
		WriteClose(reason string) error
		// IsUnexpectedCloseError determines if the error message is an unexpected close error.
		IsUnexpectedCloseError(err error) bool
		// RemoteAddr gets the remote network address of the connection.
		RemoteAddr() net.Addr
	}
)

var errSocketClosed = fmt.Errorf("socket closed")

// NewSocket creates a socket
func (cfg Config) NewSocket(log *log.Logger, pn player.Name, conn Conn) (*Socket, error) {
	a, err := cfg.validate(log, pn, conn)
	if err != nil {
		return nil, fmt.Errorf("creating socket: validation: %w", err)
	}
	s := Socket{
		log:        log,
		Conn:       conn,
		Config:     cfg,
		PlayerName: pn,
		Addr:       a,
	}
	return &s, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(log *log.Logger, pn player.Name, conn Conn) (net.Addr, error) {
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
	case log == nil:
		return nil, fmt.Errorf("log required")
	case cfg.TimeFunc == nil:
		return nil, fmt.Errorf("time func required")
	case cfg.ReadWait <= 0:
		return nil, fmt.Errorf("positive read wait period required")
	case cfg.WriteWait <= 0:
		return nil, fmt.Errorf("positive write wait period required")
	case cfg.PingPeriod <= 0:
		return nil, fmt.Errorf("positive ping period required")
	case cfg.HTTPPingPeriod <= 0:
		return nil, fmt.Errorf("positive http ping period required")
	case cfg.PingPeriod <= cfg.WriteWait:
		return nil, fmt.Errorf("ping period should be greater than write wait")
	}
	return a, nil
}

// Run writes messages from the connection to the shared "out" channel.
// Run writes messages recieved from the "in" channel to the connection,
// Run writes Socket messages that are recieved to the outbound channel and reads incoming messages onto the inbound channel on separate goroutines.
// The Socket runs until the connection fails for an unexpected reason or the context is cancelled.
func (s *Socket) Run(ctx context.Context, in <-chan message.Message, out chan<- message.Message) {
	pingTicker := time.NewTicker(s.PingPeriod)
	httpPingTicker := time.NewTicker(s.HTTPPingPeriod)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		wg.Wait()
		s.stop(out, pingTicker, httpPingTicker)
	}()
	go s.readMessagesSync(ctx, out, &wg)
	go s.writeMessagesSync(ctx, in, &wg, pingTicker, httpPingTicker)
}

// readMessagesSync receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessagesSync(ctx context.Context, out chan<- message.Message, wg *sync.WaitGroup) {
	defer func() {
		s.Conn.Close() // will casue writeMessages() to fail
		wg.Done()
	}()
	pongHandler := func(appData string) error {
		if err := s.refreshDeadline(s.Conn.SetReadDeadline, s.ReadWait); err != nil {
			err = fmt.Errorf("setting read deadline: %w", err)
			s.writeClose(err.Error())
			return err
		}
		return nil
	}
	if err := pongHandler(""); err != nil {
		return
	}
	s.Conn.SetPongHandler(pongHandler)
	for { // BLOCKING
		m, err := s.readMessage() // BLOCKING
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				if err != errSocketClosed {
					reason := fmt.Sprintf("reading socket messages stopped for player %v: %v", s, err)
					s.writeClose(reason)
				}
				return
			}
		}
		message.Send(*m, out, s.Debug, s.log)
	}
}

// writeMessagesSync sends messages from the outbound messages channel to the connected socket.
// Messages are not sent if the writing is cancelled from the done channel or an error is encountered and sent to the error channel.
// The tickers are used to periodically write message different ping messages.
func (s *Socket) writeMessagesSync(ctx context.Context, in <-chan message.Message, wg *sync.WaitGroup, pingTicker, httpPingTicker *time.Ticker) {
	var closeReason string
	defer func() {
		s.writeClose(closeReason)
		s.Conn.Close() // will casue readMessages() to fail
		wg.Done()
	}()
	var err error
	for { // BLOCKING
		select {
		case <-ctx.Done():
			closeReason = "server shutting down"
			return
		case m, ok := <-in:
			if !ok {
				closeReason = "server not reading messages"
				return
			}
			err = s.writeWrapper(func() error { return s.writeMessage(m) })
		case <-pingTicker.C:
			err = s.writeWrapper(s.Conn.WritePing)
		case <-httpPingTicker.C:
			err = s.handleHTTPPing()
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
	if err := s.Conn.ReadMessage(&m); err != nil { // BLOCKING
		if s.Conn.IsUnexpectedCloseError(err) {
			return nil, fmt.Errorf("unexpected socket closure: %v", err)
		}
		return nil, errSocketClosed
	}
	if s.Debug {
		s.log.Printf("socket reading message with type %v", m.Type)
	}
	if m.Game == nil {
		return nil, fmt.Errorf("received message not relating to game")
	}
	// Add the player name and address so subscribers of the socket can know who to send responses to because the out channel is shared.
	m.PlayerName = s.PlayerName
	m.Addr = s.Addr
	return &m, nil
}

// func writeWrapper sets the write deadline before calling the write function.
func (s *Socket) writeWrapper(writeFunc func() error) error {
	if err := s.refreshDeadline(s.Conn.SetWriteDeadline, s.WriteWait); err != nil {
		return fmt.Errorf("setting write deadline: %v", err)
	}
	return writeFunc()
}

// writeMessage writes a message to the connection.
func (s *Socket) writeMessage(m message.Message) error {
	if s.Debug {
		s.log.Printf("socket writing message with type %v", m.Type)
	}
	if err := s.Conn.WriteMessage(m); err != nil {
		return fmt.Errorf("writing socket message: %v", err)
	}
	if m.Type == message.PlayerRemove {
		return fmt.Errorf("player deleted")
	}
	return nil
}

// handleHTTPPing writes an HTTPPing message.
func (s *Socket) handleHTTPPing() error {
	m := message.Message{
		Type: message.SocketHTTPPing,
	}
	return s.writeMessage(m)
}

// writeClose writes a closeMessage with the reason and closes the connection, logging the reason if successful
func (s *Socket) writeClose(reason string) {
	if err := s.Conn.WriteClose(reason); err != nil {
		return
	}
	s.log.Print(reason)
}

func (s *Socket) refreshDeadline(refreshDeadlineFunc func(t time.Time) error, period time.Duration) error {
	now := s.TimeFunc()
	nowTime := time.Unix(now, 0)
	deadline := nowTime.Add(period)
	if err := refreshDeadlineFunc(deadline); err != nil {
		err = fmt.Errorf("error refreshing ping/pong deadline: %w", err)
		s.log.Print(err)
		return err
	}
	return nil
}

// stop cancels timers, closes the connection, and notifies the out channel that it is closed.
func (s *Socket) stop(out chan<- message.Message, pingTicker, activityCheckTicker *time.Ticker) {
	pingTicker.Stop()
	activityCheckTicker.Stop()
	s.Conn.Close()
	m := message.Message{
		Type:       message.SocketClose,
		PlayerName: s.PlayerName,
		Addr:       s.Addr,
	}
	message.Send(m, out, s.Debug, s.log)
}
