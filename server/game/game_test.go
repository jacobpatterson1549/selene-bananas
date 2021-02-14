package game

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
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
		tileLetters         string
		wantNum             int
		checkAllLettersUsed bool
		wantErr             bool
	}{
		{
			tileLetters:         defaultTileLetters,
			wantNum:             144,
			checkAllLettersUsed: true,
		},
		{
			tileLetters: "AAAABBABACCABAC",
		},
		{
			tileLetters: "SELENE",
		},
		{
			tileLetters: ":(",
			wantErr:     true,
		},
	}
	for i, test := range initializeUnusedTilesTests {
		g := Game{
			Config: Config{
				TileLetters: test.tileLetters,
			},
		}
		err := g.initializeUnusedTiles()
		if test.wantNum == 0 {
			test.wantNum = len(test.tileLetters)
		}
		switch {
		case test.wantErr:
			if err == nil {
				t.Errorf("Test %v: wanted error", i)
			}
		case test.wantNum != len(g.unusedTiles):
			t.Errorf("wanted %v tiles, but got %v", test.wantNum, len(g.unusedTiles))
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case test.checkAllLettersUsed:
			m := make(map[rune]struct{}, 26)
			for _, v := range g.unusedTiles {
				ch := rune(v.Ch[0])
				if ch < 'A' || ch > 'Z' {
					t.Errorf("invalid tile: %v", v)
				}
				m[ch] = struct{}{}
			}
			want := 26
			got := len(m)
			if want != got {
				t.Errorf("wanted %v different letters, but got %v", want, got)
			}
		default:
			// unique ids:
			tileIDs := make(map[tile.ID]struct{}, len(g.unusedTiles))
			for _, tile := range g.unusedTiles {
				if _, ok := tileIDs[tile.ID]; ok {
					t.Errorf("tile id %v repeated", tile.ID)
				}
				tileIDs[tile.ID] = struct{}{}
			}
			// Letter char:
			for i, ti := range g.unusedTiles {
				want := tile.Letter(test.tileLetters[i : i+1])
				got := ti.Ch
				if want != got {
					t.Errorf("wanted %v tiles, but got %v", want, got)
				}
			}
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
		UserDao: ud,
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
