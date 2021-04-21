package socket

import (
	"context"
	"errors"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
	"github.com/jacobpatterson1549/selene-bananas/server/log/logtest"
)

func TestNewSocket(t *testing.T) {
	testLog := logtest.DiscardLogger
	timeFunc := func() int64 { return 0 }
	pn := player.Name("selene")
	addr := mockAddr("selene.pc")
	conn0 := &mockConn{}
	newSocketTests := []struct {
		wantOk     bool
		want       *Socket
		playerName player.Name
		Conn
		remoteAddr net.Addr
		log        log.Logger
		Config
	}{
		{}, // no playerName
		{ // no conn
			playerName: pn,
		},
		{ // no remote addr
			playerName: pn,
			Conn:       conn0,
		},
		{ // no log
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
		},
		{ // no timeFunc
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
		},
		{ // bad ReadWait
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc: timeFunc,
			},
		},
		{ // bad WriteWait
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc: timeFunc,
				ReadWait: 2 * time.Hour,
			},
		},
		{ // bad PingPeriod
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc:  timeFunc,
				ReadWait:  2 * time.Hour,
				WriteWait: 2 * time.Hour,
			},
		},
		{ // bad ActivityCheckPeriod
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc:   timeFunc,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
			},
		},
		{ // PingPeriod not less than WriteWait
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc:       timeFunc,
				ReadWait:       1 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     1 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
		},
		{ // ok
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				TimeFunc:       timeFunc,
				ReadWait:       2 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     4 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
			want: &Socket{
				PlayerName: pn,
				Addr:       message.Addr(addr),
				Conn:       conn0,
				log:        testLog,
				Config: Config{
					ReadWait:       2 * time.Hour,
					WriteWait:      2 * time.Hour,
					PingPeriod:     4 * time.Hour,
					HTTPPingPeriod: 15 * time.Hour,
				},
			},
			wantOk: true,
		},
		{ // ok with debug
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			log:        testLog,
			Config: Config{
				Debug:          true,
				TimeFunc:       timeFunc,
				ReadWait:       2 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     4 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
			want: &Socket{
				PlayerName: pn,
				Addr:       message.Addr(addr),
				Conn:       conn0,
				log:        testLog,
				Config: Config{
					Debug:          true,
					ReadWait:       2 * time.Hour,
					WriteWait:      2 * time.Hour,
					PingPeriod:     4 * time.Hour,
					HTTPPingPeriod: 15 * time.Hour,
				},
			},
			wantOk: true,
		},
	}
	for i, test := range newSocketTests {
		if test.Conn != nil {
			test.Conn.(*mockConn).RemoteAddrFunc = func() net.Addr {
				return test.remoteAddr
			}
		}
		got, err := test.Config.NewSocket(test.log, test.playerName, test.Conn)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			got.TimeFunc = nil // funcs cannot be compared
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v: sockets not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	}
}

func TestRunSocket(t *testing.T) {
	runSocketTests := []struct {
		callCancelFunc bool
	}{
		{},
		{
			callCancelFunc: true,
		},
	}
	for i, test := range runSocketTests {
		var wg sync.WaitGroup
		wg.Add(1)
		conn := mockConn{
			SetReadDeadlineFunc: func(t time.Time) error {
				wg.Done() // ensures the socket is run
				return errors.New("socket close")
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
			CloseFunc: func() error {
				return nil
			},
		}
		cfg := Config{
			TimeFunc:       func() int64 { return 0 },
			ReadWait:       2 * time.Hour,
			WriteWait:      2 * time.Hour,
			PingPeriod:     1 * time.Hour,
			HTTPPingPeriod: 3 * time.Hour,
		}
		s := Socket{
			log:    logtest.DiscardLogger,
			Conn:   &conn,
			Config: cfg,
			Addr:   "some.addr",
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		in := make(chan message.Message)
		out := make(chan message.Message, 1)
		if test.callCancelFunc {
			cancelFunc()
		}
		s.Run(ctx, &wg, in, out)
		switch {
		case test.callCancelFunc:
			wg.Wait()
			if len(out) != 0 {
				t.Errorf("Test %v: wanted no messages sent back on out channel", i)
			}
		default:
			wantM := message.Message{
				Type:       message.SocketClose,
				PlayerName: s.PlayerName,
				Addr:       "some.addr",
			}
			gotM := <-out
			if !reflect.DeepEqual(wantM, gotM) {
				t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, wantM, gotM)
			}
		}
		cancelFunc()
		wg.Wait() // ensure SetReadDeadline is called
	}
}

func TestReadMessagesSync(t *testing.T) {
	pn := player.Name("selene")
	addr := message.Addr("selene.pc.addr")
	readMessagesTests := []struct {
		callCancelFunc     bool
		setReadDeadlineErr error
		readMessageErr     error
		isNormalCloseErr   bool
		gameMissing        bool
		debug              bool
		wantOk             bool
	}{
		{
			callCancelFunc: true,
		},
		{
			setReadDeadlineErr: errors.New("could not set read deadline"),
		},
		{
			readMessageErr:   errors.New("normal close"),
			isNormalCloseErr: true,
		},
		{
			readMessageErr: errors.New("unexpected close"),
		},
		{
			gameMissing: true,
		},
		{
			wantOk: true,
		},
		{
			wantOk: true,
			debug:  true,
		},
	}
	for i, test := range readMessagesTests {
		setPongHandlerFuncCalled := false
		normalMessageInfo := "normal message"
		j := 0
		conn := mockConn{
			ReadMessageFunc: func(m *message.Message) error {
				if test.readMessageErr != nil {
					return test.readMessageErr
				}
				src := message.Message{
					Info: normalMessageInfo,
				}
				if !test.gameMissing {
					src.Game = &game.Info{}
				}
				*m = src
				j++
				if test.wantOk && j > 1 {
					test.isNormalCloseErr = true
					return errors.New("ok read cancel") // only read one message
				}
				return nil
			},
			SetReadDeadlineFunc: func(t time.Time) error {
				return test.setReadDeadlineErr
			},
			IsNormalCloseFunc: func(err error) bool {
				return test.isNormalCloseErr
			},
			CloseFunc: func() error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
			SetPongHandlerFunc: func(h func(appData string) error) {
				setPongHandlerFuncCalled = true
			},
		}
		log := logtest.NewLogger()
		s := Socket{
			Conn: &conn,
			log:  log,
			Config: Config{
				Debug:    test.debug,
				TimeFunc: func() int64 { return 0 },
			},
			PlayerName: pn,
			Addr:       addr,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		wantNumMessagesRead := 1 // the last message should Type.SocketClose
		switch {
		case test.callCancelFunc:
			wantNumMessagesRead--
			cancelFunc()
		case test.wantOk:
			wantNumMessagesRead++
		}
		out := make(chan message.Message, wantNumMessagesRead)
		var wg sync.WaitGroup
		wg.Add(1)
		go s.readMessagesSync(ctx, &wg, out)
		wg.Wait()
		gotMessages := make([]message.Message, wantNumMessagesRead)
		for j := 0; j < wantNumMessagesRead; j++ {
			gotMessages[j] = <-out
		}
		switch {
		case len(out) != 0:
			t.Errorf("Test %v: extra messages exist on out channel", i)
		case wantNumMessagesRead != len(gotMessages):
			t.Errorf("Test %v: wanted %v messages sent on out channel, got %v", i, wantNumMessagesRead, len(out))
		case test.callCancelFunc:
			// NOOP
		case gotMessages[len(gotMessages)-1].Type != message.SocketClose,
			gotMessages[len(gotMessages)-1].PlayerName != pn,
			gotMessages[len(gotMessages)-1].Addr != addr:
			t.Errorf("Test %v: wanted last message to be socket close, got %v", i, gotMessages[len(gotMessages)-1])
		case test.setReadDeadlineErr == nil && !setPongHandlerFuncCalled:
			t.Errorf("Test %v: wanted pong handler to be set", i)
		case !test.wantOk && log.Empty() && !test.isNormalCloseErr:
			t.Errorf("Test %v: wanted message to be logged", i)
		case test.wantOk && !log.Empty() != test.debug:
			t.Errorf("Test %v: wanted message to be logged (%v), got '%v'", i, test.debug, log.String())
		case test.wantOk && gotMessages[0].Info != normalMessageInfo:
			t.Errorf("Test %v: wanted first message to be normal message, got %v", i, gotMessages[0])
		}
		cancelFunc()
	}
}

func TestWriteMessagesSync(t *testing.T) {
	writeMessagesTests := []struct {
		callCancelFunc      bool
		inClosed            bool
		m                   message.Message
		wantM               message.Message
		setWriteDeadlineErr error
		writeErr            error
		pingTick            bool
		pingErr             error
		httpPingTick        bool
		wantOk              bool
	}{
		{
			callCancelFunc: true,
		},
		{ // inbound channel closed
			inClosed: true,
		},
		{ // normal message
			m: message.Message{
				Type: message.GameChat,
				Info: "server says hi",
			},
			wantM: message.Message{
				Type: message.GameChat,
				Info: "server says hi",
			},
			wantOk: true,
		},
		{ // normal message: setWriteDeadline  error
			m:                   message.Message{},
			setWriteDeadlineErr: errors.New("setWriteDeadline error"),
		},
		{ // write error
			m:        message.Message{},
			writeErr: errors.New("problem writing message"),
		},
		{ // websocket ping
			pingTick: true,
			wantOk:   true,
		},
		{ // websocket ping
			pingTick: true,
			pingErr:  errors.New("error writing ping"),
		},
		{ // websocket ping: setWriteDeadline error
			pingTick:            true,
			setWriteDeadlineErr: errors.New("setWriteDeadline error"),
		},
		{ // httpPing: ok
			httpPingTick: true,
			wantM: message.Message{
				Type: message.SocketHTTPPing,
			},
			wantOk: true,
		},
		{ // httpPing, but ping write fails
			httpPingTick: true,
			writeErr:     errors.New("error writing activity check ping"),
		},
	}
	for i, test := range writeMessagesTests {
		writtenMessages := make(chan message.Message, 1)
		pingC := make(chan time.Time, 1)
		pingTicker := &time.Ticker{
			C: pingC,
		}
		httpPingC := make(chan time.Time, 1)
		httpPingTicker := &time.Ticker{
			C: httpPingC,
		}
		in := make(chan message.Message, 1)
		conn := mockConn{
			CloseFunc: func() error {
				return nil
			},
			WriteMessageFunc: func(m message.Message) error {
				close(in) // only read once
				switch {
				case test.writeErr != nil:
					return test.writeErr
				default:
					writtenMessages <- m
					return nil
				}
			},
			SetWriteDeadlineFunc: func(t time.Time) error {
				return test.setWriteDeadlineErr
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
			WritePingFunc: func() error {
				close(in)
				return test.pingErr
			},
		}
		log := logtest.NewLogger()
		s := Socket{
			Conn: &conn,
			log:  log,
			Config: Config{
				TimeFunc: func() int64 { return 0 },
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		switch {
		case test.inClosed:
			close(in)
		case test.pingTick:
			pingC <- time.Time{}
		case test.httpPingTick:
			httpPingC <- time.Time{}
		case test.wantOk, test.writeErr != nil, test.setWriteDeadlineErr != nil:
			in <- test.m
		}
		var wg sync.WaitGroup
		wg.Add(1)
		go s.writeMessagesSync(ctx, &wg, in, pingTicker, httpPingTicker)
		switch {
		case test.callCancelFunc:
			// NOOP
		case !test.wantOk && len(writtenMessages) != 0:
			t.Errorf("Test %v: wanted no messages written to connection", i)
		case !test.wantOk && !test.inClosed && test.setWriteDeadlineErr != nil:
			close(in)
		case test.wantOk && !test.pingTick:
			gotM := <-writtenMessages
			if !reflect.DeepEqual(test.wantM, gotM) {
				t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
		}
		cancelFunc()
		wg.Wait()
	}
}

func TestWriteMessage(t *testing.T) {
	writeMessageTests := []struct {
		m            message.Message
		debug        bool
		connWriteErr error
		wantOk       bool
	}{
		{
			wantOk: true,
		},
		{
			debug:  true,
			wantOk: true,
		},
		{
			connWriteErr: errors.New("cannot write message to connection"),
		},
		{
			m: message.Message{
				Type: message.PlayerRemove,
			},
		},
	}
	for i, test := range writeMessageTests {
		log := logtest.NewLogger()
		s := Socket{
			log: log,
			Config: Config{
				Debug: test.debug,
			},
			Conn: &mockConn{
				WriteMessageFunc: func(m message.Message) error {
					return test.connWriteErr
				},
			},
		}
		err := s.writeMessage(test.m)
		switch {
		case test.debug != !log.Empty():
			t.Errorf("Test %v: wanted debug only when debug is on, got %v", i, log.String())
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
	}
}

func TestWriteClose(t *testing.T) {
	writeCloseTests := []struct {
		connCloseErr error
		reasonErr    error
		wantLog      bool
	}{
		{},
		{
			connCloseErr: errors.New("cannot write message to connection"),
		},
		{
			reasonErr:    errSocketClosed,
		},
		{
			reasonErr:    errServerShuttingDown,
			connCloseErr: errors.New("cannot write message to connection"),
			wantLog:      true, // should still log to server logs
		},
		{
			reasonErr: errServerShuttingDown,
			wantLog:   true,
		},
	}
	for i, test := range writeCloseTests {
		log := logtest.NewLogger()
		s := Socket{
			log: log,
			Conn: &mockConn{
				WriteCloseFunc: func(reason string) error {
					return test.connCloseErr
				},
			},
		}
		s.writeClose(test.reasonErr)
		if test.wantLog != !log.Empty() {
			t.Errorf("Test %v: wanted log (%v), got '%v'", i, test.wantLog, log.String())
		}
	}
}

// TestWriteMessagesSkipSend ensures that no messages are sent on the connection after it first fails.
// This prevents deadlocks if the lobby keeps sending it messages before it is closed.
func TestWriteMessagesSkipSend(t *testing.T) {
	numMessagesWritten := 0
	timeFunc := func() int64 { return 0 }
	conn := mockConn{
		CloseFunc: func() error {
			return nil
		},
		WriteMessageFunc: func(m message.Message) error {
			numMessagesWritten++
			return nil
		},
		SetWriteDeadlineFunc: func(t time.Time) error {
			if numMessagesWritten == 0 {
				return nil
			}
			return errors.New("connection cannot set write deadline")
		},
		WriteCloseFunc: func(reason string) error {
			return nil
		},
		WritePingFunc: func() error {
			return nil
		},
	}
	s := Socket{
		log:  logtest.DiscardLogger,
		Conn: &conn,
		Config: Config{
			TimeFunc: timeFunc,
		},
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	in := make(chan message.Message, 1)
	pingTicker := &time.Ticker{
		C: make(chan time.Time),
	}
	httpPingTicker := &time.Ticker{
		C: make(chan time.Time),
	}
	wg.Add(1)
	go s.writeMessagesSync(ctx, &wg, in, pingTicker, httpPingTicker)
	for i := 0; i < 3; i++ {
		in <- message.Message{}
	}
	cancelFunc()
	wg.Wait()
	if numMessagesWritten != 1 {
		t.Errorf("wanted only 1 message written to the bad connection, got %v", numMessagesWritten)
	}
}
