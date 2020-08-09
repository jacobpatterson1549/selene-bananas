package game

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
)

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (ud mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return ud.UpdatePointsIncrementFunc(ctx, userPoints)
}

func TestInitializeUnusedTilesCorrectAmount(t *testing.T) {
	g := Game{
		tileLetters: defaultTileLetters,
	}
	if err := g.initializeUnusedTiles(); err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	want := 144
	got := len(g.unusedTiles)
	if want != got {
		t.Errorf("wanted %v tiles, but got %v", want, got)
	}
}

func TestInitializeUnusedTilesAllLetters(t *testing.T) {
	g := Game{
		tileLetters: defaultTileLetters,
	}
	if err := g.initializeUnusedTiles(); err != nil {
		t.Errorf("unwanted error: %v", err)
	}
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
			tileLetters: "AZ",
			shuffleUnusedTilesFunc: func(tiles []tile.Tile) {
				sort.Slice(tiles, func(i, j int) bool {
					lessThan := tiles[i].Ch < tiles[j].Ch
					if len(test.inReverse) > 0 {
						return !lessThan
					}
					return lessThan
				})
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

func TestInitializeUnusedTilesUniqueIds(t *testing.T) {
	tileLetters := "AAAABBABACCABAC"
	g := Game{tileLetters: tileLetters}
	if err := g.initializeUnusedTiles(); err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	tileIDs := make(map[tile.ID]struct{}, len(g.unusedTiles))
	for _, tile := range g.unusedTiles {
		if _, ok := tileIDs[tile.ID]; ok {
			t.Errorf("tile id %v repeated", tile.ID)
		}
		tileIDs[tile.ID] = struct{}{}
	}
}

func TestInitializeUnusedTilesCustom(t *testing.T) {
	tileLetters := "SELENE"
	g := Game{tileLetters: tileLetters}
	if err := g.initializeUnusedTiles(); err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	for i, ti := range g.unusedTiles {
		want := tile.Letter(tileLetters[i : i+1])
		got := ti.Ch
		if want != got {
			t.Errorf("wanted %v tiles, but got %v", want, got)
		}
	}
}

func TestInitializeUnusedTilesInvalid(t *testing.T) {
	tileLetters := ":("
	g := Game{tileLetters: tileLetters}
	if err := g.initializeUnusedTiles(); err == nil {
		t.Errorf("wanted error while initializing tiles with text: '%v'", tileLetters)
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
