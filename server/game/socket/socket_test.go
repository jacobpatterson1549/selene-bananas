package socket

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockConn struct {
	ReadJSONFunc               func(m *message.Message) error
	WriteJSONFunc              func(m message.Message) error
	CloseFunc                  func() error
	WritePingFunc              func() error
	WriteCloseFunc             func(reason string) error
	IsUnexpectedCloseErrorFunc func(err error) bool
	RemoteAddrFunc             func() net.Addr
}

func (c *mockConn) ReadJSON(m *message.Message) error {
	return c.ReadJSONFunc(m)
}

func (c *mockConn) WriteJSON(m message.Message) error {
	return c.WriteJSONFunc(m)
}

func (c *mockConn) Close() error {
	return c.CloseFunc()
}

func (c *mockConn) WritePing() error {
	return c.WritePingFunc()
}

func (c *mockConn) WriteClose(reason string) error {
	return c.WriteCloseFunc(reason)
}

func (c *mockConn) IsUnexpectedCloseError(err error) bool {
	return c.IsUnexpectedCloseErrorFunc(err)
}

func (c *mockConn) RemoteAddr() net.Addr {
	return c.RemoteAddrFunc()
}

// mockConnReadMessage reads the src message into the destination value using reflection.
func mockConnReadMessage(dest *message.Message, src message.Message) {
	srcV := reflect.ValueOf(src)
	destV := reflect.ValueOf(dest)
	destVE := destV.Elem()
	destVE.Set(srcV)
}

// ReadMinimalMessage reads a message into the json that will not cause an error.
func mockConnReadMinimalMessage(dest *message.Message) {
	src := message.Message{
		Game: &game.Info{},
	}
	mockConnReadMessage(dest, src)
}

type mockAddr string

func (a mockAddr) Network() string {
	return string(a) + "_NETWORK"
}

func (a mockAddr) String() string {
	return string(a)
}

func TestNewSocket(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	conn0 := &mockConn{}
	newSocketTests := []struct {
		wantOk bool
		want   *Socket
		Conn
		Config
	}{
		{}, // no Log
		{ // no conn
			Config: Config{
				Log: testLog,
			},
		},
		{ // bad ReadWait
			Conn: conn0,
			Config: Config{
				Log: testLog,
			},
		},
		{ // bad WriteWait
			Conn: conn0,
			Config: Config{
				Log:      testLog,
				ReadWait: 2 * time.Hour,
			},
		},
		{ // bad PingPeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
			},
		},
		{ // bad IdlePeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
			},
		},
		{ // bad HTTPPingPeriod
			Conn: conn0,
			Config: Config{
				Log:        testLog,
				ReadWait:   2 * time.Hour,
				WriteWait:  2 * time.Hour,
				PingPeriod: 1 * time.Hour,
				IdlePeriod: 1 * time.Hour,
			},
		},
		{ // PingPeriod not less than ReadWait
			Conn: conn0,
			Config: Config{
				Log:            testLog,
				ReadWait:       1 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     1 * time.Hour,
				IdlePeriod:     1 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
		},
		{ // ok
			Conn: conn0,
			Config: Config{
				Log:            testLog,
				ReadWait:       2 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     1 * time.Hour,
				IdlePeriod:     1 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
			want: &Socket{
				Conn: conn0,
				Config: Config{
					Log:            testLog,
					ReadWait:       2 * time.Hour,
					WriteWait:      2 * time.Hour,
					PingPeriod:     1 * time.Hour,
					IdlePeriod:     1 * time.Hour,
					HTTPPingPeriod: 15 * time.Hour,
				},
			},
			wantOk: true,
		},
	}
	for i, test := range newSocketTests {
		got, err := test.Config.NewSocket(test.Conn)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v: sockets not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	}
}

func TestRunSocket(t *testing.T) {
	runSocketTests := []struct {
		alreadyRunning bool
		stopFunc       func(cancelFunc context.CancelFunc, in chan<- message.Message)
	}{
		{
			alreadyRunning: true,
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan<- message.Message) {
				cancelFunc()
			},
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan<- message.Message) {
				// the "out" stream is shared, the socket is cancelled by telling it that the player has been deleted through the in stream.
				m := message.Message{
					Type: message.PlayerDelete,
				}
				in <- m
			},
		},
	}
	for i, test := range runSocketTests {
		readBlocker := make(chan struct{})
		var wg sync.WaitGroup
		conn := mockConn{
			ReadJSONFunc: func(m *message.Message) error {
				<-readBlocker
				return errors.New("unexpected close")
			},
			IsUnexpectedCloseErrorFunc: func(err error) bool {
				return true
			},
			CloseFunc: func() error {
				wg.Done()
				return nil
			},
			WriteJSONFunc: func(m message.Message) error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
		}
		cfg := Config{
			Log:            log.New(ioutil.Discard, "test", log.LstdFlags),
			ReadWait:       2 * time.Hour,
			WriteWait:      2 * time.Hour,
			PingPeriod:     1 * time.Hour,
			IdlePeriod:     1 * time.Hour,
			HTTPPingPeriod: 15 * time.Hour,
		}
		s := Socket{
			Conn:   &conn,
			Config: cfg,
			active: true,
		}
		if test.alreadyRunning {
			ctx := context.Background()
			ctx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			in := make(chan message.Message)
			out := make(chan message.Message)
			err := s.Run(ctx, in, out)
			if err != nil {
				t.Errorf("Test %v: unwanted error running socket: %v", i, err)
				continue
			}
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		in := make(chan message.Message)
		out := make(chan message.Message)
		err := s.Run(ctx, in, out)
		switch {
		case test.alreadyRunning:
			if err == nil {
				t.Errorf("Test %v: wanted error running socket that should already be running", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case s.active:
			t.Errorf("Test %v: socket should be set to not active when starting a run", i)
		default:
			if !s.IsRunning() {
				t.Errorf("Test %v wanted socket to be running", i)
			}
			wg.Add(1)
			close(readBlocker)
			test.stopFunc(cancelFunc, in)
			wg.Wait()
			if s.IsRunning() {
				t.Errorf("Test %v: wanted socket to not be running after it finished", i)
			}
			if err := s.Run(ctx, in, out); err == nil {
				t.Errorf("Test %v: wanted error running socket after it is finished", i)
			}
		}
	}
}

func TestSocketReadMessages(t *testing.T) {
	readMessagesTests := []struct {
		readMessageErr         error
		isUnexpectedCloseError bool
		gameMissing            bool
		alreadyCancelled       bool
		wantOk                 bool
	}{
		{
			readMessageErr: errors.New("normal close"),
		},
		{
			readMessageErr:         errors.New("unexpected close"),
			isUnexpectedCloseError: true,
		},
		{
			gameMissing:            true,
			isUnexpectedCloseError: true,
		},
		{
			alreadyCancelled: true,
		},
		{
			wantOk: true,
		},
	}
	for i, test := range readMessagesTests {
		testIn := make(chan message.Message)
		defer close(testIn)
		closeMessageWritten := false
		conn := mockConn{
			ReadJSONFunc: func(m *message.Message) error {
				src := <-testIn
				if test.readMessageErr != nil {
					return test.readMessageErr
				}
				if !test.gameMissing {
					src.Game = &game.Info{}
				}
				mockConnReadMessage(m, src)
				return nil
			},
			IsUnexpectedCloseErrorFunc: func(err error) bool {
				return test.isUnexpectedCloseError
			},
			CloseFunc: func() error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				closeMessageWritten = true
				return nil
			},
		}
		s := Socket{
			Conn: &conn,
			Config: Config{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		out := make(chan message.Message, 1) // check the active flag before reading from this
		var wg sync.WaitGroup
		wg.Add(1)
		go s.readMessages(ctx, out, &wg)
		if test.alreadyCancelled {
			cancelFunc()
		}
		testIn <- message.Message{}
		switch {
		case !test.wantOk:
			wg.Wait()
			if s.active {
				t.Errorf("Test %v: wanted socket to be not active because it failed after reading a message", i)
			}
			if test.isUnexpectedCloseError != closeMessageWritten {
				t.Errorf("Test %v: wanted close message to be written when error is unexpected", i)
			}
		case !s.active:
			t.Errorf("Test %v: wanted socket to still be active", i)
		default:
			_, ok := <-out
			if !ok {
				t.Errorf("Test %v: wanted message to be read", i)
			}
		}
	}
}

func TestSocketWriteMessages(t *testing.T) {
	writeMessagesTests := []struct {
		cancel       bool
		outClosed    bool
		m            message.Message
		wantM        message.Message
		writeErr     error
		pingTick     bool
		pingErr      error
		httpPingTick bool
		activeTick   bool
		active       bool
		wantOk       bool
	}{
		{ // context canceled
			cancel: true,
		},
		{ // outbound channel closed
			outClosed: true,
		},
		{ // normal message
			m: message.Message{
				Type: message.Chat,
				Info: "server says hi",
			},
			wantM: message.Message{
				Type: message.Chat,
				Info: "server says hi",
			},
			wantOk: true,
		},
		{ // socket/player removed
			m: message.Message{
				Type: message.PlayerDelete,
			},
			wantM: message.Message{
				Type: message.PlayerDelete,
			},
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
		{ // http ping to keep heroku server alive
			httpPingTick: true,
			wantM: message.Message{
				Type: message.SocketHTTPPing,
			},
			wantOk: true,
		},
		{ // http ping to keep heroku server alive
			httpPingTick: true,
			writeErr:     errors.New("error writing HTTP ping"),
		},
		{ // socket activity check, but socket is not active
			activeTick: true,
		},
		{ // socket activity check
			activeTick: true,
			active:     true,
			wantOk:     true,
		},
	}
	for i, test := range writeMessagesTests {
		closeMessageWritten := false
		writtenMessages := make(chan message.Message, 1)
		pingC := make(chan time.Time, 1)
		pingTicker := &time.Ticker{
			C: pingC,
		}
		httpPingC := make(chan time.Time, 1)
		httpPingTicker := &time.Ticker{
			C: httpPingC,
		}
		idleC := make(chan time.Time, 1)
		idleTicker := &time.Ticker{
			C: idleC,
		}
		conn := mockConn{
			CloseFunc: func() error {
				return nil
			},
			WriteJSONFunc: func(m message.Message) error {
				writtenMessages <- m
				return test.writeErr
			},
			WriteCloseFunc: func(reason string) error {
				closeMessageWritten = len(reason) > 0
				return nil
			},
			WritePingFunc: func() error {
				close(pingC)
				return test.pingErr
			},
		}
		s := Socket{
			Conn: &conn,
			Config: Config{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		out := make(chan message.Message, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		defer cancelFunc()
		switch {
		case test.cancel:
			cancelFunc()
		case test.outClosed:
			close(out)
		case test.pingTick:
			pingC <- time.Now()
		case test.httpPingTick:
			httpPingC <- time.Now()
		case test.activeTick:
			if test.active {
				s.active = true
			}
			idleC <- time.Now()
		default:
			out <- test.m
		}
		go s.writeMessages(ctx, out, &wg, pingTicker, httpPingTicker, idleTicker)
		switch {
		case !test.wantOk:
			wg.Wait()
			if s.active {
				t.Errorf("Test %v: wanted socket to be not active because it failed after reading a message", i)
			}
			if !closeMessageWritten {
				t.Errorf("Test %v: wanted close message to be written", i)
			}
		case test.pingTick:
			if _, ok := <-pingC; !ok {
				t.Errorf("Test %v: wanted websocket ping to close mock ping channel", i)
			}
		case test.activeTick:
		// 	// TODO: yield to the writeMessages() goroutine so the idle tick can be handled
		// 	runtime.Gosched()
		// 	if s.active {
		// 		t.Errorf("Test %v: wanted socket to not be active after idle tick", i)
		// 	}
		default:
			gotM := <-writtenMessages
			if !reflect.DeepEqual(test.wantM, gotM) {
				t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
		}
	}
}
