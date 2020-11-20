package lobby

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	gameController "github.com/jacobpatterson1549/selene-bananas/server/game"
)

func TestNew(t *testing.T) {
	newLobbyTests := []struct {
		log        *log.Logger
		maxGames   int
		maxSockets int
		wantOk     bool
	}{
		{},
		{
			log: log.New(ioutil.Discard, "test", log.LstdFlags),
		},
		{
			log:      log.New(ioutil.Discard, "test", log.LstdFlags),
			maxGames: 4,
		},
		{
			log:        log.New(ioutil.Discard, "test", log.LstdFlags),
			maxGames:   2,
			maxSockets: 16,
			wantOk:     true,
		},
	}
	for i, test := range newLobbyTests {
		cfg := Config{
			Log:        test.log,
			MaxGames:   test.maxGames,
			MaxSockets: test.maxSockets,
		}
		l, err := cfg.NewLobby()
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case test.maxGames != l.maxGames,
			test.maxSockets != l.maxSockets,
			l.upgrader == nil:
			t.Errorf("Test %v: values not initialized properly", i)
		}
	}
}

func TestAddUser(t *testing.T) {
	var w http.ResponseWriter
	var r http.Request
	addUserTests := []struct {
		username  string
		resultErr error
	}{
		{},
		{
			username: "fred",
		},
		{
			username:  "dino",
			resultErr: errors.New("humans only"),
		},
	}
	for i, test := range addUserTests {
		l := Lobby{
			addSockets: make(chan playerSocket, 1),
		}
		go func() {
			ps := <-l.addSockets
			if string(ps.playerName) != test.username {
				t.Errorf("Test %v: wanted player name to be %v, got %v", i, test.username, ps.playerName)
			}
			ps.result <- test.resultErr
		}()
		gotErr := l.AddUser(test.username, w, &r)
		if test.resultErr != gotErr {
			t.Errorf("Test %v: wanted error '%v', got '%v'", i, test.resultErr, gotErr)
		}
	}
}

func TestRemoveUser(t *testing.T) {
	username := "barney"
	l := Lobby{
		socketMessages: make(chan game.Message, 1),
	}
	l.RemoveUser(username)
	m := <-l.socketMessages
	switch {
	case m.Type != game.PlayerDelete:
		t.Errorf("wanted player delete messageType (%v), got %v", game.PlayerDelete, m.Type)
	case string(m.PlayerName) != username:
		t.Errorf("wanted playerName %v in message, got %v", username, m.PlayerName)
	}
}

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (d mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return d.UpdatePointsIncrementFunc(ctx, userPoints)
}

func TestCreateGame(t *testing.T) {
	log := log.New(ioutil.Discard, "test", log.LstdFlags)
	mockUserDao := mockUserDao{
		UpdatePointsIncrementFunc: func(ctx context.Context, userPoints map[string]int) error {
			return errors.New("unexpected call")
		},
	}
	gameCfg := gameController.Config{
		Log:                    log,
		MaxPlayers:             1,
		NumNewTiles:            9,
		UserDao:                mockUserDao,
		IdlePeriod:             8,
		TimeFunc:               func() int64 { return 7 },
		ShuffleUnusedTilesFunc: func(tiles []tile.Tile) {},
		ShufflePlayersFunc:     func(playerNames []player.Name) {},
	}
	l := Lobby{
		maxGames: 1,
		gameCfg:  gameCfg,
		games:    make(map[game.ID]gameMessageHandler, 1),
	}
	m := game.Message{
		BoardConfig: &board.Config{NumRows: 23, NumCols: 21},
		WordsConfig: &game.WordsConfig{CheckOnSnag: true, Penalize: true, MinLength: 3, FinishedAllowMove: true},
	}
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	l.createGame(ctx, m)
	wc := game.WordsConfig{}
	switch {
	case l.gameCfg.WordsConfig != wc:
		t.Errorf("creating a game unwantedly stored the game's config in the lobby")
	case len(l.games) != 1:
		t.Errorf("wanted 1 game, got %v", len(l.games))
	}
	for _, gmh := range l.games {
		gmh.CancelFunc()
	}
}
