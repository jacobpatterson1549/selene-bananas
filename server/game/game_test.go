package game

import (
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
		var wordChecker mockWordChecker
		var userDao mockUserDao
		got, err := test.Config.NewGame(testLog, id, wordChecker, userDao)
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
			!reflect.DeepEqual(wordChecker, got.wordChecker),
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
		errCheckTests := []struct {
			Config
			*log.Logger
			game.ID
			WordChecker
			UserDao
			wantOk bool
		}{
			{}, // no log
			{ // id  not positive
				Logger: testLog,
			},
			{ // no word checker
				Logger: testLog,
				ID:     1,
			},
			{ // no user dao
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
			},
			{ // no time func
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
			},
			{ // low maxPlayers
				Config: Config{
					TimeFunc: timeFunc,
				},
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
			},
			{ // low num newTiles
				Config: Config{
					TimeFunc:   timeFunc,
					MaxPlayers: 4,
				},
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
			},
			{ // low idle period
				Config: Config{
					TimeFunc:    timeFunc,
					MaxPlayers:  4,
					NumNewTiles: 16,
				},
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
			},
			{ // missing shuffle tiles func
				Config: Config{
					TimeFunc:    timeFunc,
					MaxPlayers:  4,
					NumNewTiles: 16,
					IdlePeriod:  1 * time.Hour,
				},
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
			},
			{ // missing shuffle players func
				Config: Config{
					TimeFunc:               timeFunc,
					MaxPlayers:             4,
					NumNewTiles:            16,
					IdlePeriod:             1 * time.Hour,
					ShuffleUnusedTilesFunc: shuffleUnusedTilesFunc,
				},
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
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
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
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
				Logger:      testLog,
				ID:          1,
				WordChecker: mockWordChecker{},
				UserDao:     mockUserDao{},
				wantOk:      true,
			},
		}
		for i, test := range errCheckTests {
			err := test.Config.validate(test.Logger, test.ID, test.WordChecker, test.UserDao)
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
			test.Config.validate(log, 1, mockWordChecker{}, mockUserDao{}) // Ignore the error.  This test doesn't care about it.
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
	t.Run("TestRunSyncMessageHandlers", func(t *testing.T) {
		t.Skip() // TODO: add tests for all expected message types, expect gameWarning for game not being started for most
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
		{"A", ""},
		{"Z", " IN REVERSE"},
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
		wantErr     bool
		want        []tile.Tile
	}{
		{
			tileLetters: ":(",
			wantErr:     true,
		},
		{
			tileLetters: "AAABAC",
			want: []tile.Tile{
				{ID: 6, Ch: "C"},
				{ID: 4, Ch: "B"},
				{ID: 1, Ch: "A"},
				{ID: 2, Ch: "A"},
				{ID: 3, Ch: "A"},
				{ID: 5, Ch: "A"},
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
		case test.wantErr:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
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
	ud := mockUserDao{
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
		userDao: ud,
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
					Board: board.New([]tile.Tile{{ID: 13, Ch: "D"}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 6, Ch: "E"}, {ID: 17, Ch: "B"}, {ID: 8, Ch: "A"}, {ID: 4, Ch: "F"}},
				players: map[player.Name]*playerController.Player{
					"shaggy": {
						Board: board.New([]tile.Tile{{ID: 13, Ch: "D"}}, nil),
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
			wantBoard:     board.New([]tile.Tile{{ID: 4, Ch: "F"}, {ID: 6, Ch: "E"}, {ID: 8, Ch: "A"}}, nil),
		},
		{ // 2 tiles left, 1 player, shuffle alphabetically
			Message: message.Message{
				PlayerName: "selene",
				Game: &game.Info{
					Board: board.New([]tile.Tile{{ID: 3, Ch: "D"}}, nil),
				},
			},
			Game: Game{
				status:      game.InProgress,
				unusedTiles: []tile.Tile{{ID: 6, Ch: "E"}, {ID: 8, Ch: "A"}},
				players: map[player.Name]*playerController.Player{
					"selene": {
						Board: board.New([]tile.Tile{{ID: 3, Ch: "D"}}, nil),
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
			wantBoard: board.New([]tile.Tile{{ID: 8, Ch: "A"}, {ID: 3, Ch: "D"}, {ID: 6, Ch: "E"}}, nil),
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
		gotMessages := make(map[player.Name]message.Message, len(test.Game.players))
		send := func(m message.Message) {
			if _, ok := test.Game.players[m.PlayerName]; !ok {
				t.Errorf("Test %v: message sent to unknown player: %v", i, m)
			}
			if _, ok := gotMessages[m.PlayerName]; ok {
				t.Errorf("Test %v: extra message sent to %v: %v", i, m.PlayerName, m)
			}
			gotMessages[m.PlayerName] = m
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
		default:
			for pn, p := range test.Game.players {
				pM := gotMessages[pn]
				switch {
				case test.wantTilesLeft != pM.Game.TilesLeft:
					t.Errorf("Test %v: message sent to %v did not have correct tilesLeft: wanted %v, got %v", i, pn, test.wantTilesLeft, pM.Game.TilesLeft)
				case pn == test.Message.PlayerName: // the player making the swap
					if !reflect.DeepEqual(test.wantBoard, pM.Game.Board) {
						t.Errorf("Test %v: newly swapped tile not equal:\nwanted: %v\ngot:    %v", i, test.wantBoard, pM.Game.Board)
					}
					for tID, tile := range pM.Game.Board.UnusedTiles {
						if _, ok := p.Board.UnusedTiles[tID]; !ok {
							t.Errorf("Test %v: wanted tile %v added to player's board in game", i, tile)
						}
						if hasTile(test.Game.unusedTiles, tID) {
							t.Errorf("Test %v: player recieved tileId=%v, but game still has it: %v", i, tID, test.Game.unusedTiles)
						}
					}
				case pM.Game.Board != nil:
					t.Errorf("Test %v: wanted no board/tile information sent to player who did not make swap: got %v", i, pM)
				}
			}
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
						{Tile: tile.Tile{ID: 8, Ch: "D"}, X: 7, Y: 4},
					}),
				},
			},
			Board: &board.Board{
				Config:       board.Config{NumRows: 10, NumCols: 10},
				UsedTiles:    map[tile.ID]tile.Position{8: {Tile: tile.Tile{ID: 8, Ch: "D"}, X: 17, Y: 3}},
				UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{17: {3: {ID: 8, Ch: "D"}}},
			},
			wantOk: true,
			want: &board.Board{
				Config:       board.Config{NumRows: 10, NumCols: 10},
				UsedTiles:    map[tile.ID]tile.Position{8: {Tile: tile.Tile{ID: 8, Ch: "D"}, X: 7, Y: 4}},
				UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{7: {4: {ID: 8, Ch: "D"}}},
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
					Tile: tile.Tile{ID: 1, Ch: "A"},
					X:    100,
					Y:    100,
				},
			}),
			want: message.Message{
				PlayerName: "fred",
				Game: &game.Info{
					Board:   board.New([]tile.Tile{{ID: 1, Ch: "A"}}, nil),
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
