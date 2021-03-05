package game

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"strings"
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

func TestNewGame(t *testing.T) {
	testLog := log.New(io.Discard, "", 0)
	timeFunc := func() int64 {
		return 47
	}
	shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
		// NOOP
	}
	shufflePlayersFunc := func(playerNames []player.Name) {
		// NOOP
	}
	newGameTests := []struct {
		Config
		wantOk bool
		want   *Game
	}{
		{},
		{
			Config: Config{
				TimeFunc:               timeFunc,
				MaxPlayers:             4,
				NumNewTiles:            16,
				TileLetters:            "INVALID WORDS :(",
				IdlePeriod:             1 * time.Hour,
				ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
				ShufflePlayersFunc:     shufflePlayersFunc,
			},
		},
		{
			Config: Config{
				TimeFunc:               timeFunc,
				MaxPlayers:             4,
				NumNewTiles:            16,
				TileLetters:            "VALIDWORDSAREHAPPY",
				IdlePeriod:             1 * time.Hour,
				ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
				ShufflePlayersFunc:     shufflePlayersFunc,
			},
			wantOk: true,
		},
	}
	for i, test := range newGameTests {
		id := game.ID(7)
		var WordValidator mockWordValidator
		var userDao mockUserDao
		got, err := test.Config.NewGame(testLog, id, WordValidator, userDao)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(testLog, got.log),
			got.id != id,
			got.createdAt != 47,
			got.status != game.NotStarted,
			len(got.players) != 0,
			!reflect.DeepEqual(WordValidator, got.WordValidator),
			!reflect.DeepEqual(userDao, got.userDao),
			!reflect.DeepEqual(test.Config.TileLetters, got.Config.TileLetters):
			t.Errorf("Test %v: fields not set", i)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("TestErrCheck", func(t *testing.T) {
		testLog := log.New(io.Discard, "", 0)
		timeFunc := func() int64 {
			return 0
		}
		shuffleUnusedTilesFunc := func(tiles []tile.Tile) {
			// NOOP
		}
		shufflePlayersFunc := func(playerNames []player.Name) {
			// NOOP
		}
		var wordValidator mockWordValidator
		var userDao mockUserDao
		errCheckTests := []struct {
			Config
			*log.Logger
			game.ID
			WordValidator
			UserDao
			wantOk bool
		}{
			{}, // no log
			{ // id  not positive
				Logger: testLog,
			},
			{ // no word validator
				Logger: testLog,
				ID:     1,
			},
			{ // no user dao
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
			},
			{ // no time func
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // low maxPlayers
				Config: Config{
					TimeFunc: timeFunc,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // low num newTiles
				Config: Config{
					TimeFunc:   timeFunc,
					MaxPlayers: 4,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // low idle period
				Config: Config{
					TimeFunc:    timeFunc,
					MaxPlayers:  4,
					NumNewTiles: 16,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // missing shuffle tiles func
				Config: Config{
					TimeFunc:    timeFunc,
					MaxPlayers:  4,
					NumNewTiles: 16,
					IdlePeriod:  1 * time.Hour,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // missing shuffle players func
				Config: Config{
					TimeFunc:               timeFunc,
					MaxPlayers:             4,
					NumNewTiles:            16,
					IdlePeriod:             1 * time.Hour,
					ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{ // too few tiles for one player to start
				Config: Config{
					TimeFunc:               timeFunc,
					MaxPlayers:             4,
					NumNewTiles:            16,
					TileLetters:            "ABC",
					IdlePeriod:             1 * time.Hour,
					ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
					ShufflePlayersFunc:     shufflePlayersFunc,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
			},
			{
				Config: Config{
					TimeFunc:               timeFunc,
					MaxPlayers:             4,
					NumNewTiles:            16,
					TileLetters:            "HOWMANYWORDSCANYOUMAKEWITHTHESELETTERS",
					IdlePeriod:             1 * time.Hour,
					ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
					ShufflePlayersFunc:     shufflePlayersFunc,
				},
				Logger:        testLog,
				ID:            1,
				WordValidator: wordValidator,
				UserDao:       userDao,
				wantOk:        true,
			},
		}
		for i, test := range errCheckTests {
			err := test.Config.validate(test.Logger, test.ID, test.WordValidator, test.UserDao)
			switch {
			case !test.wantOk:
				if err == nil {
					t.Errorf("Test %v: wanted error", i)
				}
			case err != nil:
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		}
	})
	t.Run("TestSetTileLetters", func(t *testing.T) {
		setTileLettersTests := []struct {
			Config
			wantTileLetters string
		}{
			{
				wantTileLetters: defaultTileLetters,
			},
			{
				Config: Config{
					TileLetters: "ABC",
				},
				wantTileLetters: "ABC",
			},
		}
		for i, test := range setTileLettersTests {
			log := log.New(io.Discard, "", 0)
			wordValidator := mockWordValidator(func(word string) bool { return false })
			userDao := mockUserDao{}
			test.Config.validate(log, 1, wordValidator, userDao) // Ignore the error.  This test doesn't care about it.
			got := test.Config
			if test.wantTileLetters != got.TileLetters {
				t.Errorf("Test %v: not equal:\nwanted: %v\ngot:    %v", i, test.wantTileLetters, got.TileLetters)
			}
		}
	})
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	in := make(chan message.Message)
	out := make(chan message.Message, 1)
	g := Game{
		players: map[player.Name]*playerController.Player{
			"selene": nil,
		},
		Config: Config{
			IdlePeriod: 1 * time.Hour,
		},
	}
	g.Run(ctx, &wg, in, out)
	m := message.Message{
		Type:       message.GameChat,
		PlayerName: "selene",
	}
	in <- m
	m2 := <-out
	cancelFunc()
	wg.Wait()
	if m2.Type != message.GameChat {
		t.Errorf("wanted game to relay simple chat message back to player")
	}
}

func TestRunSync(t *testing.T) {
	testLog := log.New(io.Discard, "", 0)
	t.Run("TestValidMessageHandler", func(t *testing.T) {
		validMessageHandlerTests := []struct {
			message.Type
			wantError bool // most tests should return a gameWarning, indicating the a messageHandler exists for the message Type
		}{
			{
				Type:      message.SocketHTTPPing,
				wantError: true,
			},
			{
				Type: message.JoinGame,
			},
			{
				Type: message.DeleteGame,
			},
			{
				Type: message.ChangeGameStatus,
			},
			{
				Type: message.SnagGameTile,
			},
			{
				Type: message.SwapGameTile,
			},
			{
				Type: message.MoveGameTile,
			},
			{
				Type: message.GameChat,
			},
			{
				Type: message.RefreshGameBoard,
			},
		}
		for i, test := range validMessageHandlerTests {
			m := message.Message{
				Type:       test.Type,
				PlayerName: "selene",
				Game: &game.Info{
					Status: game.Finished,
					Board:  &board.Board{},
				},
			}
			g := Game{
				status: 0, // this should cause a gameWarning error
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: &board.Board{},
					},
				},
			}
			ctx := context.Background()
			ctx, cancelFunc := context.WithCancel(ctx)
			var wg sync.WaitGroup
			in := make(chan message.Message, 1)  // deleteGame will cause the run to stop before the second message is passed
			out := make(chan message.Message, 2) // the second message might be handled, for an unknown message type
			idleTicker := &time.Ticker{}
			wg.Add(1)
			go g.runSync(ctx, &wg, in, out, idleTicker)
			in <- m
			in <- message.Message{} // force the game to handle the first Message, this will cause a socketError message to be sent
			cancelFunc()
			wg.Wait()
			got := <-out
			if test.wantError != (got.Type == message.SocketError) {
				t.Errorf("Test %v: when test is %v, got %v", i, test, got.Type)
			}
		}
	})
	t.Run("TestRunSyncStop", func(t *testing.T) {
		testRunSyncTickerTests := []struct {
			ctxCancelled      bool
			inClosed          bool
			idleTick          bool
			gameDeleteMessage bool
		}{
			{ctxCancelled: true},
			{inClosed: true},
			{idleTick: true},
			{gameDeleteMessage: true},
		}
		for i, test := range testRunSyncTickerTests {
			ctx := context.Background()
			ctx, cancelFunc := context.WithCancel(ctx)
			var wg sync.WaitGroup
			in := make(chan message.Message, 1)
			out := make(chan message.Message, 2)
			idleC := make(chan time.Time, 1)
			idleTicker := &time.Ticker{
				C: idleC,
			}
			pn := player.Name("selene")
			g := Game{
				log: testLog,
				players: map[player.Name]*playerController.Player{
					pn: nil,
				},
			}
			switch {
			case test.ctxCancelled:
				cancelFunc()
			case test.inClosed:
				close(in)
			case test.idleTick:
				idleC <- time.Time{}
			case test.gameDeleteMessage:
				m := message.Message{
					Type:       message.DeleteGame,
					PlayerName: pn,
				}
				in <- m
			}
			wg.Add(1)
			g.runSync(ctx, &wg, in, out, idleTicker)
			cancelFunc()
			wg.Wait()
			numWaiting := len(out)
			wantGameDelete := test.idleTick || test.gameDeleteMessage
			switch {
			case !wantGameDelete:
				if numWaiting != 0 {
					t.Errorf("Test %v: wanted no messages left on out channel, got %v", i, numWaiting)
				}
			case numWaiting != 2:
				t.Errorf("Test %v: wanted two messages left on out channel, got %v", i, numWaiting)
			default:
				gotM1 := <-out
				gotM2 := <-out
				switch {
				case gotM1.Type != message.LeaveGame, gotM1.PlayerName != pn:
					t.Errorf("Test %v: wanted leave message sent to %v, got %v", i, pn, gotM1)
				case gotM2.Type != message.GameInfos:
					t.Errorf("Test %v: wanted final message sent by inactive game to be GameInfos, got %v", i, gotM2)
				}
			}
		}
	})
	t.Run("TestRunSyncIdleTickAfterMessage", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancelFunc := context.WithCancel(ctx)
		var wg sync.WaitGroup
		in := make(chan message.Message)
		out := make(chan message.Message, 2)
		idleC := make(chan time.Time)
		idleTicker := &time.Ticker{
			C: idleC,
		}
		pn := player.Name("selene")
		g := Game{
			log: testLog,
			players: map[player.Name]*playerController.Player{
				pn: nil,
			},
		}
		wg.Add(1)
		go g.runSync(ctx, &wg, in, out, idleTicker)
		in <- message.Message{
			Type:       message.GameChat,
			PlayerName: pn,
		}
		<-out // error for unknown message type
		idleC <- time.Time{}
		cancelFunc()
		wg.Wait()
		numWaiting := len(out)
		if numWaiting != 0 {
			t.Errorf("wanted no messages left on out channel, got %v", numWaiting)
		}
	})
}

func TestSendMessage(t *testing.T) {
	sendMessageTests := []struct {
		Game
		message.Message
		wantGameID game.ID
	}{
		{ // no game on message
			Game: Game{
				id: 1,
			},
			wantGameID: 1,
		},
		{
			Game: Game{
				id: 2,
			},
			Message: message.Message{
				Game: &game.Info{
					ID: 2,
				},
			},
			wantGameID: 2,
		},
		{ // should be overwritten because it is coming from game 2
			Game: Game{
				id: 3,
			},
			Message: message.Message{
				Game: &game.Info{
					ID: 4,
				},
			},
			wantGameID: 3,
		},
	}
	for i, test := range sendMessageTests {
		out := make(chan message.Message, 1)
		send := test.Game.sendMessage(out)
		send(test.Message)
		got := <-out
		gotGameID := got.Game.ID
		if test.wantGameID != gotGameID {
			t.Errorf("Test %v: game ids not equal: wanted %v, got %v", i, test.wantGameID, got.Game.ID)
		}
	}
}

func TestInitializeUnusedTilesShuffled(t *testing.T) {
	createTilesShuffledTests := []struct {
		want      tile.Letter
		inReverse string
	}{
		{'A', ""},
		{'Z', " IN REVERSE"},
	}
	for _, test := range createTilesShuffledTests {
		g := Game{
			Config: Config{
				TileLetters: "AZ",
				ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
					sort.Slice(tiles, func(i, j int) bool {
						lessThan := tiles[i].Ch < tiles[j].Ch
						if len(test.inReverse) > 0 {
							return !lessThan
						}
						return lessThan
					})
				},
			},
		}
		if err := g.initializeUnusedTiles(); err != nil {
			t.Errorf("unwanted error: %v", err)
		}
		got := g.unusedTiles[0].Ch
		if test.want != got {
			t.Errorf("wanted first tile to be %q when sorted%v (a fake shuffle), but was %q", test.want, test.inReverse, got)
		}
	}
}

func TestInitializeUnusedTiles(t *testing.T) {
	initializeUnusedTilesTests := []struct {
		tileLetters string
		wantOk      bool
		want        []tile.Tile
	}{
		{
			tileLetters: ":(",
		},
		{
			tileLetters: "AAABAC",
			wantOk:      true,
			want: []tile.Tile{
				{ID: 6, Ch: 'C'},
				{ID: 4, Ch: 'B'},
				{ID: 1, Ch: 'A'},
				{ID: 2, Ch: 'A'},
				{ID: 3, Ch: 'A'},
				{ID: 5, Ch: 'A'},
			},
		},
	}
	for i, test := range initializeUnusedTilesTests {
		shuffleFunc := func(tiles []tile.Tile) {
			sort.Slice(tiles, func(i, j int) bool {
				if tiles[i].Ch == tiles[j].Ch {
					return tiles[i].ID < tiles[j].ID
				}
				return tiles[i].Ch > tiles[j].Ch
			})
		}
		g := Game{
			Config: Config{
				TileLetters:            test.tileLetters,
				ShuffleUnusedTilesFunc: shuffleFunc,
			},
		}
		err := g.initializeUnusedTiles()
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, g.unusedTiles):
			t.Errorf("Test %v: unusedTiles not equal:\nwanted: %v\ngot:    %v", i, test.want, g.unusedTiles)
		}
	}
}

func TestUpdateUserPoints(t *testing.T) {
	want := fmt.Errorf("calling UpdatePointsIncrement")
	ctx := context.Background()
	wantUserPoints := map[string]int{
		"alice":  1,
		"bob":    1,
		"selene": 5,
	}
	userDao := mockUserDao{
		UpdatePointsIncrementFunc: func(ctx context.Context, gotUserPoints map[string]int) error {
			switch {
			case ctx == nil:
				return fmt.Errorf("context missing")
			case !reflect.DeepEqual(wantUserPoints, gotUserPoints):
				return fmt.Errorf("user points not equal\nwanted: %v\ngot:    %v", wantUserPoints, gotUserPoints)
			}
			return want
		},
	}
	alicePlayer, err := playerController.Config{WinPoints: 4}.New(&board.Board{})
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	selenePlayer, err := playerController.Config{WinPoints: 5}.New(&board.Board{})
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	g := Game{
		players: map[player.Name]*playerController.Player{
			"alice":  alicePlayer,
			"selene": selenePlayer,
			"bob":    {},
		},
		userDao: userDao,
	}
	got := g.updateUserPoints(ctx, "selene")
	if want != got {
		t.Errorf("wanted error %v, got %v", want, got)
	}
}

func TestPlayerNames(t *testing.T) {
	g := Game{
		players: map[player.Name]*playerController.Player{
			"b": {},
			"c": {},
			"a": {},
		},
	}
	want := []string{"a", "b", "c"}
	got := g.playerNames()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("player names not equal/sorted:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestHandleMessage(t *testing.T) {
	handleMessageTests := []struct {
		message.Message
		Game
		messageHandlers map[message.Type]messageHandler
		wantSendType    message.Type
		wantActive      bool
		wantLog         bool
	}{
		{ // unknown message type #1
			wantSendType: message.SocketError,
		},
		{ // unknown message type #2
			Message: message.Message{
				Type: message.CreateGame,
			},
			messageHandlers: map[message.Type]messageHandler{
				message.JoinGame: func(ctx context.Context, m message.Message, send messageSender) error {
					return nil
				},
			},
			wantSendType: message.SocketError,
		},
		{ // unknown message type #3 with debug
			Game: Game{
				Config: Config{
					Debug: true,
				},
			},
			wantSendType: message.SocketError,
			wantLog:      true,
		},
		{ // unknown player
			Message: message.Message{
				Type:       message.SnagGameTile,
				PlayerName: "ghost",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return nil
				},
			},
			wantSendType: message.SocketError,
		},
		{ // message handler error
			Message: message.Message{
				Type:       message.SnagGameTile,
				PlayerName: "selene",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return fmt.Errorf("snag tile error")
				},
			},
			wantSendType: message.SocketError,
			wantActive:   true,
		},
		{ // message handler error game warning
			Message: message.Message{
				Type:       message.SnagGameTile,
				PlayerName: "selene",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return gameWarning("snag tile warning")
				},
			},
			wantSendType: message.SocketWarning,
			wantActive:   true,
		},
		{ // message ok
			Message: message.Message{
				Type:       message.SwapGameTile,
				PlayerName: "selene",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return fmt.Errorf("wrong handler")
				},
				message.SwapGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					m2 := message.Message{
						Type: message.ChangeGameTiles,
					}
					send(m2)
					return nil
				},
			},
			wantSendType: message.ChangeGameTiles,
			wantActive:   true,
		},
		{ // message ok with no want send type (normal snag sends no message)
			Message: message.Message{
				Type:       message.SnagGameTile,
				PlayerName: "selene",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return nil
				},
			},
			wantActive: true,
		},
		{ // message ok with debug
			Message: message.Message{
				Type:       message.SnagGameTile,
				PlayerName: "selene",
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": nil,
				},
				Config: Config{
					Debug: true,
				},
			},
			messageHandlers: map[message.Type]messageHandler{
				message.SnagGameTile: func(ctx context.Context, m message.Message, send messageSender) error {
					return nil
				},
			},
			wantActive: true,
			wantLog:    true,
		},
	}
	for i, test := range handleMessageTests {
		ctx := context.Background()
		var buf bytes.Buffer
		test.Game.log = log.New(&buf, "", 0)
		gotSend := false
		send := func(m message.Message) {
			gotSend = true
			switch {
			case m.Type != test.wantSendType:
				t.Errorf("Test %v: wanted message sent with type %v, got %v", i, m.Type, test.wantSendType)
			case m.Type == message.SocketError, m.Type == message.SocketWarning:
				if m.PlayerName != test.Message.PlayerName || !reflect.DeepEqual(test.Message.Game, m.Game) || len(m.Info) == 0 {
					t.Errorf("Test: %v wanted message for %v with game and error info, got %v", i, test.Message.PlayerName, m)
				}
			}
		}
		active := false
		test.Game.handleMessage(ctx, test.Message, send, &active, test.messageHandlers)
		switch {
		case active != test.wantActive:
			t.Errorf("Test %v: wanted active flag to be %v after handler was run, got %v", i, active, test.wantActive)
		case test.wantLog != (buf.Len() > 0):
			t.Errorf("Test %v: wanted message logged (%v), got %v", i, test.wantLog, buf.Len() > 0)
		case test.wantSendType != 0 && !gotSend:
			t.Errorf("Test %v: wanted message to be sent", i)
		}
	}
}

func TestHandleGameJoin(t *testing.T) {
	handleGameJoinTests := []struct {
		message.Message
		Game
		wantOk         bool
		wantNumPlayers int
	}{
		{ // rejoin
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Board: &board.Board{},
				},
			},
			Game: Game{
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: &board.Board{},
					},
				},
			},
			wantOk:         true,
			wantNumPlayers: 1,
		},
		{}, // game not started
		{ // no room for new player
			Game: Game{
				status: game.NotStarted,
			},
		},
		{ // not enough tiles for new player
			Game: Game{
				status: game.NotStarted,
				Config: Config{
					MaxPlayers:  4,
					NumNewTiles: 12,
				},
			},
		},
		{ // add player, error
			Message: message.Message{
				PlayerName: "young",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 79,
							NumCols: 28,
						},
					},
				},
			},
			Game: Game{
				status: game.NotStarted,
				Config: Config{
					MaxPlayers: 3,
					PlayerCfg: playerController.Config{
						WinPoints: 7,
					},
				},
				players: map[player.Name]*playerController.Player{
					"crosby": {},
					"stills": {},
					"nash":   {},
				},
			},
		},
		{ // add player
			Message: message.Message{
				PlayerName: "young",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 79,
							NumCols: 28,
						},
					},
				},
			},
			Game: Game{
				status: game.NotStarted,
				Config: Config{
					MaxPlayers: 4,
					PlayerCfg: playerController.Config{
						WinPoints: 7,
					},
				},
				players: map[player.Name]*playerController.Player{
					"crosby": {},
					"stills": {},
					"nash":   {},
				},
			},
			wantOk:         true,
			wantNumPlayers: 4,
		},
	}
	for i, test := range handleGameJoinTests {
		ctx := context.Background()
		messageSent := false
		send := func(m message.Message) {
			messageSent = true
			if !test.wantOk && (m.Type != message.LeaveGame || m.PlayerName != test.Message.PlayerName) {
				t.Errorf("Test %v: wanted leavegame message sent to %v if an error occurred, got %v", i, test.Message.PlayerName, m)
			}
		}
		err := test.Game.handleGameJoin(ctx, test.Message, send)
		switch {
		case !messageSent:
			t.Errorf("Test %v: wanted at least one message sent after a join-game attempt", i)
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.wantNumPlayers != len(test.Game.players):
			t.Errorf("Test %v: wanted %v players in game after join/rejoin, got %v", i, test.wantNumPlayers, len(test.Game.players))
		}
	}
}

func TestHandleAddPlayer(t *testing.T) {
	withConfig := func(b *board.Board, cfg board.Config) *board.Board {
		b.Config = cfg
		return b
	}
	hasPlayer := func(players []string, player string) bool {
		for _, p := range players {
			if p == player {
				return true
			}
		}
		return false
	}
	handleAddPlayerTests := []struct {
		message.Message
		Game
		wantOk          bool
		wantUnusedTiles []tile.Tile
		wantPlayer      *playerController.Player
	}{
		{ // board config error
			Message: message.Message{
				Game: &game.Info{
					Board: &board.Board{},
				},
			},
		},
		{ // player config error
			Message: message.Message{
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 79,
							NumCols: 28,
						},
					},
				},
			},
			Game: Game{
				Config: Config{
					PlayerCfg: playerController.Config{
						WinPoints: -1,
					},
				},
			},
		},
		{ // happy path
			Message: message.Message{
				PlayerName: "young",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 79,
							NumCols: 28,
						},
					},
				},
			},
			Game: Game{
				Config: Config{
					NumNewTiles: 2,
					PlayerCfg: playerController.Config{
						WinPoints: 7,
					},
				},
				players: map[player.Name]*playerController.Player{
					"crosby": {},
					"stills": {},
					"nash":   {},
				},
				unusedTiles: []tile.Tile{{ID: 11}, {ID: 22}, {ID: 3}, {ID: 6}, {ID: 2}},
			},
			wantOk:          true,
			wantUnusedTiles: []tile.Tile{{ID: 3}, {ID: 6}, {ID: 2}},
			wantPlayer: &playerController.Player{
				WinPoints: 7,
				Board: withConfig(
					board.New([]tile.Tile{{ID: 11}, {ID: 22}}, nil),
					board.Config{
						NumRows: 79,
						NumCols: 28,
					}),
			},
		},
	}
	for i, test := range handleAddPlayerTests {
		ctx := context.Background()
		gotInfoChanged := false
		gotMessages := make(map[player.Name]struct{}, len(test.Game.players))
		wantTilesLeft := len(test.wantUnusedTiles)
		send := func(m message.Message) {
			if m.Type == message.GameInfos {
				gotInfoChanged = true
				return
			}
			pn := m.PlayerName
			if _, ok := test.Game.players[pn]; !ok {
				t.Errorf("message sent to unknown player: %v", m)
			}
			if _, ok := gotMessages[pn]; ok {
				t.Errorf("extra message sent to %v: %v", pn, m)
			}
			gotMessages[pn] = struct{}{}
			switch {
			case wantTilesLeft != m.Game.TilesLeft:
				t.Errorf("Test %v: tilesLeft in message sent to %v not equal: wanted %v, got: %v", i, pn, wantTilesLeft, m.Game.TilesLeft)
			case !hasPlayer(m.Game.Players, string(test.Message.PlayerName)):
				t.Errorf("Test %v: wanted new player %v to be in players slice, got %v", i, test.Message.PlayerName, m.Game.Players)
			case pn == test.Message.PlayerName && m.Game.Board == nil:
				t.Errorf("Test %v: wanted board resize info sent to new player (%v), got %v", i, test.Message.PlayerName, m)
			}
		}
		err := test.Game.handleAddPlayer(ctx, test.Message, send)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case len(test.Game.players) != len(gotMessages):
			t.Errorf("Test %v: wanted messages sent to all players but the new one (%v total), got %v", i, len(test.Game.players), len(gotMessages))
		case !gotInfoChanged:
			t.Errorf("Test %v: wanted to get message to change game info", i)
		case !reflect.DeepEqual(test.wantUnusedTiles, test.Game.unusedTiles):
			t.Errorf("Test %v: game unused tiles not equal after adding new player:\nwanted: %v\ngot:    %v", i, test.wantUnusedTiles, test.Game.unusedTiles)
		case !reflect.DeepEqual(test.wantPlayer, test.Game.players[test.Message.PlayerName]):
			t.Errorf("Test %v: new player not equal:\nwanted: %v\ngot:    %v", i, test.wantPlayer, test.Game.players[test.Message.PlayerName])
		}
	}
}

func TestHandleGameDelete(t *testing.T) {
	g := Game{
		players: map[player.Name]*playerController.Player{
			"larry": {},
			"curly": {},
			"moe":   {},
		},
	}
	ctx := context.Background()
	var m message.Message
	gotMessages := make(map[player.Name]struct{}, len(g.players))
	gotInfoChanged := false
	send := func(m message.Message) {
		switch m.Type {
		case message.GameInfos:
			gotInfoChanged = true
			return
		case message.LeaveGame:
			// NOOP falthrough
		default:
			t.Errorf("wanted leave game message, got %v", m.Type)
		}
		pn := m.PlayerName
		if _, ok := g.players[pn]; !ok {
			t.Errorf("message sent to unknown player: %v", m)
		}
		if _, ok := gotMessages[pn]; ok {
			t.Errorf("extra message sent to %v: %v", pn, m)
		}
		gotMessages[pn] = struct{}{}
	}
	err := g.handleGameDelete(ctx, m, send)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case len(gotMessages) != 3:
		t.Errorf("wanted messages sent to all players (%v), got %v", len(g.players), len(gotMessages))
	case g.status != game.Deleted:
		t.Errorf("wanted game status to be deleted, got %v", g.status)
	case !gotInfoChanged:
		t.Errorf("wanted to get message to change game info")
	}
}

func TestHandleGameStatusChange(t *testing.T) {
	handleGameStatusChangeTests := []struct {
		message.Message
		Game
		wantOk     bool
		wantStatus game.Status
		wantInfos  bool
	}{
		{ // do not update the game status
			Message: message.Message{
				Game: &game.Info{
					Status: game.NotStarted,
				},
			},
		},
		{ // bad start
			Message: message.Message{
				Game: &game.Info{
					Status: game.InProgress,
				},
			},
			Game: Game{
				status: game.InProgress,
			},
		},
		{ // bad finish
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Status: game.Finished,
				},
			},
			Game: Game{
				status: game.InProgress,
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{ // multiple groups
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
							{Tile: tile.Tile{ID: 3}, X: 8, Y: 4},
						}),
					},
				},
			},
		},
		{ // ok start game
			Message: message.Message{
				Game: &game.Info{
					Status: game.InProgress,
				},
			},
			Game: Game{
				status: game.NotStarted,
			},
			wantOk:     true,
			wantStatus: game.InProgress,
		},
		{ // ok finish game
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Status: game.Finished,
				},
			},
			Game: Game{
				status: game.InProgress,
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{{}}),
					},
				},
				userDao: mockUserDao{
					UpdatePointsIncrementFunc: func(ctx context.Context, userPoints map[string]int) error {
						return nil
					},
				},
			},
			wantOk:     true,
			wantStatus: game.Finished,
		},
	}
	for i, test := range handleGameStatusChangeTests {
		ctx := context.Background()
		gotInfoChanged := false
		send := func(m message.Message) {
			if m.Type == message.GameInfos {
				gotInfoChanged = true
			}
		}
		err := test.Game.handleGameStatusChange(ctx, test.Message, send)
		switch {
		case !test.wantOk:
			switch {
			case err == nil:
				t.Errorf("Test %v: wanted error", i)
			case !test.wantInfos && gotInfoChanged:
				t.Errorf("Test %v: did not want infos changed", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !gotInfoChanged:
			t.Errorf("Test %v: wanted to get message to change game info", i)
		case test.wantStatus != test.Game.status:
			t.Errorf("Test %v: game statuses not equal: wanted %v, got %v", i, test.wantStatus, test.Game.status)
		}
	}
}

func TestHandleGameStart(t *testing.T) {
	handleGameStartTests := []struct {
		message.Message
		Game
		wantOk        bool
		wantTilesLeft int
	}{
		{}, // game not started
		{
			Message: message.Message{
				PlayerName: "curly",
			},
			Game: Game{
				status: game.NotStarted,
				players: map[player.Name]*playerController.Player{
					"moe":   nil,
					"larry": nil,
					"curly": nil,
				},
				unusedTiles: []tile.Tile{{}, {}, {}, {}},
			},
			wantOk:        true,
			wantTilesLeft: 4,
		},
	}
	for i, test := range handleGameStartTests {
		ctx := context.Background()
		gotMessages := make(map[player.Name]struct{}, len(test.Game.players))
		send := func(m message.Message) {
			pn := m.PlayerName
			if _, ok := test.Game.players[pn]; !ok {
				t.Errorf("Test %v: message sent to unknown player: %v", i, m)
			}
			if _, ok := gotMessages[pn]; ok {
				t.Errorf("Test %v: extra message sent to %v: %v", i, pn, m)
			}
			gotMessages[pn] = struct{}{}
			switch {
			case m.Type != message.ChangeGameStatus,
				!strings.Contains(m.Info, string(test.Message.PlayerName)),
				m.Game.TilesLeft != test.wantTilesLeft:
				t.Errorf("Test %v: wanted change game status message from %v, got %v", i, test.Message.PlayerName, m)
			}
		}
		err := test.Game.handleGameStart(ctx, test.Message, send)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case len(test.Game.players) != len(gotMessages):
			t.Errorf("Test %v: wanted messages sent to all players (%v), got %v", i, len(test.Game.players), len(gotMessages))
		}
	}
}

func TestCheckPlayerBoard(t *testing.T) {
	checkPlayerBoardTests := []struct {
		playerController.Player
		WordValidator
		game.Config
		checkWords    bool
		wantWinPoints int
		penalize      bool
		wantOk        bool
		wantUsedWords []string
	}{
		{ // not all tiles used
			Player: playerController.Player{
				WinPoints: 6,
				Board:     board.New([]tile.Tile{{}}, nil),
			},
			wantWinPoints: 6,
		},
		{ // multiple letter groups
			Player: playerController.Player{
				Board: board.New(nil, []tile.Position{
					{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
					{Tile: tile.Tile{ID: 3}, X: 8, Y: 4},
				}),
			},
		},
		{ // check words error
			Player: playerController.Player{
				Board: board.New(nil, []tile.Position{
					{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
					{Tile: tile.Tile{ID: 3}, X: 4, Y: 4},
				}),
			},
			WordValidator: mockWordValidator(func(word string) bool {
				return false
			}),
			checkWords: true,
		},
		{ // win points decremented from 10,
			Player: playerController.Player{
				WinPoints: 10,
				Board:     board.New([]tile.Tile{{}}, nil),
			},
			wantWinPoints: 9,
			WordValidator: mockWordValidator(func(word string) bool {
				return false
			}),
			Config: game.Config{
				Penalize: true,
			},
		},
		{ // ok, check used words
			Player: playerController.Player{
				WinPoints: 3,
				Board: board.New(nil, []tile.Position{
					{Tile: tile.Tile{ID: 2, Ch: 'Q'}, X: 3, Y: 4},
					{Tile: tile.Tile{ID: 3, Ch: 'X'}, X: 4, Y: 4},
				}),
			},
			checkWords: true,
			WordValidator: mockWordValidator(func(word string) bool {
				return true
			}),
			wantOk:        true,
			wantUsedWords: []string{"QX"},
			wantWinPoints: 3,
		},
		{ // ok, would-be check words error
			Player: playerController.Player{
				WinPoints: 8,
				Board: board.New(nil, []tile.Position{
					{Tile: tile.Tile{ID: 2, Ch: 'Q'}, X: 3, Y: 4},
					{Tile: tile.Tile{ID: 3, Ch: 'X'}, X: 4, Y: 4},
				}),
			},
			WordValidator: mockWordValidator(func(word string) bool {
				return false
			}),
			Config: game.Config{
				CheckOnSnag: true, // this is overridden
			},
			checkWords:    false, // this is being tested
			wantOk:        true,
			wantWinPoints: 8,
		},
	}
	pn := player.Name("selene")
	for i, test := range checkPlayerBoardTests {
		g := Game{
			players: map[player.Name]*playerController.Player{
				pn: &test.Player,
			},
			WordValidator: test.WordValidator,
			Config: Config{
				Config: test.Config,
			},
		}
		gotUsedWords, err := g.checkPlayerBoard(pn, test.checkWords)
		switch {
		case test.wantWinPoints != g.players[pn].WinPoints:
			t.Errorf("Test %v: wanted player win points to be %v after check, got %v", i, test.wantWinPoints, g.players[pn].WinPoints)
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.wantUsedWords, gotUsedWords):
			t.Errorf("Test % v: used words not equal:\nwanted: %v\ngot:    %v", i, test.wantUsedWords, gotUsedWords)
		}
	}
}

func TestCheckPlayerBoardPenalize(t *testing.T) {
	checkPlayerBoardPenalizeTests := []struct {
		penalize       bool
		startWinPoints int
		numChecks      int
		wantWinPoints  int
	}{
		{
			penalize:       false,
			startWinPoints: 10,
			numChecks:      99,
			wantWinPoints:  10,
		},
		{
			penalize:       true,
			startWinPoints: 8,
			numChecks:      5,
			wantWinPoints:  3,
		},
		{
			penalize:       true,
			startWinPoints: 8,
			numChecks:      7,
			wantWinPoints:  2, // should stay above one
		},
	}
	for i, test := range checkPlayerBoardPenalizeTests {
		pn := player.Name("selene")
		unusedTiles := []tile.Tile{{}} // 1 unused tile
		g := Game{
			players: map[player.Name]*playerController.Player{
				pn: {
					Board:     board.New(unusedTiles, nil),
					WinPoints: test.startWinPoints,
				},
			},
			Config: Config{
				Config: game.Config{
					Penalize: test.penalize,
				},
			},
		}
		for j := 0; j < test.numChecks; j++ {
			g.checkPlayerBoard(pn, false)
		}
		got := g.players[pn].WinPoints
		if test.wantWinPoints != got {
			t.Errorf("Test %v: wanted player to have %v winPoints, got %v", i, test.wantWinPoints, got)
		}
	}
}

func TestCheckWords(t *testing.T) {
	checkWordsTests := []struct {
		game.Config
		*board.Board
		WordValidator
		wantOk        bool
		wantUsedWords []string
	}{
		{ // happy path
			Board: board.New(nil, []tile.Position{
				{X: 2, Y: 2},
				{X: 1, Y: 2},
				{X: 2, Y: 1}, // y coordinates are inverted (upperleft-> down)
			}),
			WordValidator: mockWordValidator(func(word string) bool {
				return true
			}),
		},
		{ // short word
			Config: game.Config{
				MinLength: 3,
			},
			Board: board.New(nil, []tile.Position{
				{X: 2, Y: 2},
				{X: 1, Y: 2},
			}),
		},
		{ // word validator error
			Board: board.New(nil, []tile.Position{
				{X: 2, Y: 2},
				{X: 1, Y: 2},
			}),
			WordValidator: mockWordValidator(func(word string) bool {
				return false
			}),
		},
		{ // ok (with duplicate and short words, but config is loose)
			Config: game.Config{
				AllowDuplicates: true,
			},
			Board: board.New(nil, []tile.Position{
				{Tile: tile.Tile{ID: 1, Ch: 'Q'}, X: 2, Y: 2},
				{Tile: tile.Tile{ID: 2, Ch: 'X'}, X: 1, Y: 2},
				{Tile: tile.Tile{ID: 3, Ch: 'X'}, X: 2, Y: 1}, // y coordinates are inverted (upperleft-> down)
			}),
			WordValidator: mockWordValidator(func(word string) bool {
				return true
			}),
			wantOk:        true,
			wantUsedWords: []string{"XQ", "XQ"},
		},
	}
	for i, test := range checkWordsTests {
		pn := player.Name("selene")
		g := Game{
			Config: Config{
				Config: test.Config,
			},
			players: map[player.Name]*playerController.Player{
				pn: {
					Board: test.Board,
				},
			},
			WordValidator: test.WordValidator,
		}
		gotUsedWords, err := g.checkWords(pn)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.wantUsedWords, gotUsedWords):
			t.Errorf("Test % v: used words not equal:\nwanted: %v\ngot:    %v", i, test.wantUsedWords, gotUsedWords)
		}
	}
}

func TestHandleGameFinish(t *testing.T) {
	handleGameFinishTests := []struct {
		message.Message
		Game
		wantOk     bool
		userDaoErr error
	}{

		{}, // game not in progress
		{ // game has used tiles left
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{}},
			},
		},
		{ // player board has multiple used groups
			Message: message.Message{
				PlayerName: "selene",
			},
			Game: Game{
				status: game.InProgress,
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
							{Tile: tile.Tile{ID: 3}, X: 8, Y: 4},
						}),
					},
				},
			},
		},
		{ // happy path
			Message: message.Message{
				PlayerName: "fred",
			},
			Game: Game{
				status: game.InProgress,
				players: map[player.Name]*playerController.Player{
					"fred": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
							{Tile: tile.Tile{ID: 3}, X: 4, Y: 4},
						}),
					},
					"barney": {
						Board: &board.Board{},
					},
				},
				WordValidator: mockWordValidator(func(word string) bool {
					return true
				}),
			},
			wantOk: true,
		},
		{ // user dao error: still want ok, but expect message to be logged
			Message: message.Message{
				PlayerName: "fred",
			},
			Game: Game{
				status: game.InProgress,
				players: map[player.Name]*playerController.Player{
					"fred": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
							{Tile: tile.Tile{ID: 3}, X: 4, Y: 4},
						}),
					},
					"barney": {
						Board: &board.Board{},
					},
				},
				WordValidator: mockWordValidator(func(word string) bool {
					return true
				}),
			},
			wantOk:     true,
			userDaoErr: fmt.Errorf("user dao error"),
		},
	}
	for i, test := range handleGameFinishTests {
		ctx := context.Background()
		gotMessages := make(map[player.Name]struct{}, len(test.Game.players))
		send := func(m message.Message) {
			pn := m.PlayerName
			if _, ok := test.Game.players[pn]; !ok {
				t.Errorf("Test %v: message sent to unknown player: %v", i, m)
			}
			if _, ok := gotMessages[pn]; ok {
				t.Errorf("Test %v: extra message sent to %v: %v", i, pn, m)
			}
			gotMessages[pn] = struct{}{}
			switch {
			case m.Type != message.ChangeGameStatus, m.Game.Status != game.Finished, len(m.Game.FinalBoards) != len(test.players):
				t.Errorf("Test %v: wanted finish message with all final boards sent to %v, got %v", i, pn, m)
			}
		}
		userDaoCalled := false
		var buf bytes.Buffer
		test.Game.log = log.New(&buf, "", 0)
		test.Game.userDao = mockUserDao{
			UpdatePointsIncrementFunc: func(ctx context.Context, userPoints map[string]int) error {
				userDaoCalled = true
				return test.userDaoErr
			},
		}
		err := test.Game.handleGameFinish(ctx, test.Message, send)
		switch {
		case test.wantOk != userDaoCalled:
			t.Errorf("Test %v: wanted user dao to be called to increment points of users", i)
		case (buf.Len() != 0) != (test.userDaoErr != nil):
			t.Errorf("Test %v: wanted log message (%v) if and only if user dao fails (%v)", i, buf.Len() != 0, test.userDaoErr != nil)
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case len(test.Game.players) != len(gotMessages):
			t.Errorf("Test %v: wanted messages sent to all players (%v), got %v", i, len(test.Game.players), len(gotMessages))
		case test.Game.status != game.Finished:
			t.Errorf("Test %v: wanted game to be finished, got %v", i, test.Game.status)
		}
	}
}

func TestHandleGameSnag(t *testing.T) {
	handleGameSnagTests := []struct {
		message.Message
		Game
		wantOk        bool
		wantTilesLeft int
		wantBoards    map[player.Name]*board.Board
	}{
		{}, // game not in progress
		{ // no game unused tiles
			Game: Game{
				status: game.InProgress,
			},
		},
		{ // player board not valid (not all tiles used)
			Message: message.Message{
				PlayerName: "selene",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 2}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New([]tile.Tile{{ID: 1}}, nil),
					},
				},
			},
		},
		{ // player board not valid (not one group)
			Message: message.Message{
				PlayerName: "selene",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 1}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
							{Tile: tile.Tile{ID: 3}, X: 8, Y: 4},
						}),
					},
				},
			},
		},
		{ // player board not valid (word not valid) and CheckOnSnag == true
			Message: message.Message{
				PlayerName: "selene",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 1}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{
							{X: 3, Y: 4},
							{X: 4, Y: 4},
						}),
					},
				},
				WordValidator: mockWordValidator(func(word string) bool {
					return false
				}),
				Config: Config{
					Config: game.Config{
						CheckOnSnag: true,
					},
				},
			},
		},
		{ // player already has tile - this should never happen
			Message: message.Message{
				PlayerName: "selene",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 1}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 1}, X: 3, Y: 4}, // game also has this tile
						}),
					},
				},
				Config: Config{
					ShufflePlayersFunc: func(playerNames []player.Name) {
						// NOOP
					},
				},
			},
		},
		{ // other player already has tile - this should never happen
			Message: message.Message{
				PlayerName: "fred",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 2}, {ID: 3}},
				players: map[player.Name]*playerController.Player{
					"fred": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 1}, X: 3, Y: 4},
						}),
					},
					"barney": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 3}, X: 3, Y: 4}, // game also has this tile
						}),
					},
				},
				Config: Config{
					ShufflePlayersFunc: func(playerNames []player.Name) {
						// NOOP
					},
				},
			},
		},
		{ // all 3 players get tiles
			Message: message.Message{
				PlayerName: "larry",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 4}, {ID: 5}, {ID: 6}, {ID: 7}, {ID: 8}},
				players: map[player.Name]*playerController.Player{
					"larry": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 1}, X: 3, Y: 4},
						}),
					},
					"curly": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
						}),
					},
					"moe": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 3}, X: 3, Y: 4},
						}),
					},
				},
				Config: Config{
					ShufflePlayersFunc: func(playerNames []player.Name) {
						sort.Slice(playerNames, func(i, j int) bool {
							return playerNames[i] > playerNames[j] // order moe before curly
						})
					},
				},
			},
			wantOk:        true,
			wantTilesLeft: 2,
			wantBoards: map[player.Name]*board.Board{
				"larry": board.New([]tile.Tile{{ID: 4}}, nil),
				"curly": board.New([]tile.Tile{{ID: 6}}, nil),
				"moe":   board.New([]tile.Tile{{ID: 5}}, nil),
			},
		},
		{ // 3 players, only 2 get tiles
			Message: message.Message{
				PlayerName: "larry",
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 4}, {ID: 5}},
				players: map[player.Name]*playerController.Player{
					"larry": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 1}, X: 3, Y: 4},
						}),
					},
					"curly": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 2}, X: 3, Y: 4},
						}),
					},
					"moe": {
						Board: board.New(nil, []tile.Position{
							{Tile: tile.Tile{ID: 3}, X: 3, Y: 4},
						}),
					},
				},
				Config: Config{
					ShufflePlayersFunc: func(playerNames []player.Name) {
						sort.Slice(playerNames, func(i, j int) bool {
							return playerNames[i] > playerNames[j] // order moe before curly
						})
					},
				},
			},
			wantOk:        true,
			wantTilesLeft: 0,
			wantBoards: map[player.Name]*board.Board{
				"larry": board.New([]tile.Tile{{ID: 4}}, nil),
				"moe":   board.New([]tile.Tile{{ID: 5}}, nil),
			},
		},
	}
	hasTile := func(tiles []tile.Tile, tID tile.ID) bool {
		for _, tile := range tiles {
			if tile.ID == tID {
				return true
			}
		}
		return false
	}
	for i, test := range handleGameSnagTests {
		ctx := context.Background()
		gotMessages := make(map[player.Name]struct{}, len(test.Game.players))
		send := func(m message.Message) {
			pn := m.PlayerName
			if _, ok := test.Game.players[pn]; !ok {
				t.Errorf("Test %v: message sent to unknown player: %v", i, m)
			}
			if _, ok := gotMessages[pn]; ok {
				t.Errorf("Test %v: extra message sent to %v: %v", i, pn, m)
			}
			gotMessages[pn] = struct{}{}
			switch {
			case test.wantTilesLeft != m.Game.TilesLeft:
				t.Errorf("Test %v: message sent to %v did not have correct tilesLeft: wanted %v, got %v", i, pn, test.wantTilesLeft, m.Game.TilesLeft)
			case test.wantBoards[pn] != nil:
				if !reflect.DeepEqual(test.wantBoards[pn], m.Game.Board) {
					t.Errorf("Test %v: snag board sent to %v not equal:\nwanted: %v\ngot:    %v", i, pn, test.wantBoards[pn], m.Game.Board)
				}
				for tID, tile := range m.Game.Board.UnusedTiles {
					if _, ok := test.Game.players[pn].Board.UnusedTiles[tID]; !ok {
						t.Errorf("Test %v: wanted tile %v added to player's board in game", i, tile)
					}
					if hasTile(test.Game.unusedTiles, tID) {
						t.Errorf("Test %v: player received tileId=%v, but game still has it: %v", i, tID, test.Game.unusedTiles)
					}
				}
			case m.Game.Board != nil && test.wantBoards[pn] != nil:
				t.Errorf("Test %v: wanted no board/tile information sent to player who did not make snag and should not get a tile because none are left: got %v", i, m)
			}
		}
		err := test.Game.handleGameSnag(ctx, test.Message, send)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case len(test.Game.players) != len(gotMessages):
			t.Errorf("Test %v: wanted messages sent to all players (%v), got %v", i, len(test.Game.players), len(gotMessages))
		}
	}
}

func TestHandleGameSwap(t *testing.T) {
	handleGameSwapTests := []struct {
		message.Message
		Game
		wantOk        bool
		wantTilesLeft int
		wantBoard     *board.Board
	}{
		{}, // game not in progress
		{ // no swap tile specified
			Message: message.Message{
				Game: &game.Info{
					Board: &board.Board{},
				},
			},
			Game: Game{
				status: game.InProgress,
			},
		},
		{ // no game unused tiles
			Message: message.Message{
				Game: &game.Info{
					Board: board.New([]tile.Tile{{}}, nil),
				},
			},
			Game: Game{
				status: game.InProgress,
			},
		},
		{ // player does not have tile
			Message: message.Message{
				PlayerName: "alice",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 6}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 7}},
				players: map[player.Name]*playerController.Player{
					"alice": {
						Board: &board.Board{},
					},
				},
			},
		},
		{ // player already has tile (this should never occur)
			Message: message.Message{
				PlayerName: "alice",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 6}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 6}},
				players: map[player.Name]*playerController.Player{
					"alice": {
						Board: board.New([]tile.Tile{{ID: 6}}, nil),
					},
				},
				Config: Config{
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
						// NOOP, expect an error occurs after shuffle
					},
				},
			},
		},
		{ // 4 tiles, 3 players, shuffle by id
			Message: message.Message{
				PlayerName: "shaggy",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 13, Ch: 'D'}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 6, Ch: 'E'}, {ID: 17, Ch: 'B'}, {ID: 8, Ch: 'A'}, {ID: 4, Ch: 'F'}},
				players: map[player.Name]*playerController.Player{
					"shaggy": {
						Board: board.New([]tile.Tile{{ID: 13, Ch: 'D'}}, nil),
					},
					"daphine": nil,
					"fred":    nil,
					"velma":   nil,
				},
				Config: Config{
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
						sort.Slice(tiles, func(i, j int) bool {
							return tiles[i].ID < tiles[j].ID // sort ASC by ID
						})
					},
				},
			},
			wantOk:        true,
			wantTilesLeft: 2,
			wantBoard:     board.New([]tile.Tile{{ID: 4, Ch: 'F'}, {ID: 6, Ch: 'E'}, {ID: 8, Ch: 'A'}}, nil),
		},
		{ // 2 tiles left, 1 player, shuffle alphabetically
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 3, Ch: 'D'}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 6, Ch: 'E'}, {ID: 8, Ch: 'A'}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New([]tile.Tile{{ID: 3, Ch: 'D'}}, nil),
					},
				},
				Config: Config{
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
						sort.Slice(tiles, func(i, j int) bool {
							return tiles[i].Ch < tiles[j].Ch // sort ASC by letter
						})
					},
				},
			},
			wantOk:    true,
			wantBoard: board.New([]tile.Tile{{ID: 8, Ch: 'A'}, {ID: 3, Ch: 'D'}, {ID: 6, Ch: 'E'}}, nil),
		},
		{ // 1 tile left, 1 player.  Because there is only one tile left, the player will get back the tile they swapped because they always get up to three tiles
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 1}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 2}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New([]tile.Tile{{ID: 1}}, nil),
					},
				},
				Config: Config{
					ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {
						sort.Slice(tiles, func(i, j int) bool {
							return tiles[i].ID > tiles[j].ID // sort DESC
						})
					},
				},
			},
			wantOk:    true,
			wantBoard: board.New([]tile.Tile{{ID: 2}, {ID: 1}}, nil),
		},
	}
	hasTile := func(tiles []tile.Tile, tID tile.ID) bool {
		for _, tile := range tiles {
			if tile.ID == tID {
				return true
			}
		}
		return false
	}
	for i, test := range handleGameSwapTests {
		ctx := context.Background()
		gotMessages := make(map[player.Name]struct{}, len(test.Game.players))
		send := func(m message.Message) {
			pn := m.PlayerName
			if _, ok := test.Game.players[pn]; !ok {
				t.Errorf("Test %v: message sent to unknown player: %v", i, m)
			}
			if _, ok := gotMessages[pn]; ok {
				t.Errorf("Test %v: extra message sent to %v: %v", i, pn, m)
			}
			gotMessages[pn] = struct{}{}
			switch {
			case test.wantTilesLeft != m.Game.TilesLeft:
				t.Errorf("Test %v: message sent to %v did not have correct tilesLeft: wanted %v, got %v", i, pn, test.wantTilesLeft, m.Game.TilesLeft)
			case pn == test.Message.PlayerName: // the player making the swap
				if !reflect.DeepEqual(test.wantBoard, m.Game.Board) {
					t.Errorf("Test %v: newly swapped tile not equal:\nwanted: %v\ngot:    %v", i, test.wantBoard, m.Game.Board)
				}
				for tID, tile := range m.Game.Board.UnusedTiles {
					if _, ok := test.Game.players[pn].Board.UnusedTiles[tID]; !ok {
						t.Errorf("Test %v: wanted tile %v added to player's board in game", i, tile)
					}
					if hasTile(test.Game.unusedTiles, tID) {
						t.Errorf("Test %v: player received tileId=%v, but game still has it: %v", i, tID, test.Game.unusedTiles)
					}
				}
			case m.Game.Board != nil:
				t.Errorf("Test %v: wanted no board/tile information sent to player who did not make swap: got %v", i, m)
			}
		}
		err := test.Game.handleGameSwap(ctx, test.Message, send)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case len(test.Game.players) != len(gotMessages):
			t.Errorf("Test %v: wanted messages sent to all players (%v), got %v", i, len(test.Game.players), len(gotMessages))
		}
	}
}

func TestHandleInfoChanged(t *testing.T) {
	want := message.Message{
		Type: message.GameInfos,
		Game: &game.Info{
			ID:        7,
			Status:    game.InProgress,
			Players:   []string{"barney", "fred"},
			CreatedAt: 555,
			Capacity:  7,
		},
	}
	g := Game{
		id:     7,
		status: game.InProgress,
		players: map[player.Name]*playerController.Player{
			"fred":   nil,
			"barney": nil,
		},
		createdAt: 555,
		Config: Config{
			MaxPlayers: 7,
		},
	}
	var got message.Message
	send := func(m message.Message) {
		got = m
	}
	g.handleInfoChanged(send)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("messages not equal for game %v:\nwanted: %v\ngot:    %v", g, want, got)
	}
}

func TestHandleGameTilesMoved(t *testing.T) {
	handleGameTilesMovedTests := []struct {
		game.Status
		message.Message
		*board.Board
		wantOk bool
		want   *board.Board
	}{
		{}, // game not in progress
		{
			Status: game.InProgress,
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Board: board.New(nil, []tile.Position{
						{Tile: tile.Tile{ID: 8}, X: 7, Y: 4},
					}),
				},
			},
			Board: &board.Board{
				Config:       board.Config{NumRows: 10, NumCols: 10},
				UsedTiles:    map[tile.ID]tile.Position{8: {Tile: tile.Tile{ID: 8}, X: 17, Y: 3}},
				UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{17: {3: {ID: 8}}},
			},
			wantOk: true,
			want: &board.Board{
				Config:       board.Config{NumRows: 10, NumCols: 10},
				UsedTiles:    map[tile.ID]tile.Position{8: {Tile: tile.Tile{ID: 8}, X: 7, Y: 4}},
				UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{7: {4: {ID: 8}}},
			},
		},
	}
	for i, test := range handleGameTilesMovedTests {
		g := Game{
			status: test.Status,
			players: map[player.Name]*playerController.Player{
				test.Message.PlayerName: {
					Board: test.Board,
				},
			},
		}
		ctx := context.Background()
		send := func(m message.Message) {
			t.Errorf("Test %v: unwanted message sent: %v", i, m)
		}
		err := g.handleGameTilesMoved(ctx, test.Message, send)
		got := g.players[test.Message.PlayerName].Board
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v: boards not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestHandleBoardRefresh(t *testing.T) {
	wantType := message.JoinGame // messages with types other than message.RefreshGameBoard may call this
	g := Game{
		players: map[player.Name]*playerController.Player{
			"fred": {
				Board: &board.Board{},
			},
		},
	}
	ctx := context.Background()
	m := message.Message{
		Type:       wantType,
		PlayerName: "fred",
		Game: &game.Info{
			Board: &board.Board{},
		},
	}
	var got message.Message
	send := func(m message.Message) {
		got = m
	}
	err := g.handleBoardRefresh(ctx, m, send)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case wantType != got.Type:
		t.Errorf("types not equal: wanted %v, got %v", wantType, got.Type)
	}
}

func TestHandleGameChat(t *testing.T) {
	from := "scooby"
	secret := "he's wearing a mask!"
	playerNames := []player.Name{"shaggy", "velma", "fred", "daphine"}
	players := make(map[player.Name]*playerController.Player, len(playerNames))
	playersAwaitingChat := make(map[player.Name]struct{}, len(playerNames))
	for _, pn := range playerNames {
		players[pn] = nil
		playersAwaitingChat[pn] = struct{}{}
	}
	g := Game{
		players: players,
	}
	ctx := context.Background()
	m := message.Message{
		PlayerName: player.Name(from),
		Info:       secret,
	}
	send := func(m message.Message) {
		_, ok := playersAwaitingChat[m.PlayerName]
		switch {
		case !ok:
			t.Errorf("message sent to unknown player or to player more than once: %v", m)
		case m.Type != message.GameChat:
			t.Errorf("wanted chat message, got %v", m.Type)
		case !strings.Contains(m.Info, from):
			t.Errorf("wanted message info to contain who the chat is from (%v), got %v", from, m.Info)
		case !strings.Contains(m.Info, secret):
			t.Errorf("wanted message info to contain the chat message (%v), got %v", secret, m.Info)
		default:
			delete(playersAwaitingChat, m.PlayerName)
		}
	}
	g.handleGameChat(ctx, m, send)
	if len(playersAwaitingChat) != 0 {
		t.Errorf("wanted chat message sent to all players, %v didn't receive it", len(playersAwaitingChat))
	}
}

func TestResizeBoard(t *testing.T) {
	barneyBoard := &board.Board{
		UnusedTileIDs: []tile.ID{2},
	}
	resizeBoardTests := []struct {
		message.Message
		playerBoard     board.Board
		gameConfig      game.Config
		gameID          game.ID
		gameStatus      game.Status
		gameUnusedTiles []tile.Tile
		want            message.Message
		wantInfo        bool
	}{
		{ // no board change, check message
			Message: message.Message{
				Type:       message.RefreshGameBoard,
				Info:       "x",
				PlayerName: "fred",
				Game: &game.Info{
					Board: board.New(nil, nil),
				},
			},
			gameConfig: game.Config{
				Penalize: true,
			},
			gameID:          7,
			gameStatus:      game.InProgress,
			gameUnusedTiles: []tile.Tile{{}, {}, {}},
			want: message.Message{
				Type:       message.RefreshGameBoard,
				Info:       "",
				PlayerName: "fred",
				Game: &game.Info{
					Board:     board.New(nil, nil),
					TilesLeft: 3,
					Status:    game.InProgress,
					Players:   []string{"barney", "fred"},
					ID:        7,
					Config: &game.Config{
						Penalize: true,
					},
				},
			},
		},
		{ // board much smaller
			Message: message.Message{
				PlayerName: "fred",
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 5,
							NumCols: 5,
						},
					},
				},
			},
			playerBoard: *board.New(nil, []tile.Position{
				{
					Tile: tile.Tile{ID: 1},
					X:    100,
					Y:    100,
				},
			}),
			want: message.Message{
				PlayerName: "fred",
				Game: &game.Info{
					Board:   board.New([]tile.Tile{{ID: 1}}, nil),
					Players: []string{"barney", "fred"},
					Config:  &game.Config{},
				},
			},
			wantInfo: true,
		},
		{ // finished game
			Message: message.Message{
				PlayerName: "fred",
				Game: &game.Info{
					Board: board.New(nil, nil),
				},
			},
			gameStatus: game.Finished,
			want: message.Message{
				PlayerName: "fred",
				Game: &game.Info{
					Board:   board.New(nil, nil),
					Status:  game.Finished,
					Players: []string{"barney", "fred"},
					Config:  &game.Config{},
					FinalBoards: map[string]board.Board{
						"fred":   {},
						"barney": *barneyBoard,
					},
				},
			},
		},
	}
	for i, test := range resizeBoardTests {
		g := Game{
			id: test.gameID,
			Config: Config{
				Config: test.gameConfig,
			},
			players: map[player.Name]*playerController.Player{
				"fred": {
					Board: &test.playerBoard,
				},
				"barney": {
					Board: barneyBoard,
				},
			},
			status:      test.gameStatus,
			unusedTiles: test.gameUnusedTiles,
		}
		got, err := g.resizeBoard(test.Message)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.wantInfo != (len(got.Info) > 0):
			t.Errorf("Test %v: wanted info (%v), got: '%v", i, test.wantInfo, got.Info)
		default:
			got.Info = ""
			if !reflect.DeepEqual(test.want, *got) {
				t.Errorf("Test %v: resize board messages not equal:\nwanted: %v\ngot:    %v", i, test.want, *got)
			}
		}
	}
}

func TestPlayerFinalBoards(t *testing.T) {
	b1 := board.Board{
		UnusedTileIDs: []tile.ID{1},
	}
	b2 := board.Board{
		UnusedTileIDs: []tile.ID{2},
	}
	g := Game{
		players: map[player.Name]*playerController.Player{
			"fred":   {Board: &b1},
			"barney": {Board: &b2},
		},
	}
	want := map[string]board.Board{
		"fred":   b1,
		"barney": b2,
	}
	got := g.playerFinalBoards()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("final boards not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}
