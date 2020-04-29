package game

import (
	"reflect"
	"sort"
	"testing"
)

func TestInitializeUnusedTiles_correctAmount(t *testing.T) {
	g := game{}
	g.initializeUnusedTiles()
	want := 144
	got := len(g.unusedTiles)
	if want != got {
		t.Errorf("wanted %v tiles, but got %v", want, got)
	}
}

func TestInitializeUnusedTiles_allLetters(t *testing.T) {
	g := game{}
	g.initializeUnusedTiles()
	m := make(map[letter]bool, 26)
	for _, v := range g.unusedTiles {
		ch := v.Ch
		if ch < 'A' || ch > 'Z' {
			t.Errorf("invalid tile: %v", v)
		}
		m[ch] = true
	}
	want := 26
	got := len(m)
	if want != got {
		t.Errorf("wanted %v different letters, but got %v", want, got)
	}
}

func TestInitializeUnusedTiles_shuffled(t *testing.T) {
	createTilesShuffledTests := []struct {
		want      letter
		inReverse string
	}{
		{'A', ""},
		{'Z', " IN REVERSE"},
	}
	for _, test := range createTilesShuffledTests {
		g := game{
			shuffleUnusedTilesFunc: func(tiles []tile) {
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
		got := g.unusedTiles[0].Ch
		if test.want != got {
			t.Errorf("expected first tile to be %q when sorted%v (a fake shuffle), but was %q", test.want, test.inReverse, got)
		}
	}
}

func TestInitializeUnusedTiles_uniqueIds(t *testing.T) {
	g := game{}
	g.initializeUnusedTiles()
	tileIds := make(map[int]bool, len(g.unusedTiles))
	for _, tile := range g.unusedTiles {
		if _, ok := tileIds[tile.ID]; ok {
			t.Errorf("tile id %v repeated", tile.ID)
		}
		tileIds[tile.ID] = true
	}
}

func TestInitializeUnusedTiles_custom(t *testing.T) {
	tileLetters := "selene"
	g := game{tileLetters: tileLetters}
	g.initializeUnusedTiles()
	for i, tile := range g.unusedTiles {
		want := letter(tileLetters[i])
		got := tile.Ch
		if want != got {
			t.Errorf("wanted %v tiles, but got %v", want, got)
		}
	}
}

func TestUsedWords(t *testing.T) {
	usedWordsTests := []struct {
		usedTiles    map[int]tilePosition
		usedTileLocs map[int]map[int]tile
		want         []string
	}{
		{
			usedTiles: map[int]tilePosition{
				5: {Tile: tile{ID: 5, Ch: 'A'}, X: 2, Y: 7},
				4: {Tile: tile{ID: 4, Ch: 'B'}, X: 2, Y: 8},
				7: {Tile: tile{ID: 5, Ch: 'C'}, X: 2, Y: 10},
				3: {Tile: tile{ID: 4, Ch: 'D'}, X: 2, Y: 11},
			},
			usedTileLocs: map[int]map[int]tile{
				2: {
					7:  {ID: 5, Ch: 'A'},
					8:  {ID: 4, Ch: 'B'},
					10: {ID: 7, Ch: 'C'},
					11: {ID: 3, Ch: 'D'},
				},
			},
			want: []string{"AB", "CD"},
		},
		{
			usedTiles: map[int]tilePosition{
				5: {Tile: tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile{ID: 4, Ch: 'B'}, X: 8, Y: 2},
			},
			usedTileLocs: map[int]map[int]tile{
				7: {
					2: {ID: 5, Ch: 'A'},
				},
				8: {
					2: {ID: 4, Ch: 'B'},
				},
			},
			want: []string{"AB"},
		},
		{
			usedTiles: map[int]tilePosition{
				8: {Tile: tile{ID: 8, Ch: 'N'}, X: 4, Y: 3},
				7: {Tile: tile{ID: 7, Ch: 'A'}, X: 5, Y: 3},
				4: {Tile: tile{ID: 4, Ch: 'P'}, X: 6, Y: 3},
				9: {Tile: tile{ID: 9, Ch: 'O'}, X: 4, Y: 4},
				1: {Tile: tile{ID: 1, Ch: 'R'}, X: 5, Y: 4},
				2: {Tile: tile{ID: 2, Ch: 'E'}, X: 5, Y: 5},
			},
			usedTileLocs: map[int]map[int]tile{
				4: {
					3: {ID: 8, Ch: 'N'},
					4: {ID: 9, Ch: 'O'},
				},
				5: {
					3: {ID: 7, Ch: 'A'},
					4: {ID: 1, Ch: 'R'},
					5: {ID: 2, Ch: 'E'},
				},
				6: {
					3: {ID: 4, Ch: 'P'},
				},
			},
			want: []string{"NAP", "OR", "NO", "ARE"},
		},
		{
			want: []string{},
		},
	}
	for i, test := range usedWordsTests {
		gps := gamePlayerState{
			usedTiles:    test.usedTiles,
			usedTileLocs: test.usedTileLocs,
		}
		got := gps.usedWords()
		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestSingleUsedGroup(t *testing.T) {
	singleUsedGroupTests := []struct {
		usedTiles    map[int]tilePosition
		usedTileLocs map[int]map[int]tile
		want         bool
	}{
		{
			usedTiles: map[int]tilePosition{
				5: {Tile: tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile{ID: 4, Ch: 'B'}, X: 7, Y: 3},
			},
			usedTileLocs: map[int]map[int]tile{
				7: {
					2: {ID: 5, Ch: 'A'},
					3: {ID: 4, Ch: 'B'},
				},
			},
			want: true,
		},
		{
			usedTiles: map[int]tilePosition{
				5: {Tile: tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile{ID: 4, Ch: 'B'}, X: 7, Y: 4},
			},
			usedTileLocs: map[int]map[int]tile{
				7: {
					2: {ID: 5, Ch: 'A'},
					4: {ID: 4, Ch: 'B'},
				},
			},
			want: false,
		},
		{},
	}
	for i, test := range singleUsedGroupTests {
		gps := gamePlayerState{
			usedTiles:    test.usedTiles,
			usedTileLocs: test.usedTileLocs,
		}
		got := gps.singleUsedGroup()
		if test.want != got {
			t.Errorf("Test %v: wanted: %v, got: %v", i, test.want, got)
		}
	}
}
