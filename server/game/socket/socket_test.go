package socket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

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
		log        *log.Logger
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
				Addr:       addr,
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
				Addr:       addr,
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
	readBlocker := make(chan struct{})
	var closeWG sync.WaitGroup
	closeWG.Add(3)
	conn := mockConn{
		SetReadDeadlineFunc: func(t time.Time) error {
			return fmt.Errorf("could not set read deadline")
		},
		CloseFunc: func() error {
			closeWG.Done()
			return nil
		},
		WriteCloseFunc: func(reason string) error {
			return nil
		},
		SetWriteDeadlineFunc: func(t time.Time) error {
			return fmt.Errorf("could not set write deadline")
		},
	}
	cfg := Config{
		TimeFunc:       func() int64 { return 0 },
		ReadWait:       2 * time.Hour,
		WriteWait:      2 * time.Hour,
		PingPeriod:     1 * time.Hour,
		HTTPPingPeriod: 3 * time.Hour,
	}
	addr := mockAddr("some.addr")
	s := Socket{
		log:    log.New(ioutil.Discard, "test", log.LstdFlags),
		Conn:   &conn,
		Config: cfg,
		Addr:   addr,
	}
	in := make(chan message.Message)
	out := make(chan message.Message, 1)
	s.Run(in, out)
	close(readBlocker)
	close(in) // will cause the socket to stop
	closeWG.Wait()
	got := <-out
	switch {
	case got.Type != message.SocketClose, got.PlayerName != s.PlayerName, got.Addr != addr:
		t.Errorf("wanted SocketClose with socket address and player name")
	}
}

func TestReadMessagesSync(t *testing.T) {
	pn := player.Name("selene")
	addr := mockAddr("selene.pc.addr")
	readMessagesTests := []struct {
		setReadDeadlineErr error
		readMessageErr     error
		isNormalCloseErr   bool
		gameMissing        bool
		debug              bool
		wantOk             bool
	}{
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
		var closeWG sync.WaitGroup
		closeWG.Add(1)
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
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
				mockConnReadMessage(m, src)
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
				closeWG.Done()
				return nil
			},
			WriteCloseFunc: func(reason string) error {
				return nil
			},
			SetPongHandlerFunc: func(h func(appData string) error) {
				setPongHandlerFuncCalled = true
			},
		}
		var bb bytes.Buffer
		s := Socket{
			Conn: &conn,
			log:  log.New(&bb, "", 0),
			Config: Config{
				Debug:    test.debug,
				TimeFunc: func() int64 { return 0 },
			},
			PlayerName: pn,
			Addr:       addr,
		}
		wantNumMessagesRead := 1 // the last message should Type.SocketClose
		if test.wantOk {
			wantNumMessagesRead++
		}
		out := make(chan message.Message, wantNumMessagesRead)
		go s.readMessagesSync(out)
		closeWG.Wait()
		gotMessages := make([]message.Message, wantNumMessagesRead)
		for j := 0; j < wantNumMessagesRead; j++ {
			gotMessages[j] = <-out
		}
		switch {
		case len(out) != 0:
			t.Errorf("Test %v: extra messages exist on out channel", i)
		case wantNumMessagesRead != len(gotMessages):
			t.Errorf("Test %v: wanted %v messages sent on out channel, got %v", i, wantNumMessagesRead, len(out))
		case gotMessages[len(gotMessages)-1].Type != message.SocketClose,
			gotMessages[len(gotMessages)-1].PlayerName != pn,
			gotMessages[len(gotMessages)-1].Addr != addr:
			t.Errorf("Test %v: wanted last message to be socket close, got %v", i, gotMessages[len(gotMessages)-1])
		case test.setReadDeadlineErr == nil && !setPongHandlerFuncCalled:
			t.Errorf("Test %v: wanted pong handler to be set", i)
		case !test.wantOk:
			if bb.Len() == 0 && !test.isNormalCloseErr {
				t.Errorf("Test %v: wanted message to be logged", i)
			}
		case (bb.Len() != 0) != test.debug:
			t.Errorf("Test %v: wanted no message to be logged, got '%v'", i, bb.String())
		case gotMessages[0].Info != normalMessageInfo:
			t.Errorf("Test %v: wanted first message to be normal message, got %v", i, gotMessages[0])
		}
	}
}

func TestWriteMessagesSync(t *testing.T) {
	writeMessagesTests := []struct {
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
		var closeWG sync.WaitGroup
		closeWG.Add(1)
		var closeOnce sync.Once // one call is deferred, one should be called through all non-panicing paths
		in := make(chan message.Message, 1)
		conn := mockConn{
			CloseFunc: func() error {
				closeOnce.Do(func() {
					closeWG.Done()
				})
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
		var bb bytes.Buffer
		s := Socket{
			Conn: &conn,
			log:  log.New(&bb, "test", log.LstdFlags),
			Config: Config{
				TimeFunc: func() int64 { return 0 },
			},
		}
		switch {
		case test.inClosed:
			close(in)
		case test.pingTick:
			pingC <- time.Now()
		case test.httpPingTick:
			httpPingC <- time.Now()
		case test.wantOk, test.writeErr != nil, test.setWriteDeadlineErr != nil:
			in <- test.m
		}
		go s.writeMessagesSync(in, pingTicker, httpPingTicker)
		closeWG.Wait()
		switch {
		case !test.wantOk:
			if len(writtenMessages) != 0 {
				t.Errorf("Test %v: wanted no messages written to connection", i)
			}
		case !test.pingTick:
			gotM := <-writtenMessages
			switch {
			case !reflect.DeepEqual(test.wantM, gotM):
				t.Errorf("Test %v: messages not equal:\nwanted: %v\ngot:    %v", i, test.wantM, gotM)
			}
		}
	}
}
