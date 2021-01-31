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
		var wg sync.WaitGroup                // Done called by closeFunc when readMessages() returns
		wg.Add(1)                            // decremented when read errs
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
	// TODO
}
