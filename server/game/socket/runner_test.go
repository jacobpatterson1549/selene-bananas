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
	"sync"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

func TestNewRunner(t *testing.T) {
	testLog := log.New(ioutil.Discard, "", 0)
	newRunnerTests := []struct {
		log          *log.Logger
		RunnerConfig RunnerConfig
		wantOk       bool
		want         *Runner
	}{
		{},
		{ // no log
			RunnerConfig: RunnerConfig{
				MaxSockets:       1,
				MaxPlayerSockets: 1,
			},
		},
		{ // low maxSockets
			log: testLog,
			RunnerConfig: RunnerConfig{
				MaxPlayerSockets: 1,
			},
		},
		{ // low maxPlayerSockets
			log: testLog,
			RunnerConfig: RunnerConfig{
				MaxSockets: 1,
			},
		},
		{ // maxSockets < maxPlayerSockets
			log: testLog,
			RunnerConfig: RunnerConfig{
				MaxSockets:       1,
				MaxPlayerSockets: 2,
			},
		},
		{
			log: testLog,
			RunnerConfig: RunnerConfig{
				MaxSockets:       10,
				MaxPlayerSockets: 3,
			},
			wantOk: true,
			want: &Runner{
				log:           testLog,
				playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
				playerGames:   map[player.Name]map[game.ID]net.Addr{},
				RunnerConfig: RunnerConfig{
					MaxSockets:       10,
					MaxPlayerSockets: 3,
				},
			},
		},
	}
	for i, test := range newRunnerTests {
		got, err := test.RunnerConfig.NewRunner(test.log)
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

func TestRunRunner(t *testing.T) {
	runRunnerTests := []struct {
		stopFunc func(cancelFunc context.CancelFunc, in chan message.Message)
	}{
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
	for i, test := range runRunnerTests {
		r := Runner{
			log: log.New(ioutil.Discard, "", 0),
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		var wg sync.WaitGroup
		in := make(chan message.Message)
		runnerOut := r.Run(ctx, &wg, in)
		test.stopFunc(cancelFunc, in)
		wg.Wait()
		_, runnerOutOpen := <-runnerOut
		if runnerOutOpen {
			t.Errorf("Test %v: wanted runner 'out' channel to be closed after 'in' channel was closed", i)
		}
	}
}

// TestRunRunnerHandleLobbyMessage ensures a basic yet invalid lobby message passes throgh the runner correctly.
func TestRunRunnerHandleLobbyMessage(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	in := make(chan message.Message)
	var bb bytes.Buffer
	log := log.New(&bb, "", 0)
	r := Runner{
		log: log,
	}
	r.Run(ctx, &wg, in)
	m := message.Message{}
	in <- m
	cancelFunc()
	wg.Wait()
	if bb.Len() == 0 {
		t.Errorf("wanted error to be logged when loby sent socket runner invalid message")
	}
}

// TestRunRunnerHandleSocketMessage ensures a basic, yet invalid socket messages passes throgh the runner correctly.
func TestRunRunnerHandleSocketMessage(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	in := make(chan message.Message)
	var bb bytes.Buffer
	log := log.New(&bb, "", 0)
	r := Runner{
		log: log,
	}
	r.Run(ctx, &wg, in)
	m := message.Message{}
	r.socketOut <- m
	cancelFunc()
	wg.Wait()
	if bb.Len() == 0 {
		t.Errorf("wanted error to be logged when loby sent socket runner invalid message")
	}
}

func TestRunnerHandleAddSocketCheckResult(t *testing.T) {
	runnerAddSocketTests := []struct {
		m      message.Message
		wantOk bool
	}{
		{}, // no playerName in message
		{
			m: message.Message{
				PlayerName: "selene",
			},
			wantOk: true,
		},
	}
	for i, test := range runnerAddSocketTests {
		addr := mockAddr("selen.pc")
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			return &mockConn{
				RemoteAddrFunc: func() net.Addr {
					return addr
				},
				SetReadDeadlineFunc: func(t time.Time) error {
					return errors.New("stop run for test")
				},
				ReadMessageFunc: func(m *message.Message) error {
					return nil
				},
				WriteCloseFunc: func(reason string) error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			}, nil
		}
		runnerConfig := RunnerConfig{
			MaxSockets:       1,
			MaxPlayerSockets: 1,
			SocketConfig: Config{
				TimeFunc:       func() int64 { return 0 },
				ReadWait:       2 * time.Hour,
				WriteWait:      1 * time.Hour,
				PingPeriod:     2 * time.Hour,
				HTTPPingPeriod: 3 * time.Hour,
			},
		}
		r := Runner{
			log:           log.New(ioutil.Discard, "", 0),
			upgrader:      mockUpgrader(upgradeFunc),
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			RunnerConfig:  runnerConfig,
			socketOut:     make(chan message.Message, 1), // the socket will run and fail, posting a message here
		}
		ctx := context.Background()
		result := make(chan message.Message, 1)
		test.m.Type = message.SocketAdd
		test.m.AddSocketRequest = &message.AddSocketRequest{
			Result: result,
		}
		var wg sync.WaitGroup
		r.handleLobbyMessage(ctx, &wg, test.m)
		got := <-result
		switch {
		case !test.wantOk:
			if got.Type != message.SocketError {
				t.Errorf("Test %v: wanted message to be socket error, got %v", i, got)
			}
		default:
			switch {
			case got.Type != message.GameInfos:
				t.Errorf("Test %v: wanted message to be for game infos, got %v", i, got)
			case addr != got.Addr:
				t.Errorf("Test %v: wanted message addr to be %v, got %v", i, addr, got.Addr)
			case test.m.PlayerName != got.PlayerName:
				t.Errorf("Test %v: wanted player name in message to be %v, got %v", i, test.m.PlayerName, got.PlayerName)
			case len(r.playerSockets) != 1:
				t.Errorf("Test %v: wanted new socket to be added to runner", i)
			}
		}
	}
}

func TestRunnerHandleAddSocket(t *testing.T) {
	runnerAddSocketTests := []struct {
		maxSockets       int
		maxPlayerSockets int
		playerName       string
		wantOk           bool
		upgradeErr       error
		Config
	}{
		{}, // no room
		{ // player quota reached
			maxSockets:       1,
			maxPlayerSockets: 0,
		},
		{ // no playerName
			maxSockets:       1,
			maxPlayerSockets: 1,
		},
		{ // bad upgrade
			maxSockets:       1,
			maxPlayerSockets: 1,
			playerName:       "selene",
			upgradeErr:       errors.New("upgrade error"),
		},
		{ // bad socket config
			maxSockets:       1,
			maxPlayerSockets: 1,
			playerName:       "selene",
			Config:           Config{},
		},
		{ // ok
			maxSockets:       1,
			maxPlayerSockets: 1,
			playerName:       "selene",
			Config: Config{
				TimeFunc:       func() int64 { return 0 },
				ReadWait:       2 * time.Hour,
				WriteWait:      1 * time.Hour,
				PingPeriod:     2 * time.Hour,
				HTTPPingPeriod: 3 * time.Hour,
			},
			wantOk: true,
		},
	}
	for i, test := range runnerAddSocketTests {
		socketRun := false
		addr := mockAddr("an.addr")
		var wg sync.WaitGroup
		if test.wantOk {
			wg.Add(1) // ensure SetReadDeadline is called
		}
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			if test.upgradeErr != nil {
				return nil, test.upgradeErr
			}
			return &mockConn{
				RemoteAddrFunc: func() net.Addr {
					return addr
				},
				SetReadDeadlineFunc: func(t time.Time) error {
					socketRun = true
					wg.Done()
					return errors.New("stop run for test")
				},
				WriteCloseFunc: func(reason string) error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			}, nil
		}
		runnerConfig := RunnerConfig{
			MaxSockets:       test.maxSockets,
			MaxPlayerSockets: test.maxPlayerSockets,
			SocketConfig:     test.Config,
		}
		r := Runner{
			log:           log.New(ioutil.Discard, "", 0),
			upgrader:      mockUpgrader(upgradeFunc),
			playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message),
			playerGames:   make(map[player.Name]map[game.ID]net.Addr),
			RunnerConfig:  runnerConfig,
			socketOut:     make(chan message.Message, 1), // the socket will run and fail, posting a message here
		}
		pn, w, req := mockAddUserRequest(test.playerName)
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		s, err := r.handleAddSocket(ctx, &wg, pn, w, req)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error adding socket", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error adding socket: %v", i, err)
		case addr != s.Addr:
			t.Errorf("Test %v: wanted addr to be %v, got %v", i, addr, s.Addr)
		case len(r.playerSockets) != 1:
			t.Errorf("Test %v: wanted 1 player to have a socket, got %v", i, len(r.playerSockets))
		case len(r.playerSockets[pn]) != 1:
			t.Errorf("Test %v: wanted 1 socket for %v, got %v", i, pn, len(r.playerSockets[pn]))
		}
		cancelFunc()
		wg.Wait()
		if test.wantOk && !socketRun {
			t.Errorf("Test %v: wanted socket to be run", i)
		}
	}
}

func TestRunnerHandleAddSocketSecond(t *testing.T) {
	runnerAddSecondSocketTests := []struct {
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
	for i, test := range runnerAddSecondSocketTests {
		name1 := "fred"
		socket1Addr := "fred.pc"
		j := 0
		upgradeFunc := func(w http.ResponseWriter, r *http.Request) (Conn, error) {
			j++
			var addr net.Addr
			switch j {
			case 1:
				addr = mockAddr(socket1Addr)
			case 2:
				addr = mockAddr(test.socket2Addr)
			default:
				return nil, errors.New("too many calls to upgrade")
			}
			return &mockConn{
				RemoteAddrFunc: func() net.Addr {
					return addr
				},
				SetReadDeadlineFunc: func(t time.Time) error {
					return errors.New("stop run for test")
				},
				WriteCloseFunc: func(reason string) error {
					return nil
				},
				CloseFunc: func() error {
					return nil
				},
			}, nil
		}
		runnerConfig := RunnerConfig{
			MaxSockets:       test.maxSockets,
			MaxPlayerSockets: test.maxPlayerSockets,
			SocketConfig: Config{
				TimeFunc:       func() int64 { return 0 },
				ReadWait:       2 * time.Hour,
				WriteWait:      1 * time.Hour,
				PingPeriod:     2 * time.Hour,
				HTTPPingPeriod: 3 * time.Hour,
			},
		}
		r := Runner{
			log:           log.New(ioutil.Discard, "", 0),
			upgrader:      mockUpgrader(upgradeFunc),
			playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message),
			playerGames:   make(map[player.Name]map[game.ID]net.Addr),
			RunnerConfig:  runnerConfig,
			socketOut:     make(chan message.Message, 2), // the sockets will run and fail, posting a message here
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		pn1, w1, r1 := mockAddUserRequest(name1)
		_, err1 := r.handleAddSocket(ctx, &wg, pn1, w1, r1)
		switch {
		case err1 != nil:
			t.Errorf("Test %v: unwanted error adding first socket: %v", i, err1)
		default:
			pn2, w2, r2 := mockAddUserRequest(test.name2)
			_, err2 := r.handleAddSocket(ctx, &wg, pn2, w2, r2)
			switch {
			case !test.wantOk:
				if err2 == nil {
					t.Errorf("Test %v: wanted error adding second socket", i)
				}
			case err2 != nil:
				t.Errorf("Test %v: unwanted error adding second socket: %v", i, err2)
			case len(r.playerSockets) != test.wantNumPlayers:
				t.Errorf("Test %v: wanted %v players to have a socket, got %v", i, test.wantNumPlayers, len(r.playerSockets))
			case len(r.playerSockets[pn2]) != test.wantNumPlayer2Sockets:
				t.Errorf("Test %v: wanted %v socket for %v, got %v", i, test.wantNumPlayer2Sockets, pn2, len(r.playerSockets[pn2]))
			}
		}
		cancelFunc()
		wg.Wait()
	}
}

func TestRunnerHandleLobbyMessage(t *testing.T) {
	addr1 := mockAddr("addr1")
	addr2 := mockAddr("addr2")
	handleLobbyMessageTests := []struct {
		playerSockets   map[player.Name]map[net.Addr]chan<- message.Message
		playerGames     map[player.Name]map[game.ID]net.Addr
		m               message.Message
		wantPlayerGames map[player.Name]map[game.ID]net.Addr
		wantErr         bool
	}{
		{}, // no game on message
		{ // no game id on normal message
			m: message.Message{
				Type: message.ChangeGameTiles,
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
				Type:       message.ChangeGameTiles, // new tile info omitted, but message should only be sent to fred
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
				Type:  message.GameInfos,
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
		{ // game infos, with addr: only send to player for the socket
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1:             make(chan<- message.Message, 1),
					addr2:             nil,
					mockAddr("addr3"): nil,
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
				Type:       message.GameInfos,
				PlayerName: "fred",
				Addr:       addr1,
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					2: addr2,
				},
			},
		},
		{ // game infos, with addr: only send to player for the socket, but no player socket exists
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			m: message.Message{
				Type:       message.GameInfos,
				PlayerName: "fred",
				Addr:       addr1,
			},
			wantErr: true,
		},
		{ // game infos, with addr: only send to player for the socket, but no socket exists
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {},
			},
			m: message.Message{
				Type:       message.GameInfos,
				PlayerName: "fred",
				Addr:       addr1,
			},
			wantErr: true,
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
				Type:       message.LeaveGame,
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
				Type:       message.ChangeGameTiles,
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
				Type:       message.ChangeGameTiles,
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
		},
		{ // player not active in game, don't send message #3
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
				},
			},
			m: message.Message{
				Type:       message.ChangeGameTiles,
				PlayerName: "fred",
				Game: &game.Info{
					ID: 2,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: addr1,
				},
			},
		},
		{ // AddSocket no AddSocketRequest
			m: message.Message{
				Type: message.SocketAdd,
			},
			wantErr: true,
		},
		{ // AddSocket no AddSocketRquest.Result
			m: message.Message{
				Type:             message.SocketAdd,
				AddSocketRequest: &message.AddSocketRequest{},
			},
			wantErr: true,
		},
		{ // player join game.  The lobby sends this when a player creates a game.
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"barney": {
					addr1: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.JoinGame,
				PlayerName: "barney",
				Game: &game.Info{
					ID: 3,
				},
				Addr: addr1,
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"barney": {
					3: addr1,
				},
			},
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
				Type:       message.JoinGame,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
		},
		{ // join game from other socket, other socket should leave game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
					addr2: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr2,
				},
			},
			m: message.Message{
				Type:       message.JoinGame,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
		},
		{ // join game, switching games
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					7: addr1,
				},
			},
			m: message.Message{
				Type:       message.JoinGame,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 8,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					8: addr1,
				},
			},
		},
		{ // leave game, when not allowed in game.  This causes the ui to kick the player even though he was never added in the server
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message, 1),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.LeaveGame,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 1,
				},
			},
			wantPlayerGames: map[player.Name]map[game.ID]net.Addr{},
		},
	}
	for i, test := range handleLobbyMessageTests {
		var bb bytes.Buffer
		log := log.New(&bb, "", 0)
		r := Runner{
			log:           log,
			playerSockets: test.playerSockets,
			playerGames:   test.playerGames,
		}
		ctx := context.Background()
		var wg sync.WaitGroup
		r.handleLobbyMessage(ctx, &wg, test.m)
		switch {
		case test.wantErr:
			if bb.Len() == 0 {
				t.Errorf("Test %v: wanted error logged for bad message", i)
			}
		case !reflect.DeepEqual(test.wantPlayerGames, r.playerGames):
			t.Errorf("Test %v: player games not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerGames, r.playerGames)
		default:
			// ensure each non-nil socket has a message sent to it.  The test setup should only make buffered message channels if a message is expected to be sent on it.
			for pn, addrs := range r.playerSockets {
				for addr, socketIn := range addrs {
					if socketIn != nil && len(socketIn) != 1 {
						t.Errorf("Test %v: wanted 1 message to be sent on socket for %v at %v", i, pn, addr)
					}
				}
			}
		}
	}
}

func TestRunnerHandleSocketMessage(t *testing.T) {
	addr1 := mockAddr("addr1")
	addr2 := mockAddr("addr2")
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
				Type:       message.SnagGameTile,
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
				Type:       message.SnagGameTile,
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
				Type:       message.SnagGameTile,
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
				Type:       message.SnagGameTile,
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
				Type:       message.CreateGame,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					Config: &game.Config{}, // this should be populated, but the gameRunner checks this
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
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
				Type:       message.LeaveGame,
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
		{ // leave game when player not in any game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: nil,
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.LeaveGame,
				PlayerName: "fred",
				Addr:       addr2,
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
		},
		{ // socket close
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					9: addr1,
				},
			},
			m: message.Message{
				Type:       message.SocketClose,
				PlayerName: "fred",
				Addr:       addr1,
				Game: &game.Info{
					ID: 9,
				},
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			wantPlayerGames:   map[player.Name]map[game.ID]net.Addr{},
			wantOk:            true,
			skipOutSend:       true,
		},
		{ // socket close when not in any game
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {
					addr1: make(chan<- message.Message),
				},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{},
			m: message.Message{
				Type:       message.SocketClose,
				PlayerName: "fred",
				Addr:       addr1,
			},
			wantPlayerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			wantPlayerGames:   map[player.Name]map[game.ID]net.Addr{},
			wantOk:            true,
			skipOutSend:       true,
		},
	}
	for i, test := range handleSocketMessageTests {
		var bb bytes.Buffer
		log := log.New(&bb, "", 0)
		r := Runner{
			log:           log,
			playerSockets: test.playerSockets,
			playerGames:   test.playerGames,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		gameOut := make(chan message.Message, 1)
		r.handleSocketMessage(ctx, test.m, gameOut)
		switch {
		case !test.wantOk:
			if bb.Len() == 0 {
				t.Errorf("Test %v: wanted error logged for bad message", i)
			}
		case !reflect.DeepEqual(test.wantPlayerSockets, r.playerSockets):
			t.Errorf("Test %v: player sockets not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerSockets, r.playerSockets)
		case !reflect.DeepEqual(test.wantPlayerGames, r.playerGames):
			t.Errorf("Test %v: player games not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayerGames, r.playerGames)
		default:
			switch len(gameOut) {
			case 0:
				if !test.skipOutSend {
					t.Errorf("Test %v: wanted message to be sent to game runner", i)
				}
			case 1:
				if test.skipOutSend {
					t.Errorf("Test %v: wanted no message to be sent to game runner", i)
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

// TestRunnerHandleLobbyMessagePlayerRemove verifies the sockets are closed.
func TestRunnerHandleLobbyMessagePlayerRemove(t *testing.T) {
	c1 := make(chan message.Message)
	c2 := make(chan message.Message, 1)
	c3 := make(chan message.Message)
	addr1 := mockAddr("addr1")
	addr2 := mockAddr("addr2")
	addr3 := mockAddr("addr3")
	// player delete is sent from the lobby when the player is actually deleted
	playerSockets := map[player.Name]map[net.Addr]chan<- message.Message{
		"fred": {
			addr1: c1,
			addr3: c3,
		},
		"barney": {
			addr2: c2,
		},
	}
	playerGames := map[player.Name]map[game.ID]net.Addr{
		"fred": {
			1: addr1,
		},
		"barney": {
			1: addr2,
		},
	}
	r := Runner{
		playerSockets: playerSockets,
		playerGames:   playerGames,
	}
	m := message.Message{
		Type:       message.PlayerRemove,
		PlayerName: "fred",
	}
	wantPlayerSockets := map[player.Name]map[net.Addr]chan<- message.Message{
		"barney": {
			addr2: c2,
		},
	}
	wantPlayerGames := map[player.Name]map[game.ID]net.Addr{
		"barney": {
			1: addr2,
		},
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	r.handleLobbyMessage(ctx, &wg, m)
	switch {
	case !reflect.DeepEqual(wantPlayerSockets, r.playerSockets):
		t.Errorf("player sockets not equal:\nwanted: %v\ngot:    %v", wantPlayerSockets, r.playerSockets)
	case !reflect.DeepEqual(wantPlayerGames, r.playerGames):
		t.Errorf("player games not equal:\nwanted: %v\ngot:    %v", wantPlayerGames, r.playerGames)
	default:
		// ensure the sockets are closed or left open
		<-c1
		<-c3
		c2 <- message.Message{Info: "ok"}
		m := <-c2
		if m.Info != "ok" {
			t.Errorf("wansted c2 to be left open")
		}
	}
}

// TestSendMessageForGameBadRunnerState adds coverage for some scenarios where playerGames do do not have matching playerSocket entries
func TestSendMessageForGameBadRunnerState(t *testing.T) {
	tests := []struct {
		playerSockets map[player.Name]map[net.Addr]chan<- message.Message
		playerGames   map[player.Name]map[game.ID]net.Addr
	}{
		{
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: mockAddr("addr1"),
				},
			},
		},
		{
			playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
				"fred": {},
			},
			playerGames: map[player.Name]map[game.ID]net.Addr{
				"fred": {
					1: mockAddr("addr1"),
				},
			},
		},
	}
	for i, test := range tests {
		var bb bytes.Buffer
		log := log.New(&bb, "", 0)
		r := Runner{
			log:           log,
			playerSockets: test.playerSockets,
			playerGames:   test.playerGames,
		}
		ctx := context.Background()
		m := message.Message{
			Game: &game.Info{
				ID: 1,
			},
			PlayerName: "fred",
		}
		r.sendMessageForGame(ctx, m)
		if bb.Len() == 0 {
			t.Errorf("Test %v: wanted error logged for bad runner state", i)
		}
	}
}

func TestRemoveSocket(t *testing.T) {
	socketIn := make(chan message.Message)
	pn := player.Name("fred")
	addr := mockAddr("fred.pc")
	r := Runner{
		playerSockets: map[player.Name]map[net.Addr]chan<- message.Message{
			pn: {
				addr: socketIn,
			},
		},
		playerGames: map[player.Name]map[game.ID]net.Addr{
			pn: {
				1: addr,
			},
		},
	}
	ctx := context.Background()
	m := message.Message{
		PlayerName: pn,
		Addr:       addr,
	}
	r.removeSocket(ctx, m)
	<-socketIn // removing a socket should close it's in channel
	switch {
	case len(r.playerSockets) != 0:
		t.Errorf("wanted player socket to be removed")
	case len(r.playerGames) != 0:
		t.Errorf("wanted player game to be removed")
	}
}
