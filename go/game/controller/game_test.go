package controller

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

func TestInitializeUnusedTiles_correctAmount(t *testing.T) {
	g := Game{}
	g.initializeUnusedTiles()
	want := 144
	got := len(g.unusedTiles)
	if want != got {
		t.Errorf("wanted %v tiles, but got %v", want, got)
	}
}

func TestInitializeUnusedTiles_allLetters(t *testing.T) {
	g := Game{}
	g.initializeUnusedTiles()
	m := make(map[rune]bool, 26)
	for _, v := range g.unusedTiles {
		ch := rune(v.Ch)
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
		want      rune
		inReverse string
	}{
		{'A', ""},
		{'Z', " IN REVERSE"},
	}
	for _, test := range createTilesShuffledTests {
		g := Game{
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

func TestInitializeUnusedTiles_uniqueIds(t *testing.T) {
	g := Game{}
	g.initializeUnusedTiles()
	tileIDs := make(map[tile.ID]bool, len(g.unusedTiles))
	for _, tile := range g.unusedTiles {
		if _, ok := tileIDs[tile.ID]; ok {
			t.Errorf("tile id %v repeated", tile.ID)
		}
		tileIDs[tile.ID] = true
	}
}

func TestInitializeUnusedTiles_custom(t *testing.T) {
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

func TestUsedWords(t *testing.T) {
	usedWordsTests := []struct {
		usedTiles    map[tile.ID]tile.Position
		usedTileLocs map[tile.X]map[tile.Y]tile.Tile
		want         []string
	}{
		{
			usedTiles: map[tile.ID]tile.Position{
				5: {Tile: tile.Tile{ID: 5, Ch: 'A'}, X: 2, Y: 7},
				4: {Tile: tile.Tile{ID: 4, Ch: 'B'}, X: 2, Y: 8},
				7: {Tile: tile.Tile{ID: 5, Ch: 'C'}, X: 2, Y: 10},
				3: {Tile: tile.Tile{ID: 4, Ch: 'D'}, X: 2, Y: 11},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
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
			usedTiles: map[tile.ID]tile.Position{
				5: {Tile: tile.Tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile.Tile{ID: 4, Ch: 'B'}, X: 8, Y: 2},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
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
			usedTiles: map[tile.ID]tile.Position{
				8: {Tile: tile.Tile{ID: 8, Ch: 'N'}, X: 4, Y: 3},
				7: {Tile: tile.Tile{ID: 7, Ch: 'A'}, X: 5, Y: 3},
				4: {Tile: tile.Tile{ID: 4, Ch: 'P'}, X: 6, Y: 3},
				9: {Tile: tile.Tile{ID: 9, Ch: 'O'}, X: 4, Y: 4},
				1: {Tile: tile.Tile{ID: 1, Ch: 'R'}, X: 5, Y: 4},
				2: {Tile: tile.Tile{ID: 2, Ch: 'E'}, X: 5, Y: 5},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
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
			// CON
			// A
			// RUT
			usedTiles: map[tile.ID]tile.Position{
				1: {Tile: tile.Tile{ID: 1, Ch: 'C'}, X: 1, Y: 1},
				2: {Tile: tile.Tile{ID: 2, Ch: 'O'}, X: 2, Y: 1},
				3: {Tile: tile.Tile{ID: 3, Ch: 'N'}, X: 3, Y: 1},
				4: {Tile: tile.Tile{ID: 4, Ch: 'A'}, X: 1, Y: 2},
				5: {Tile: tile.Tile{ID: 5, Ch: 'R'}, X: 1, Y: 3},
				6: {Tile: tile.Tile{ID: 6, Ch: 'U'}, X: 2, Y: 3},
				7: {Tile: tile.Tile{ID: 7, Ch: 'T'}, X: 3, Y: 3},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
				1: {
					1: {ID: 1, Ch: 'C'},
					2: {ID: 4, Ch: 'A'},
					3: {ID: 5, Ch: 'R'},
				},
				2: {
					1: {ID: 2, Ch: 'O'},
					3: {ID: 6, Ch: 'U'},
				},
				3: {
					1: {ID: 3, Ch: 'N'},
					3: {ID: 7, Ch: 'T'},
				},
			},
			want: []string{"CON", "RUT", "CAR"},
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
		usedTiles    map[tile.ID]tile.Position
		usedTileLocs map[tile.X]map[tile.Y]tile.Tile
		want         bool
	}{
		{
			usedTiles: map[tile.ID]tile.Position{
				5: {Tile: tile.Tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile.Tile{ID: 4, Ch: 'B'}, X: 7, Y: 3},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
				7: {
					2: {ID: 5, Ch: 'A'},
					3: {ID: 4, Ch: 'B'},
				},
			},
			want: true,
		},
		{
			usedTiles: map[tile.ID]tile.Position{
				5: {Tile: tile.Tile{ID: 5, Ch: 'A'}, X: 7, Y: 2},
				4: {Tile: tile.Tile{ID: 4, Ch: 'B'}, X: 7, Y: 4},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
				7: {
					2: {ID: 5, Ch: 'A'},
					4: {ID: 4, Ch: 'B'},
				},
			},
			want: false,
		},
		{
			usedTiles: map[tile.ID]tile.Position{
				1: {Tile: tile.Tile{ID: 1, Ch: 'C'}, X: 1, Y: 1},
				2: {Tile: tile.Tile{ID: 2, Ch: 'O'}, X: 2, Y: 1},
				3: {Tile: tile.Tile{ID: 3, Ch: 'N'}, X: 3, Y: 1},
				4: {Tile: tile.Tile{ID: 4, Ch: 'A'}, X: 1, Y: 2},
				5: {Tile: tile.Tile{ID: 5, Ch: 'R'}, X: 1, Y: 3},
				6: {Tile: tile.Tile{ID: 6, Ch: 'U'}, X: 2, Y: 3},
				7: {Tile: tile.Tile{ID: 7, Ch: 'T'}, X: 3, Y: 3},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
				1: {
					1: {ID: 1, Ch: 'C'},
					2: {ID: 4, Ch: 'A'},
					3: {ID: 5, Ch: 'R'},
				},
				2: {
					1: {ID: 2, Ch: 'O'},
					3: {ID: 6, Ch: 'U'},
				},
				3: {
					1: {ID: 3, Ch: 'N'},
					3: {ID: 7, Ch: 'T'},
				},
			},
			want: true,
		},
		{},
	}
	for i, test := range singleUsedGroupTests {
		// var buffer bytes.Buffer
		// log := log.New(&buffer, "", log.LstdFlags)
		// log.SetOutput(os.Stdout)
		gps := gamePlayerState{
			// player:       &player{},
			// log:          log,
			usedTiles:    test.usedTiles,
			usedTileLocs: test.usedTileLocs,
		}
		got := gps.singleUsedGroup()
		if test.want != got {
			t.Errorf("Test %v: wanted: %v, got: %v", i, test.want, got)
		}
	}
}
