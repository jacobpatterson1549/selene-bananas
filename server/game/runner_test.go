package game

import (
	"context"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
)

func TestNewRunner(t *testing.T) {
	ud := mockUserDao{}
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	newRunnerTests := []struct {
		RunnerConfig RunnerConfig
		UserDao
		wantOk bool
		want   *Runner
	}{
		{}, // no log
		{ // low MaxGames
			UserDao: ud,
			RunnerConfig: RunnerConfig{
				Log: testLog,
			},
		},
		{ // low MaxGames
			UserDao: ud,
			RunnerConfig: RunnerConfig{
				Log: testLog,
			},
		},
		{
			UserDao: ud,
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 10,
			},
			wantOk: true,
			want: &Runner{
				games:   map[game.ID]chan<- message.Message{},
				UserDao: ud,
				RunnerConfig: RunnerConfig{
					Log:      testLog,
					MaxGames: 10,
				},
			},
		},
	}
	for i, test := range newRunnerTests {
		got, err := test.RunnerConfig.NewRunner(test.UserDao)
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
		var r Runner
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		in := make(chan message.Message)
		out := r.Run(ctx, in)
		test.stopFunc(cancelFunc, in)
		_, ok := <-out
		if ok {
			t.Errorf("Test %v: wanted 'out' channel to be closed after 'in' channel was closed", i)
		}
	}
}

func TestGameCreate(t *testing.T) {
	testLog := log.New(ioutil.Discard, "test", log.LstdFlags)
	gameCreateTests := []struct {
		m            message.Message
		RunnerConfig RunnerConfig
		wantOk       bool
	}{
		{ // happy path
			m: message.Message{
				Type:       message.Create,
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{NumRows: 18, NumCols: 22},
					},
				},
			},
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 1,
				GameConfig: Config{
					PlayerCfg:              playerController.Config{WinPoints: 10},
					Log:                    testLog,
					TimeFunc:               func() int64 { return 0 },
					MaxPlayers:             1,
					NumNewTiles:            1,
					IdlePeriod:             1 * time.Hour,
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {},
					ShufflePlayersFunc:     func(playerNames []player.Name) {},
				},
			},
			wantOk: true,
		},
		{ // no room for game
			m: message.Message{
				Type:       message.Create,
				PlayerName: "selene",
			},
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 0,
			},
		},
		{ // bad message: no game
			m: message.Message{
				Type:       message.Create,
				PlayerName: "selene",
			},
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 1,
			},
		}, { // bad message: no board
			m: message.Message{
				Type:       message.Create,
				PlayerName: "selene",
				Game:       &game.Info{},
			},
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 1,
			},
		},
		{ // bad gameConfig
			m: message.Message{
				Type:       message.Create,
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{NumRows: 18, NumCols: 22},
					},
				},
			},
			RunnerConfig: RunnerConfig{
				Log:      testLog,
				MaxGames: 1,
				GameConfig: Config{
					MaxPlayers: -1,
				},
			},
		},
	}
	for i, test := range gameCreateTests {
		r := Runner{
			games:        make(map[game.ID]chan<- message.Message),
			lastID:       3,
			UserDao:      mockUserDao{},
			RunnerConfig: test.RunnerConfig,
		}
		ctx := context.Background()
		in := make(chan message.Message)
		out := r.Run(ctx, in)
		in <- test.m
		gotM := <-out
		gotNumGames := len(r.games)
		switch {
		case !test.wantOk:
			if gotNumGames != 0 {
				t.Errorf("Test %v: wanted no game to be created, got %v", i, gotNumGames)
			}
			if gotM.Type != message.SocketError {
				t.Errorf("Test %v: wanted returned message to be a warning that to game could be created, but got %v", i, gotM)
			}
		case gotNumGames != 1:
			t.Errorf("Test %v: wanted 1 game to be created, got %v", i, gotNumGames)
		case r.games[4] == nil:
			t.Errorf("Test %v: wanted game of id 4 to be created", i)
		case gotM.Type != message.Join, gotM.Game.ID != 4, gotM.PlayerName != "selene":
			t.Errorf("Test %v: wanted join message for game 4 for player, got %v", i, gotM)
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
		r := Runner{
			games: map[game.ID]chan<- message.Message{
				5: gIn,
			},
			RunnerConfig: RunnerConfig{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		out := r.Run(ctx, in)
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
		gotNumGames := len(r.games)
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
		r := Runner{
			games: map[game.ID]chan<- message.Message{
				3: gIn,
			},
			RunnerConfig: RunnerConfig{
				Log: log.New(ioutil.Discard, "test", log.LstdFlags),
			},
		}
		ctx := context.Background()
		out := r.Run(ctx, in)
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
