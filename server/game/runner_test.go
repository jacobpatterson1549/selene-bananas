package game

import (
	"context"
	"io"
	"log"
	"reflect"
	"sync"
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
	var wc mockWordValidator
	userDao := mockUserDao{}
	testLog := log.New(io.Discard, "", 0)
	newRunnerTests := []struct {
		log *log.Logger
		RunnerConfig
		WordValidator
		UserDao
		wantOk bool
		want   *Runner
	}{
		{}, // no log
		{ // no word validater
			log: testLog,
		},
		{ // no user dao
			log:           testLog,
			WordValidator: wc,
		},
		{ // low MaxGames
			log:           testLog,
			WordValidator: wc,
			UserDao:       userDao,
		},
		{ // low MaxGames
			log:           testLog,
			WordValidator: wc,
			UserDao:       userDao,
		},
		{ // ok
			log:           testLog,
			WordValidator: wc,
			UserDao:       userDao,
			RunnerConfig: RunnerConfig{
				MaxGames: 10,
			},
			wantOk: true,
			want: &Runner{
				log:           testLog,
				games:         map[game.ID]chan<- message.Message{},
				WordValidator: wc,
				userDao:       userDao,
				RunnerConfig: RunnerConfig{
					MaxGames: 10,
				},
			},
		},
		{ // ok debug
			log:           testLog,
			WordValidator: wc,
			UserDao:       userDao,
			RunnerConfig: RunnerConfig{
				Debug:    true,
				MaxGames: 10,
			},
			wantOk: true,
			want: &Runner{
				log:           testLog,
				games:         map[game.ID]chan<- message.Message{},
				WordValidator: wc,
				userDao:       userDao,
				RunnerConfig: RunnerConfig{
					Debug:    true,
					MaxGames: 10,
				},
			},
		},
	}
	for i, test := range newRunnerTests {
		got, err := test.RunnerConfig.NewRunner(test.log, test.WordValidator, test.UserDao)
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
		r := Runner{
			log: log.New(io.Discard, "", 0),
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()
		var wg sync.WaitGroup
		in := make(chan message.Message)
		out := r.Run(ctx, &wg, in)
		test.stopFunc(cancelFunc, in)
		wg.Wait()
		_, ok := <-out
		if ok {
			t.Errorf("Test %v: wanted 'out' channel to be closed after 'in' channel was closed", i)
		}
	}
}

func TestGameCreate(t *testing.T) {
	basicGameConfig := &game.Config{
		CheckOnSnag: true,
	}
	testLog := log.New(io.Discard, "", 0)
	gameCreateTests := []struct {
		m      message.Message
		wantOk bool
		RunnerConfig
	}{
		{ // happy path
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{NumRows: 18, NumCols: 22},
					},
					Config: basicGameConfig,
				},
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 1,
				GameConfig: Config{
					PlayerCfg:   playerController.Config{WinPoints: 10},
					TimeFunc:    func() int64 { return 0 },
					MaxPlayers:  1,
					NumNewTiles: 1,
					IdlePeriod:  1 * time.Hour,
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
						// Not called
					},
					ShufflePlayersFunc: func(playerNames []player.Name) {
						// Not called
					},
				},
			},
			wantOk: true,
		},
		{ // no room for game
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 0,
			},
		},
		{ // bad message: no game
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 1,
			},
		}, { // bad message: no board
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
				Game:       &game.Info{},
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 1,
			},
		},
		{ // no config in game of message
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{NumRows: 18, NumCols: 22},
					},
				},
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 1,
			},
		},
		{ // bad gameConfig
			m: message.Message{
				Type:       message.CreateGame,
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{NumRows: 18, NumCols: 22},
					},
					Config: basicGameConfig,
				},
			},
			RunnerConfig: RunnerConfig{
				MaxGames: 1,
				GameConfig: Config{
					MaxPlayers: -1,
				},
			},
		},
	}
	for i, test := range gameCreateTests {
		var wordValidator mockWordValidator
		var userDao mockUserDao
		r := Runner{
			log:           testLog,
			games:         make(map[game.ID]chan<- message.Message),
			lastID:        3,
			WordValidator: wordValidator,
			userDao:       userDao,
			RunnerConfig:  test.RunnerConfig,
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		in := make(chan message.Message)
		out := r.Run(ctx, &wg, in)
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
		case gotM.Type != message.JoinGame, gotM.Game.ID != 4, gotM.PlayerName != "selene":
			t.Errorf("Test %v: wanted join message for game 4 for player, got %v", i, gotM)
		case r.RunnerConfig.GameConfig.Config != game.Config{}:
			t.Errorf("Test %v: creating a game unwantedly stored the game's config in the runner", i)
		case !reflect.DeepEqual(basicGameConfig, gotM.Game.Config):
			t.Errorf("Test %v: game config not set to basic config:\nwanted: %#v\ngot:    %#v", i, basicGameConfig, gotM.Game.Config)
		default:
			gotM2 := <-out
			if gotM2.Type != message.GameInfos {
				t.Errorf("wanted gameInfos message to be broadcast afther a game was created")
			}
		}
		cancelFunc()
		wg.Wait()
	}
}

func TestGameDelete(t *testing.T) {
	gameDeleteTests := []struct {
		m      message.Message
		wantOk bool
	}{
		{
			m: message.Message{
				Type: message.DeleteGame,
			},
		},
		{
			m: message.Message{
				Type: message.DeleteGame,
				Game: &game.Info{
					ID: 4,
				},
			},
		},
		{
			m: message.Message{
				Type: message.DeleteGame,
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
			log: log.New(io.Discard, "", 0),
			games: map[game.ID]chan<- message.Message{
				5: gIn,
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		out := r.Run(ctx, &wg, in)
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
		case !test.wantOk && gotNumGames != 1:
			t.Errorf("Test %v: wanted 1 game to be not be deleted, got %v", i, gotNumGames)
		case !test.wantOk && m2.Type != message.SocketError:
			t.Errorf("Test %v: wanted socket error message, got %v", i, m2.Type)
		case test.wantOk && gotNumGames != 0:
			t.Errorf("Test %v: wanted game to be deleted, yet %v still existed", i, gotNumGames)
		case test.wantOk && !messageHandled:
			t.Errorf("Test %v: message not handled", i)
		}
		cancelFunc()
		wg.Wait()
	}
}

func TestHandleGameMessage(t *testing.T) {
	handleGameMessageTests := []struct {
		m      message.Message
		wantOk bool
	}{
		{
			m: message.Message{
				Type: message.GameChat,
			},
		},
		{
			m: message.Message{
				Type: message.GameChat,
				Game: &game.Info{
					ID: game.ID(2),
				},
			},
		},
		{
			m: message.Message{
				Type: message.GameChat,
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
			log: log.New(io.Discard, "", 0),
			games: map[game.ID]chan<- message.Message{
				3: gIn,
			},
		}
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		out := r.Run(ctx, &wg, in)
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
		case !messageHandled:
			t.Errorf("Test %v: message not handled", i)
		}
		cancelFunc()
		wg.Wait()
	}
}
