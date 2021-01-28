package socket

import (
	"fmt"
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
		upgradeFunc upgradeFunc
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

func TestAddSocket(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	m := newSocketManager(1, 1, uf)
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	n := m.numSockets()
	switch {
	case err != nil:
		t.Errorf("problem adding socket: %v", err)
	case n != 1:
		t.Errorf("wanted 1 socket, got %v", n)
	}
}

func TestAddSocketBadConnUpgrade(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return nil, fmt.Errorf("upgrade error")
	}
	m := newSocketManager(1, 1, uf)
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err == nil {
		t.Error("wanted error creating socket with bad request")
	}
}

func TestAddSocketBadSocketConfig(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	m := newSocketManager(1, 1, uf)
	m.ManagerConfig.SocketConfig.TimeFunc = nil
	pn, w, r := mockConnection("selene")
	err := m.AddSocket(pn, w, r)
	if err == nil {
		t.Error("wanted error creating socket with bad config")
	}
}

func TestAddSocketMax(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	sm := newSocketManager(1, 1, uf)
	pn, w, r := mockConnection("selene")
	err := sm.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn, w, r = mockConnection("selene")
	err = sm.AddSocket(pn, w, r) // only one socket allowed
	if err == nil {
		t.Errorf("wanted error adding socket when maxSockets reached")
	}
}

func TestAddSocketTwo(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	sm := newSocketManager(2, 2, uf)
	pn1, w, r := mockConnection("fred")
	err := sm.AddSocket(pn1, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn2, w, r := mockConnection("barney")
	err = sm.AddSocket(pn2, w, r)
	switch {
	case err != nil:
		t.Errorf("problem adding socket: %v", err)
	case len(sm.playerSockets) != 2:
		t.Errorf("wanted 2 players to have a socket, got %v", len(sm.playerSockets))
	case len(sm.playerSockets[pn1]) != 1:
		t.Errorf("wanted 1 socket for %v, got %v", pn1, len(sm.playerSockets[pn1]))
	case len(sm.playerSockets[pn2]) != 1:
		t.Errorf("wanted 1 socket for %v, got %v", pn2, len(sm.playerSockets[pn2]))
	}
}

func TestAddSocketTwoSamePlayer(t *testing.T) {
	uf := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
		return &mockConn{}, nil
	}
	sm := newSocketManager(2, 1, uf)
	pn, w, r := mockConnection("fred")
	err := sm.AddSocket(pn, w, r)
	if err != nil {
		t.Fatalf("unwanted error adding socket: %v", err)
	}
	pn, w, r = mockConnection("fred")
	err = sm.AddSocket(pn, w, r)
	if err == nil {
		t.Errorf("wanted error adding socket for same player over maxPlayerSockets")
	}
}

// func TestJoinGame(t *testing.T) {
// 	m := newSocketManager(1, 1)
// 	s := mockSocket()
// }
