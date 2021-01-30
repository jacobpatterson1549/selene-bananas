package socket

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
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
	timeFunc := func() int64 { return 21 }
	socketCfg := Config{
		Log:            log,
		TimeFunc:       timeFunc,
		ReadWait:       2 * time.Second,
		WriteWait:      1 * time.Second,
		PingPeriod:     1 * time.Second,
		IdlePeriod:     3 * time.Minute,
		HTTPPingPeriod: 1 * time.Minute,
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
		playerSockets: make(map[player.Name][]Socket, managerCfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]Socket),
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
		want          Manager
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
			want: Manager{
				playerSockets: map[player.Name][]Socket{},
				playerGames:   map[player.Name]map[game.ID]Socket{},
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
			if !reflect.DeepEqual(test.want, *got) {
				t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, *got)
			}
		}
	}
}

func TestManagerAddSocket(t *testing.T) {
	managerAddSocketTests := []struct {
		maxSockets       int
		maxPlayerSockets int
		wantOk           bool
		upgradeFunc
		Config
	}{
		{ // no room
			maxSockets: 0,
		},
		{ // player quota reached
			maxSockets:       1,
			maxPlayerSockets: 0,
		},
		{ // bad upgrade
			maxSockets:       1,
			maxPlayerSockets: 1,
			upgradeFunc: func(w http.ResponseWriter, r *http.Request) (Conn, error) {
				return nil, errors.New("upgrade error")
			},
		},
		{ // bad socket config
			maxSockets:       1,
			maxPlayerSockets: 1,
			upgradeFunc: func(w http.ResponseWriter, r *http.Request) (Conn, error) {
				return &mockConn{}, nil
			},
			Config: Config{},
		},
		{ // ok
			maxSockets:       1,
			maxPlayerSockets: 1,
			upgradeFunc: func(w http.ResponseWriter, r *http.Request) (Conn, error) {
				return &mockConn{}, nil
			},
			Config: Config{
				Log:            log.New(ioutil.Discard, "scLog", log.LstdFlags),
				TimeFunc:       func() int64 { return 22 },
				ReadWait:       2 * time.Second,
				WriteWait:      1 * time.Second,
				PingPeriod:     1 * time.Second,
				IdlePeriod:     3 * time.Minute,
				HTTPPingPeriod: 1 * time.Minute,
			},
			wantOk: true,
		},
	}
	for i, test := range managerAddSocketTests {
		sm := newSocketManager(test.maxSockets, test.maxPlayerSockets, test.upgradeFunc)
		sm.SocketConfig = test.Config
		pn, w, r := mockConnection("selene")
		err := sm.AddSocket(pn, w, r)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error adding socket", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error adding socket: %v", i, err)
		case len(sm.playerSockets) != 1:
			t.Errorf("wanted 1 player to have a socket, got %v", len(sm.playerSockets))
		case len(sm.playerSockets[pn]) != 1:
			t.Errorf("wanted 1 socket for %v, got %v", pn, len(sm.playerSockets[pn]))
		}
	}
}

func TestManagerAddSecondSocket(t *testing.T) {
	name1 := "fred"
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	managerAddSecondSocketTests := []struct {
		maxSockets       int
		maxPlayerSockets int
		name2            string
		wantOk           bool
	}{
		{
			maxSockets:       1,
			maxPlayerSockets: 1,
			name2:            "barney",
		},
		{
			maxSockets:       2,
			maxPlayerSockets: 1,
			name2:            "barney",
			wantOk:           true,
		},
		{
			maxSockets:       2,
			maxPlayerSockets: 1,
			name2:            "fred",
		},
	}
	for i, test := range managerAddSecondSocketTests {
		sm := newSocketManager(test.maxSockets, test.maxPlayerSockets, uf)
		pn1, w1, r1 := mockConnection(name1)
		err1 := sm.AddSocket(pn1, w1, r1)
		if err1 != nil {
			t.Errorf("Test %v: unwanted error adding first socket: %v", i, err1)
			continue
		}
		pn2, w2, r2 := mockConnection(test.name2)
		err2 := sm.AddSocket(pn2, w2, r2)
		switch {
		case !test.wantOk:
			if err2 == nil {
				t.Errorf("Test %v: wanted error adding second socket", i)
			}
		case err2 != nil:
			t.Errorf("Test %v: unwanted error adding second socket: %v", i, err2)
		case len(sm.playerSockets) != 2:
			t.Errorf("wanted 2 players to have a socket, got %v", len(sm.playerSockets))
		case len(sm.playerSockets[pn2]) != 1:
			t.Errorf("wanted 1 socket for %v, got %v", pn2, len(sm.playerSockets[pn2]))
		}
	}
}

func TestRunManager(t *testing.T) {
	// TODO: ensure manager closes
}

func TestManagerHandleGameMessage(t *testing.T) {
	// TODO: test message recieved from game 'in' channel
	// * normal message
	// * messageType.Infos (send to all sockets)
	// * socketError message: with and without game id
	// * message without game[Id]: do not send, only log
	// * game leave, delete message:
	// * player delete message
}

func TestManagerHandleSocketMessage(t *testing.T) {
	// TODO: test message recieved from a socket
	// join game add to game until
	// leave game
}
