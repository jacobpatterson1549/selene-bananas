package socket

import (
	"bytes"
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
		blockingChannel := make(chan struct{})
		defer close(blockingChannel)
		j := 0
		readJSONFunc := func(m *message.Message) error {
			<-blockingChannel
			mockConnReadMinimalMessage(m)
			return nil
		}
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			j++
			switch j {
			case 1:
				return &mockConn{
					RemoteAddrFunc: func() net.Addr {
						return mockAddr(socket1Addr)
					},
					ReadJSONFunc: readJSONFunc,
				}, nil
			case 2:
				return &mockConn{
					RemoteAddrFunc: func() net.Addr {
						return mockAddr(test.socket2Addr)
					},
					ReadJSONFunc: readJSONFunc,
				}, nil
			default:
				return nil, errors.New("too many calls to upgrade")
			}
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

func TestManagerHandleGameMessage(t *testing.T) {
	addr1 := mockAddr("addr1")
	addr2 := mockAddr("addr2")
	handleGameMessageTests := []struct {
		playerSockets   map[player.Name]map[net.Addr]chan<- message.Message
		playerGames     map[player.Name]map[game.ID]net.Addr
		m               message.Message
		wantPlayerGames map[player.Name]map[game.ID]net.Addr
		wantErr         bool
	}{
		{}, // no game on message
		{ // no game id on normal message
			m: message.Message{
				Type: message.TilesChange,
			},
		},
		{ // normal message
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
				},
				"barney": {
					addr2: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr1,
				},
				"barney": {
					2: addr2,
				},
			},
			m: message.Message{
				Type:       message.TilesChange, // new tile info omitted, but message should only be sent to fred
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr1,
				},
				"barney": {
					2: addr2,
				},
			},
		},
		{ // game infos
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
				},
				"barney": {
					addr2: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr1,
				},
				"barney": {
					1: addr2,
				},
			},
			m: message.Message{
				Type:  message.Infos,
				Games: []game.Info{},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr1,
				},
				"barney": {
					1: addr2,
				},
			},
		},
		{ // game infos, with playerName: only send to player for sockets not in a game (likely only one new socket)
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1:             make(chan<- message.Message, 1),
					addr2:             nil,
					mockAddr("addr3"): make(chan<- message.Message, 1), //  // unlikely, but possible if player has two sockets not in games.  Unless the message knows the new socket address.
				},
				"barney": {
					mockAddr("addr4"): nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr2,
				},
			},
			m: message.Message{
				Type:       message.Infos,
				PlayerName: "fred",
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr2,
				},
			},
		},
		{ // socketErr message from game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
					addr2: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
					2: addr2,
				},
			},
			m: message.Message{
				Type:       message.SocketError,
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
					2: addr2,
				},
			},
		},
		{ // socketErr message for player (which socket unknown)
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
					addr2: make(chan<- message.Message, 1),
				},
				"barney": {
					mockAddr("addr3"): nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
					2: addr2,
				},
			},
			m: message.Message{
				PlayerName: "fred",
				Type:       message.SocketError,
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
					2: addr2,
				},
			},
		},
		{ // game delete gets sent as a leave
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"barney": {
					addr1: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"barney": {
					2: addr1,
				},
			},
			m: message.Message{
				Type:       message.Leave,
				PlayerName: "barney",
				Info:       "the game was deleted, so player should leave it",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
		},
		{ // player not active in game, don't send message #1
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			playerGames:   map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
		},
		{ // player not active in game, don't send message #2
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
		},
	}
	for i, test := range handleGameMessageTests {
		var bb bytes.Buffer
		log := log.New(&bb, "test", log.LstdFlags)
		sm := Manager{
			playerSockets: test.playerSockets,
			playerGames:   test.playerGames,
			ManagerConfig: ManagerConfig{
				Log: log,
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		sm.handleGameMessage(ctx, test.m)
		switch {
		case test.wantErr:
			if bb.Len() == 0 {
				t.Errorf("Test %v: wanted error logged for bad message", i)
			}
		case !reflect.DeepEqual(test.wantPlayerGames, sm.playerGames):
			t.Errorf("Test %v: player games not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerGames, sm.playerGames)
		default:
			// ensure each non-nil socket has a message sent to it.  The test setup should only make buffered message channels if a message is expected to be sent on it.
			for pn, addrs := range sm.playerSockets {
				for addr, socketIn := range addrs {
					if socketIn != nil && len(socketIn) != 1 {
						t.Errorf("Test %v: wanted 1 message to be sent on socket for %v at %v", i, pn, addr)
					}
				}
			}
		}
	}
}

func TestManagerHandleSocketMessage(t *testing.T) {
	addr1 := mockAddr("addr1")
	addr2 := mockAddr("addr2")
	socketIns := []chan<- message.Message{
		make(chan<- message.Message, 1),
	}
	handleSocketMessageTests := []struct {
		playerSockets     map[player.Name]map[net.Addr]chan<- message.Message
		playerGames       map[player.Name]map[game.ID]net.Addr
		m                 message.Message
		wantPlayerSockets map[player.Name]map[net.Addr]chan<- message.Message
		wantPlayerGames   map[player.Name]map[game.ID]net.Addr
		want              message.Message
		wantOk            bool
		skipOutSend       bool
	}{
		{ // no playerName on message
		},
		{ // no address for message
			m: message.Message{
				PlayerName: "barney",
			},
		},
		{ // no socket for message
			m: message.Message{
				PlayerName: "barney",
				Addr:       addr1,
			},
		},
		{ // no player for message
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": nil,
			},
			m: message.Message{
				PlayerName: "barney",
			},
		},
		{ // no game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			m: message.Message{
				PlayerName: "fred",
				Addr:       addr1,
			},
		},
		{ // addr not in game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Addr:       addr2,
				Game: &game.Info{
					ID: 9,
				},
			},
		},
		{ // player not in game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
		},
		{ // player playing other game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
					addr2: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
				},
			},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 2,
				},
			},
		},
		{ // player playing game at different address
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
					addr2: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
					2: addr2,
				},
			},
			m: message.Message{
				Type:       message.Snag,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 2,
				},
			},
		},
		{ // create game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			m: message.Message{
				Type:       message.Create,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					Config: &game.Config{}, // this should be populated, but the gameManager checks this
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			wantOk: true,
		},
		{ // join game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: make(map[player.Name]map[game.ID]net.Addr),
			m: message.Message{
				Type:       message.Join,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			wantOk: true,
		},
		{ // join game that is already joined, NOOP
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			m: message.Message{
				Type:       message.Join,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			wantOk:      true,
			skipOutSend: true,
		},
		{ // join game from other socket, other socket should leave game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
					addr2: socketIns[0],
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr2,
				},
			},
			m: message.Message{
				Type:       message.Join,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
					addr2: socketIns[0],
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			wantOk: true,
		},
		{ // join game, switching games
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					7: addr1,
				},
			},
			m: message.Message{
				Type:       message.Join,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 8,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					8: addr1,
				},
			},
			wantOk: true,
		},
		{ // leave game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			m: message.Message{
				Type:       message.Leave,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
			wantOk:          true,
			skipOutSend:     true, // don't tell the game the socket is not listening
		},
		{ // player delete
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			m: message.Message{
				Type:       message.PlayerDelete,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			wantPlayerGames:   map[player.Name]map[game.ID]net.Addr{},
			wantOk:            true,
		},
	}
	for i, test := range handleSocketMessageTests {
		var bb bytes.Buffer
		log := log.New(&bb, "test", log.LstdFlags)
		sm := Manager{
			playerSockets: test.playerSockets,
			playerGames:   test.playerGames,
			ManagerConfig: ManagerConfig{
				Log: log,
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		gameOut := make(chan message.Message, 1)
		sm.handleSocketMessage(ctx, test.m, gameOut)
		switch {
		case !test.wantOk:
			if bb.Len() == 0 {
				t.Errorf("Test %v: wanted error logged for bad message", i)
			}
		case !reflect.DeepEqual(test.wantPlayerSockets, sm.playerSockets):
			t.Errorf("Test %v: player sockets not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerSockets, sm.playerSockets)
		case !reflect.DeepEqual(test.wantPlayerGames, sm.playerGames):
			t.Errorf("Test %v: player games not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerGames, sm.playerGames)
		default:
			switch len(gameOut) {
			case 0:
				if !test.skipOutSend {
					t.Errorf("Test %v: wanted message to be sent to game manager", i)
				}
			case 1:
				if test.skipOutSend {
					t.Errorf("Test %v: wanted no message to be sent to game manager", i)
					continue
				}
				got := <-gameOut
				if !reflect.DeepEqual(test.m, got) { // dumb check to ensure the messages is passed through without modification
					t.Errorf("Test %v: game messages not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
				}
			default:
				t.Errorf("too many messages sent on out channel")
			}
			for n, addrs := range test.playerSockets {
				for a, socketIn := range addrs {
					if socketIn != nil && len(socketIn) != 1 {
						t.Errorf("Test %v: wanted 1 message to be sent to %v at %v", i, n, a)
					}
				}
			}
		}
	}
}
