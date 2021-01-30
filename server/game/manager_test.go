package game

import (
	"context"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestNewManager(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	newManagerTests := []struct {
		ManagerConfig ManagerConfig
		wantOk        bool
		want          *Manager
	}{
		{}, // no log
		{ // low MaxGames
			ManagerConfig: ManagerConfig{
				Log: testLog,
			},
		},
		{
			ManagerConfig: ManagerConfig{
				Log:      testLog,
				MaxGames: 10,
			},
			wantOk: true,
			want: &Manager{
				games: map[game.ID]chan<- message.Message{},
				ManagerConfig: ManagerConfig{
					Log:      testLog,
					MaxGames: 10,
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
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
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
		var gm Manager
		if test.alreadyRunning {
			ctx := context.Background()
			ctx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			in := make(chan message.Message)
			_, err := gm.Run(ctx, in)
			if err != nil {
				t.Errorf("Test %v: unwanted error running socket manager: %v", i, err)
				continue
			}
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		in := make(chan message.Message)
		out, err := gm.Run(ctx, in)
		switch {
		case test.alreadyRunning:
			if err == nil {
				t.Errorf("Test %v: wanted error running socket manager that should already be running", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		default:
			if !gm.IsRunning() {
				t.Errorf("Test %v wanted socket manager to be running", i)
			}
			test.stopFunc(cancelFunc, in)
			_, ok := <-out
			if ok {
				t.Errorf("Test %v: wanted 'out' channel to be closed after 'in' channel was closed", i)
			}
			if gm.IsRunning() {
				t.Errorf("Test %v: wanted socket manager to not be running after it finished", i)
			}
			if _, err := gm.Run(ctx, in); err == nil {
				t.Errorf("Test %v: wanted error running socket manager after it is finished", i)
			}
		}
	}
}

func TestGameCreate(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	gameCreateTests := []struct {
		ManagerConfig ManagerConfig
		wantOk        bool
	}{
		{ // happy path
			ManagerConfig: ManagerConfig{
				Log:      testLog,
				MaxGames: 1,
				GameConfig: Config{
					Log:                    log.New(ioutil.Discard, "test", log.LstdFlags),
					TimeFunc:               func() int64 { return 0 },
					UserDao:                mockUserDao{},
					MaxPlayers:             1,
					NumNewTiles:            1,
					IdlePeriod:             1 * time.Minute,
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {},
					ShufflePlayersFunc:     func(playerNames []player.Name) {},
				},
			},
			wantOk: true,
		},
		{ // no room for game
			ManagerConfig: ManagerConfig{
				Log:      testLog,
				MaxGames: 0,
			},
		},
		{ // bad gameConfig
			ManagerConfig: ManagerConfig{
				Log:      testLog,
				MaxGames: 1,
				GameConfig: Config{
					MaxPlayers: -1,
				},
			},
		},
	}
	for i, test := range gameCreateTests {
		gm := Manager{
			games:         make(map[game.ID]chan<- message.Message, 1),
			lastID:        3,
			ManagerConfig: test.ManagerConfig,
		}
		ctx := context.Background()
		in := make(chan message.Message)
		out, err := gm.Run(ctx, in)
		if err != nil {
			t.Errorf("Test %v: unwanted error running game manager: %v", i, err)
			continue
		}
		m := message.Message{
			Type: message.Create,
		}
		in <- m
		gotNumGames := len(gm.games)
		switch {
		case !test.wantOk:
			if gotNumGames != 0 {
				t.Errorf("Test %v: wanted no game to be created, got %v", i, gotNumGames)
			}
			m2 := <-out
			if m2.Type != message.SocketError {
				t.Errorf("Test %v: wanted returned message to be a warning that to game could be created, but was %v.  Info: %v", i, m2.Type, m2.Info)
			}
		default:
			if gotNumGames != 1 {
				t.Errorf("Test %v: wanted 1 game to be created, got %v", i, gotNumGames)
			}
			wantID := game.ID(4)
			if _, ok := gm.games[wantID]; !ok {
				t.Errorf("Test %v: wanted game of id %v to be created", i, wantID)
			}
		}
	}
}

func TestGameDelete(t *testing.T) {
	gameDeleteTests := []struct {
		m      message.Message
		wantOk bool
	}{
		{
			m: message.Message{
				Type: message.Delete,
			},
		},
		{
			m: message.Message{
				Type: message.Delete,
				Game: &game.Info{
					ID: 4,
				},
			},
		},
		{
			m: message.Message{
				Type: message.Delete,
				Game: &game.Info{
					ID: 5,
				},
			},
			wantOk: true,
		},
	}
	for i, test := range gameDeleteTests {
		in := make(chan message.Message)
		gIn := make(chan message.Message)
		gm := Manager{
			games: map[game.ID]chan<- message.Message{
				5: gIn,
			},
			ManagerConfig: ManagerConfig{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		out, err := gm.Run(ctx, in)
		if err != nil {
			t.Errorf("Test %v: unwanted error running game manager: %v", i, err)
			continue
		}
		messageHandled := false
		go func() { // mock game
			_, ok := <-gIn
			if !ok {
				return
			}
			messageHandled = true
			close(in)
		}()
		in <- test.m
		m2 := <-out
		gotNumGames := len(gm.games)
		switch {
		case !test.wantOk:
			if gotNumGames != 1 {
				t.Errorf("Test %v: wanted 1 game to be not be deleted, got %v", i, gotNumGames)
			}
			if m2.Type != message.SocketError {
				t.Errorf("Test %v: wanted socket error message, got %v", i, m2.Type)
			}
		default:
			if gotNumGames != 0 {
				t.Errorf("Test %v: wanted game to be deleted, yet %v still existed", i, gotNumGames)
			}
			if !messageHandled {
				t.Errorf("Test %v: message not handled", i)
			}
		}
	}
}

func TestHandleGameMessage(t *testing.T) {
	handleGameMessageTests := []struct {
		m      message.Message
		wantOk bool
	}{
		{
			m: message.Message{
				Type: message.Chat,
			},
		},
		{
			m: message.Message{
				Type: message.Chat,
				Game: &game.Info{
					ID: game.ID(2),
				},
			},
		},
		{
			m: message.Message{
				Type: message.Chat,
				Game: &game.Info{
					ID: game.ID(3),
				},
			},
			wantOk: true,
		},
	}
	for i, test := range handleGameMessageTests {
		in := make(chan message.Message)
		gIn := make(chan message.Message)
		gm := Manager{
			games: map[game.ID]chan<- message.Message{
				3: gIn,
			},
			ManagerConfig: ManagerConfig{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		out, err := gm.Run(ctx, in)
		if err != nil {
			t.Errorf("Test %v: unwanted error running game manager: %v", i, err)
			continue
		}
		messageHandled := false
		go func() { // mock game
			_, ok := <-gIn
			if !ok {
				return
			}
			messageHandled = true
			close(in)
		}()
		in <- test.m
		m2 := <-out
		switch {
		case !test.wantOk:
			if m2.Type != message.SocketError {
				t.Errorf("Test %v: wanted socket error message, got %v", i, m2.Type)
			}
		default:
			if !messageHandled {
				t.Errorf("Test %v: message not handled", i)
			}
		}
	}
}
