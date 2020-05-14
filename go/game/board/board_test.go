package board

import (
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

func TestNew(t *testing.T) {
	b := New([]tile.Tile{{ID: 1}})
	if len(b.UnusedTileIDs) != 1 || b.UnusedTiles[1].ID != 1 {
		t.Errorf("wanted only unused tile to have id 1, got %v", b.UnusedTiles)
	}
	if len(b.UnusedTileIDs) != 1 || b.UnusedTileIDs[0] != 1 {
		t.Errorf("wanted only unused tile id to be 1, got %v", b.UnusedTileIDs)
	}
	if len(b.UsedTiles) != 0 {
		t.Errorf("wanted no used tiles, got %v", b.UsedTiles)
	}
	if len(b.UsedTileLocs) != 0 {
		t.Errorf("wanted no used tiles locs, got %v", b.UsedTileLocs)
	}
}

func TestAddTile(t *testing.T) {
	b := New([]tile.Tile{})
	tl := tile.Tile{ID: 1}
	err := b.AddTile(tl)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = b.AddTile(tl)
	if err == nil {
		t.Errorf("unexpected error while adding tile that TileState already has")
	}
	tp := tile.Position{Tile: tl}
	err = b.MoveTiles([]tile.Position{tp})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = b.AddTile(tl)
	if err == nil {
		t.Errorf("unexpected while adding tile that TileState has moved")
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
		b := Board{
			UsedTiles:    test.usedTiles,
			UsedTileLocs: test.usedTileLocs,
		}
		got := b.UsedTileWords()
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
		b := Board{
			UsedTiles:    test.usedTiles,
			UsedTileLocs: test.usedTileLocs,
		}
		got := b.HasSingleUsedGroup()
		if test.want != got {
			t.Errorf("Test %v: wanted: %v, got: %v", i, test.want, got)
		}
	}
}
