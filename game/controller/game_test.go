package controller

import (
	"sort"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestInitializeUnusedTilesCorrectAmount(t *testing.T) {
	g := Game{
		tileLetters: defaultTileLetters,
	}
	g.initializeUnusedTiles()
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
	g.initializeUnusedTiles()
	var e struct{}
	m := make(map[rune]struct{}, 26)
	for _, v := range g.unusedTiles {
		ch := rune(v.Ch)
		if ch < 'A' || ch > 'Z' {
			t.Errorf("invalid tile: %v", v)
		}
		m[ch] = e
	}
	want := 26
	got := len(m)
	if want != got {
		t.Errorf("wanted %v different letters, but got %v", want, got)
	}
}

func TestInitializeUnusedTilesShuffled(t *testing.T) {
	createTilesShuffledTests := []struct {
		want      rune
		inReverse string
	}{
		{'A', ""},
		{'Z', " IN REVERSE"},
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
		g.initializeUnusedTiles()
		got := rune(g.unusedTiles[0].Ch)
		if test.want != got {
			t.Errorf("expected first tile to be %q when sorted%v (a fake shuffle), but was %q", test.want, test.inReverse, got)
		}
	}
}

func TestInitializeUnusedTilesUniqueIds(t *testing.T) {
	g := Game{}
	g.initializeUnusedTiles()
	var e struct{}
	tileIDs := make(map[tile.ID]struct{}, len(g.unusedTiles))
	for _, tile := range g.unusedTiles {
		if _, ok := tileIDs[tile.ID]; ok {
			t.Errorf("tile id %v repeated", tile.ID)
		}
		tileIDs[tile.ID] = e
	}
}

func TestInitializeUnusedTilesCustom(t *testing.T) {
	tileLetters := "SELENE"
	g := Game{tileLetters: tileLetters}
	g.initializeUnusedTiles()
	for i, tile := range g.unusedTiles {
		want := rune(tileLetters[i])
		got := rune(tile.Ch)
		if want != got {
			t.Errorf("wanted %v tiles, but got %v", want, got)
		}
	}
}
