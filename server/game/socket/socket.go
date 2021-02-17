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
		log        *log.Logger
		Conn       Conn
		PlayerName player.Name
		net.Addr
		Config
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
		// IsNormalClose determines if the error message is an error that implies a normal close or is unexpected.
		IsNormalClose(err error) bool
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
// The run stays active even if it errors out.  The only way to stop it is by closing the 'in' channel.
// If the connection has an error, the socket will send a socketClose message on the out channel, but will still consume and ignore messages from the in channel until it is closed this prevents a channel blockage.
func (s *Socket) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, out chan<- message.Message) {
	pingTicker := time.NewTicker(s.PingPeriod)
	httpPingTicker := time.NewTicker(s.HTTPPingPeriod)
	wg.Add(2)
	go s.readMessagesSync(ctx, wg, out)
	go s.writeMessagesSync(ctx, wg, in, pingTicker, httpPingTicker)
}

// readMessagesSync receives messages from the connected socket and writes the to the messages channel.
// messages are not sent if the reading is cancelled from the done channel or an error is encountered and sent to the error channel.
func (s *Socket) readMessagesSync(ctx context.Context, wg *sync.WaitGroup, out chan<- message.Message) {
	defer wg.Done()
	defer s.closeConn() // will casue writeMessages() to fail, but not stop until the in channel is closed.
	defer s.sendClose(ctx, out)
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
		}
		if err != nil {
			var reason string
			if err != errSocketClosed {
				reason = fmt.Sprintf("reading socket messages stopped for player %v: %v", s, err)
			}
			s.writeClose(reason)
			return
		}
		message.Send(*m, out, s.Debug, s.log)
	}
}

// writeMessagesSync sends messages from the outbound messages channel to the connected socket.
// Messages are not sent if the context is cancelled or an error is encountered and sent to the error channel.
// NOTE: this function does not terminate until the input channel closes.
// The tickers are used to periodically write message different ping messages.
func (s *Socket) writeMessagesSync(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, pingTicker, httpPingTicker *time.Ticker) {
	defer wg.Done()
	defer s.closeConn() // will casue readMessages() to fail
	var err error
	skipWrite, stopWrite := false, false
	write := func(writeFunc func() error) error {
		if skipWrite {
			return fmt.Errorf("skipping write for socket (%v) because an error has already occured", s)
		}
		if err := s.refreshDeadline(s.Conn.SetWriteDeadline, s.WriteWait); err != nil {
			return fmt.Errorf("setting write deadline: %v", err)
		}
		return writeFunc()
	}
	for { // BLOCKING
		select {
		case <-ctx.Done():
			s.writeClose("server shutting down")
			return
		case m, ok := <-in:
			switch {
			case !ok:
				err = fmt.Errorf("server shutting down")
				stopWrite = true
			default:
				err = write(func() error { return s.writeMessage(m) })
			}
		case <-pingTicker.C:
			err = write(s.Conn.WritePing)
		case <-httpPingTicker.C:
			m := message.Message{
				Type: message.SocketHTTPPing,
			}
			err = write(func() error { return s.writeMessage(m) })
		}
		if err != nil {
			closeReason := err.Error()
			s.writeClose(closeReason)
			s.closeConn() // will casue readMessages() to fail
			if stopWrite {
				return
			}
			skipWrite = true
		}
	}
}

// readMessage reads the next message from the connection.
func (s *Socket) readMessage() (*message.Message, error) {
	var m message.Message
	if err := s.Conn.ReadMessage(&m); err != nil { // BLOCKING
		if !s.Conn.IsNormalClose(err) {
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

// writeClose writes a closeMessage with the reason, logging the reason if successful.  Thread safe.
func (s *Socket) writeClose(reason string) {
	if err := s.Conn.WriteClose(reason); err != nil {
		return
	}
	if len(reason) != 0 {
		s.log.Print(reason)
	}
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

// sendClose notifies the out channel that it is closed, hopefully causing it to close the in channel.
func (s *Socket) sendClose(ctx context.Context, out chan<- message.Message) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	m := message.Message{
		Type:       message.SocketClose,
		PlayerName: s.PlayerName,
		Addr:       s.Addr,
	}
	message.Send(m, out, s.Debug, s.log)
}

// closeConn closes the connection of the socket.  Thread safe.
func (s *Socket) closeConn() {
	s.Conn.Close()
}
