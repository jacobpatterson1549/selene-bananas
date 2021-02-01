package socket

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	upgradeFunc  func(w http.ResponseWriter, r *http.Request) (Conn, error)
	mockUpgrader struct {
		upgradeFunc
	}
)

func (u mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	return u.upgradeFunc(w, r)
}

func newSocketManager(maxSockets int, maxPlayerSockets int, uf upgradeFunc) *Manager {
	log := log.New(ioutil.Discard, "test", log.LstdFlags)
	socketCfg := Config{
		Log:                 log,
		ReadWait:            2 * time.Hour,
		WriteWait:           1 * time.Hour,
		PingPeriod:          1 * time.Hour, // these periods must be high to allow the test to be run safely with a high count
		ActivityCheckPeriod: 3 * time.Hour,
	}
	managerCfg := ManagerConfig{
		Log:              log,
		MaxSockets:       maxSockets,
		MaxPlayerSockets: maxPlayerSockets,
		SocketConfig:     socketCfg,
	}
	sm := Manager{
		upgrader: mockUpgrader{
			upgradeFunc: uf,
		},
		playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message),
		playerGames:   make(map[player.Name]map[game.ID]net.Addr),
		ManagerConfig: managerCfg,
	}
	return &sm
}

func mockConnection(playerName string) (player.Name, http.ResponseWriter, *http.Request) {
	pn := player.Name(playerName)
	var w http.ResponseWriter
	var r *http.Request
	return pn, w, r
}

func TestNewManager(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	newManagerTests := []struct {
		ManagerConfig ManagerConfig
		wantOk        bool
		want          *Manager
	}{
		{},
		{ // no log
			ManagerConfig: ManagerConfig{
				MaxSockets:       1,
				MaxPlayerSockets: 1,
			},
		},
		{ // low maxSockets
			ManagerConfig: ManagerConfig{
				Log:              testLog,
				MaxPlayerSockets: 1,
			},
		},
		{ // low maxPlayerSockets
			ManagerConfig: ManagerConfig{
				Log:        testLog,
				MaxSockets: 1,
			},
		},
		{ // maxSockets < maxPlayerSockets
			ManagerConfig: ManagerConfig{
				Log:              testLog,
				MaxSockets:       1,
				MaxPlayerSockets: 2,
			},
		},
		{
			ManagerConfig: ManagerConfig{
				Log:              testLog,
				MaxSockets:       10,
				MaxPlayerSockets: 3,
			},
			wantOk: true,
			want: &Manager{
				playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
				playerGames:   map[player.Name]map[game.ID]net.Addr{},
				ManagerConfig: ManagerConfig{
					Log:              testLog,
					MaxSockets:       10,
					MaxPlayerSockets: 3,
				},
			},
		},
	}
	for i, test := range newManagerTests {
		got, err := test.ManagerConfig.NewManager()
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case got.upgrader == nil:
			t.Errorf("Test %v: upgrader nil", i)
		default:
			got.upgrader = nil
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
			}
		}
	}
}

func TestRunManager(t *testing.T) {
	runManagerTests := []struct {
		alreadyRunning bool
		stopFunc       func(cancelFunc context.CancelFunc, in chan message.Message)
	}{
		{
			alreadyRunning: true,
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan message.Message) {
				cancelFunc()
			},
		},
		{
			stopFunc: func(cancelFunc context.CancelFunc, in chan message.Message) {
				close(in)
			},
		},
	}
	for i, test := range runManagerTests {
		var sm Manager
		if test.alreadyRunning {
			ctx := context.Background()
			ctx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			in := make(chan message.Message)
			_, err := sm.Run(ctx, in)
			if err != nil {
				t.Errorf("Test %v: unwanted error running socket manager: %v", i, err)
				continue
			}
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		in := make(chan message.Message)
		out, err := sm.Run(ctx, in)
		switch {
		case test.alreadyRunning:
			if err == nil {
				t.Errorf("Test %v: wanted error running socket manager that should already be running", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			if !sm.IsRunning() {
				t.Errorf("Test %v wanted socket manager to be running", i)
			}
			test.stopFunc(cancelFunc, in)
			_, ok := <-out
			if ok {
				t.Errorf("Test %v: wanted 'out' channel to be closed after 'in' channel was closed", i)
			}
			if sm.IsRunning() {
				t.Errorf("Test %v: wanted socket manager to not be running after it finished", i)
			}
			if _, err := sm.Run(ctx, in); err == nil {
				t.Errorf("Test %v: wanted error running socket manager after it is finished", i)
			}
		}
	}
}

func TestManagerAddSocket(t *testing.T) {
	managerAddSocketTests := []struct {
		running          bool
		maxSockets       int
		maxPlayerSockets int
		wantOk           bool
		upgradeErr       error
		Config
	}{
		{}, // not running
		{ // no room
			running:    true,
			maxSockets: 0,
		},
		{ // player quota reached
			running:          true,
			maxSockets:       1,
			maxPlayerSockets: 0,
		},
		{ // bad upgrade
			running:          true,
			maxSockets:       1,
			maxPlayerSockets: 1,
			upgradeErr:       errors.New("upgrade error"),
		},
		{ // bad socket config
			running:          true,
			maxSockets:       1,
			maxPlayerSockets: 1,
			Config:           Config{},
		},
		{ // ok
			running:          true,
			maxSockets:       1,
			maxPlayerSockets: 1,
			Config: Config{
				Log:                 log.New(ioutil.Discard, "scLog", log.LstdFlags),
				ReadWait:            2 * time.Hour,
				WriteWait:           1 * time.Hour,
				PingPeriod:          1 * time.Hour, // these periods must be high to allow the test to be run safely with a high count
				ActivityCheckPeriod: 3 * time.Hour,
			},
			wantOk: true,
		},
	}
	for i, test := range managerAddSocketTests {
		readBlocker1 := make(chan struct{})
		readBlocker2 := make(chan struct{})
		socketRun := false
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			if test.upgradeErr != nil {
				return nil, test.upgradeErr
			}
			return &mockConn{
				RemoteAddrFunc: func() net.Addr {
					return mockAddr("an.addr")
				},
				ReadJSONFunc: func(m *message.Message) error {
					<-readBlocker1
					if !socketRun {
						socketRun = true
						close(readBlocker2)
					}
					mockConnReadMinimalMessage(m)
					return nil
				},
			}, nil
		}
		sm := newSocketManager(test.maxSockets, test.maxPlayerSockets, upgradeFunc)
		if test.running {
			ctx := context.Background()
			in := make(chan message.Message)
			_, err := sm.Run(ctx, in)
			if err != nil {
				t.Errorf("Test %v: unwanted error running socket manager: %v", i, err)
				continue
			}
		}
		sm.SocketConfig = test.Config
		pn, w, r := mockConnection("selene")
		ctx := context.Background()
		err := sm.AddSocket(ctx, pn, w, r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error adding socket", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error adding socket: %v", i, err)
		case len(sm.playerSockets) != 1:
			t.Errorf("Test %v: wanted 1 player to have a socket, got %v", i, len(sm.playerSockets))
		case len(sm.playerSockets[pn]) != 1:
			t.Errorf("Test %v: wanted 1 socket for %v, got %v", i, pn, len(sm.playerSockets[pn]))
		default:
			// test if the socket is running by sending it a message.  this relies on sm.handleSocketMessage
			close(readBlocker1)
			<-readBlocker2
			if !socketRun {
				t.Errorf("Test %v: wanted socket to be run", i)
			}
		}
	}
}

func TestManagerAddSecondSocket(t *testing.T) {
	name1 := "fred"
	socket1Addr := "fred.pc"
	managerAddSecondSocketTests := []struct {
		maxSockets            int
		maxPlayerSockets      int
		name2                 string
		socket2Addr           string
		wantOk                bool
		wantNumPlayers        int
		wantNumPlayer2Sockets int
	}{
		{
			maxSockets:       1,
			maxPlayerSockets: 1,
			name2:            "barney",
		},
		{
			maxSockets:       2,
			maxPlayerSockets: 1,
			name2:            "fred",
		},
		{
			maxSockets:       2,
			maxPlayerSockets: 2,
			name2:            "fred",
			socket2Addr:      "fred.pc",
		},
		{
			maxSockets:            2,
			maxPlayerSockets:      2,
			name2:                 "fred",
			socket2Addr:           "fred.mac",
			wantOk:                true,
			wantNumPlayers:        1,
			wantNumPlayer2Sockets: 2,
		},
		{
			maxSockets:       2,
			maxPlayerSockets: 1,
			name2:            "barney",
			socket2Addr:      "fred.pc",
		},
		{
			maxSockets:            2,
			maxPlayerSockets:      1,
			name2:                 "barney",
			socket2Addr:           "barney.pc",
			wantOk:                true,
			wantNumPlayers:        2,
			wantNumPlayer2Sockets: 1,
		},
	}
	for i, test := range managerAddSecondSocketTests {
		secondSocketAddr := false
		blockingChannel := make(chan struct{})
		defer close(blockingChannel)
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			return &mockConn{
				RemoteAddrFunc: func() net.Addr {
					if secondSocketAddr {
						return mockAddr(test.socket2Addr)
					}
					secondSocketAddr = true
					return mockAddr(socket1Addr)
				},
				ReadJSONFunc: func(m *message.Message) error {
					<-blockingChannel
					mockConnReadMinimalMessage(m)
					return nil
				},
			}, nil
		}
		sm := newSocketManager(test.maxSockets, test.maxPlayerSockets, upgradeFunc)
		ctx := context.Background()
		in := make(chan message.Message)
		_, err := sm.Run(ctx, in)
		if err != nil {
			t.Errorf("Test %v: unwanted error running socket manager: %v", i, err)
			continue
		}
		pn1, w1, r1 := mockConnection(name1)
		err1 := sm.AddSocket(ctx, pn1, w1, r1)
		if err1 != nil {
			t.Errorf("Test %v: unwanted error adding first socket: %v", i, err1)
			continue
		}
		pn2, w2, r2 := mockConnection(test.name2)
		err2 := sm.AddSocket(ctx, pn2, w2, r2)
		switch {
		case !test.wantOk:
			if err2 == nil {
				t.Errorf("Test %v: wanted error adding second socket", i)
			}
		case err2 != nil:
			t.Errorf("Test %v: unwanted error adding second socket: %v", i, err2)
		case len(sm.playerSockets) != test.wantNumPlayers:
			t.Errorf("Test %v: wanted %v players to have a socket, got %v", i, test.wantNumPlayers, len(sm.playerSockets))
		case len(sm.playerSockets[pn2]) != test.wantNumPlayer2Sockets:
			t.Errorf("Test %v: wanted %v socket for %v, got %v", i, test.wantNumPlayer2Sockets, pn2, len(sm.playerSockets[pn2]))
		}
	}
}

func TestAddSocketIsRun(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	sm := newSocketManager(1, 1, uf)
	ctx := context.Background()
	in := make(chan message.Message)
	_, err := sm.Run(ctx, in)
	if err != nil {
		t.Fatalf("unwanted error running socket manager: %v", err)
	}
	// ensure socket run started
}

func TestManagerHandleGameMessage(t *testing.T) {
	// TODO: test message recieved from game 'in' channel
	// * normal message
	// * messageType.Infos (send to all sockets)
	// * socketError message: with and without game id
	// * message without game[Id]: do not send, only log
	// * game leave, delete message:
	// * player delete message -> should close socket input streams
}

func TestManagerHandleSocketMessage(t *testing.T) {
	// TODO: test message recieved from a socket
	// join game add to game until
	// leave game
}
