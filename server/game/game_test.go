package game

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
)

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
