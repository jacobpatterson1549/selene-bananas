package socket

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type mockConn struct {
	ReadMessageFunc            func(m *message.Message) error
	WriteMessageFunc           func(m message.Message) error
	SetReadDeadlineFunc        func(t time.Time) error
	SetWriteDeadlineFunc       func(t time.Time) error
	CloseFunc                  func() error
	WritePingFunc              func() error
	WriteCloseFunc             func(reason string) error
	IsUnexpectedCloseErrorFunc func(err error) bool
	RemoteAddrFunc             func() net.Addr
}

func (c *mockConn) ReadMessage(m *message.Message) error {
	return c.ReadMessageFunc(m)
}

func (c *mockConn) WriteMessage(m message.Message) error {
	return c.WriteMessageFunc(m)
}

func (c *mockConn) SetReadDeadline(t time.Time) error {
	return c.SetReadDeadlineFunc(t)
}

func (c *mockConn) SetWriteDeadline(t time.Time) error {
	return c.SetWriteDeadlineFunc(t)
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
			Config: Config{
				Log: testLog,
			},
		},
		{ // bad ReadWait
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			Config: Config{
				Log:      testLog,
				TimeFunc: timeFunc,
			},
		},
		{ // bad WriteWait
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			Config: Config{
				Log:      testLog,
				TimeFunc: timeFunc,
				ReadWait: 2 * time.Hour,
			},
		},
		{ // bad PingPeriod
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			Config: Config{
				Log:       testLog,
				TimeFunc:  timeFunc,
				ReadWait:  2 * time.Hour,
				WriteWait: 2 * time.Hour,
			},
		},
		{ // bad ActivityCheckPeriod
			playerName: pn,
			Conn:       conn0,
			remoteAddr: addr,
			Config: Config{
				Log:        testLog,
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
			Config: Config{
				Log:            testLog,
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
			Config: Config{
				Log:            testLog,
				TimeFunc:       timeFunc,
				ReadWait:       2 * time.Hour,
				WriteWait:      2 * time.Hour,
				PingPeriod:     4 * time.Hour,
				HTTPPingPeriod: 15 * time.Hour,
			},
			want: &Socket{
				PlayerName: pn,
				Addr:       addr,
				Conn:       conn0,
				Config: Config{
					Log:            testLog,
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
		switch test.Conn.(type) {
		case *mockConn:
			test.Conn.(*mockConn).RemoteAddrFunc = func() net.Addr {
				return test.remoteAddr
			}
		}
		got, err := test.Config.NewSocket(test.playerName, test.Conn)
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
		alreadyRunning bool
		stopFunc       func(cancelFunc context.CancelFunc, in chan<- message.Message)
	}{
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan<- message.Message) {
				cancelFunc()
			},
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan<- message.Message) {
				// the "out" stream is shared, the socket is cancelled by telling it that the player has been deleted through the in stream.
				m := message.Message{
					Type: message.PlayerRemove,
				}
				in <- m
			},
		},
	}
	for i, test := range runSocketTests {
		readBlocker := make(chan struct{})
		var closedMu sync.Mutex
		closedCount := 0
		var wg sync.WaitGroup
		wg.Add(3)
		conn := mockConn{
			ReadMessageFunc: func(m *message.Message) error {
				<-readBlocker
				return errors.New("unexpected close")
			},
			SetReadDeadlineFunc: func(t time.Time) error {
				return nil
			},
			IsUnexpectedCloseErrorFunc: func(err error) bool {
				return true
			},
			CloseFunc: func() error {
				closedMu.Lock()
				defer closedMu.Unlock()
				if closedCount <= 3 { // HACK readMessages(), writeMessages(), and Run() all call Close() to make this work.
					wg.Done()
					closedCount++
				}
				return nil
			},
			WriteMessageFunc: func(m message.Message) error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
		}
		cfg := Config{
			Log:            log.New(ioutil.Discard, "test", log.LstdFlags),
			TimeFunc:       func() int64 { return 0 },
			ReadWait:       2 * time.Hour,
			WriteWait:      2 * time.Hour,
			PingPeriod:     1 * time.Hour,
			HTTPPingPeriod: 3 * time.Hour,
		}
		addr := mockAddr("some.addr")
		s := Socket{
			Conn:   &conn,
			Config: cfg,
			Addr:   addr,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		in := make(chan message.Message)
		out := make(chan message.Message)
		s.Run(ctx, in, out)
		close(readBlocker)
		test.stopFunc(cancelFunc, in)
		wg.Wait()
		got := <-out
		switch {
		case got.Type != message.SocketClose, got.PlayerName != s.PlayerName, got.Addr != addr:
			t.Errorf("Test %v: wanted SocketClose with socket address and player name", i)
		}
	}
}

func TestSocketReadMessages(t *testing.T) {
	pn := player.Name("selene")
	addr := mockAddr("selene.pc.addr")
	readMessagesTests := []struct {
		readMessageErr         error
		isUnexpectedCloseError bool
		gameMissing            bool
		alreadyCancelled       bool
		wantOk                 bool
		debug                  bool
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
		{
			wantOk: true,
			debug:  true,
		},
	}
	for i, test := range readMessagesTests {
		testIn := make(chan message.Message)
		defer close(testIn)
		conn := mockConn{
			ReadMessageFunc: func(m *message.Message) error {
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
			SetReadDeadlineFunc: func(t time.Time) error {
				return nil
			},
			IsUnexpectedCloseErrorFunc: func(err error) bool {
				return test.isUnexpectedCloseError
			},
			CloseFunc: func() error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
		}
		var bb bytes.Buffer
		log := log.New(&bb, "test", log.LstdFlags)
		s := Socket{
			Conn: &conn,
			Config: Config{
				Log:      log,
				TimeFunc: func() int64 { return 0 },
				Debug:    test.debug,
			},
			PlayerName: pn,
			Addr:       addr,
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
		case test.debug:
			if bb.Len() == 0 {
				t.Errorf("Test %v: wanted message to be logged", i)
			}
		case bb.Len() != 0:
			t.Errorf("Test %v: wanted no message to be logged", i)
		default:
			got, ok := <-out
			if !ok {
				t.Errorf("Test %v: wanted message to be read", i)
			}
			want := message.Message{
				Game:       &game.Info{},
				PlayerName: pn,
				Addr:       addr,
			}
			if !reflect.DeepEqual(want, got) {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, want, got)
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
		wantOk       bool
		debug        bool
	}{
		{ // context canceled
			cancel: true,
		},
		{ // outbound channel closed
			outClosed: true,
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
		{ // normal message, with debug
			m:      message.Message{},
			wantM:  message.Message{},
			wantOk: true,
			debug:  true,
		},
		{ // socket/player removed
			m: message.Message{
				Type: message.PlayerRemove,
			},
			wantM: message.Message{
				Type: message.PlayerRemove,
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
		{ // httpPing: ok
			httpPingTick: true,
			wantM: message.Message{
				Type: message.SocketHTTPPing,
			},
			wantOk:     true,
		},
		{ // activity check, but ping write fails
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
		closed := false
		conn := mockConn{
			CloseFunc: func() error {
				closed = true
				return nil
			},
			WriteMessageFunc: func(m message.Message) error {
				writtenMessages <- m
				return test.writeErr
			},
			SetWriteDeadlineFunc: func(t time.Time) error {
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
			WritePingFunc: func() error {
				return test.pingErr
			},
		}
		var bb bytes.Buffer
		log := log.New(&bb, "test", log.LstdFlags)
		s := Socket{
			Conn: &conn,
			Config: Config{
				Log:      log,
				TimeFunc: func() int64 { return 0 },
				Debug:    test.debug,
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
		default:
			out <- test.m
		}
		go s.writeMessages(ctx, out, &wg, pingTicker, httpPingTicker)
		switch {
		case !test.wantOk:
			wg.Wait()
			if !closed {
				t.Errorf("Test %v: wanted socket to be not active because it failed after writing a message", i)
			}
		case !test.pingTick:
			gotM := <-writtenMessages
			switch {
			case !reflect.DeepEqual(test.wantM, gotM):
				t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			case test.debug:
				if bb.Len() == 0 {
					t.Errorf("Test %v: wanted message to be logged", i)
				}
			case bb.Len() != 0:
				t.Errorf("Test %v: wanted no message to be logged", i)
			}
		}
	}
}

func TestWriteClose(t *testing.T) {
	writeCloseTests := []struct {
		reason           string
		alreadyClosed    bool
		wantReasonLogged bool
	}{
		{},
		{
			reason:           "server halted",
			wantReasonLogged: true,
		},
		{
			alreadyClosed: true,
		},
	}
	for i, test := range writeCloseTests {
		writeCloseCalled := false
		conn := mockConn{
			WriteCloseFunc: func(reason string) error {
				writeCloseCalled = true
				if test.alreadyClosed {
					return errors.New("already closed")
				}
				return nil
			},
		}
		var bb bytes.Buffer
		log := log.New(&bb, "test", 0)
		s := Socket{
			Conn: &conn,
			Config: Config{
				Log: log,
			},
		}
		s.writeClose(test.reason)
		switch {
		case !writeCloseCalled:
			t.Errorf("Test %v: write close not called", i)
		case test.alreadyClosed:
			if bb.Len() != 0 {
				t.Errorf("Test %v: wanted no reason logged when already closed", i)
			}
		default:
			got := bb.String()
			if !strings.Contains(got, test.reason) {
				t.Errorf("Test %v: wanted logged reason to contain '%v', got '%v'", i, test.reason, got)
			}
		}
	}
}
