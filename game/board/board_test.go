package board

import (
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestNewFromConfig(t *testing.T) {
	newFromConfigTests := []struct {
		Config
		wantOk bool
	}{
		{},
		{ // rows too small
			Config: Config{
				NumRows: 10,
				NumCols: 3,
			},
		},
		{ // cols too small
			Config: Config{
				NumRows: 10,
				NumCols: 3,
			},
		},
		{ // happy path (with minimum size)
			Config: Config{
				NumCols: 10,
				NumRows: 10,
			},
			wantOk: true,
		},
	}
	for i, test := range newFromConfigTests {
		unusedTiles := []tile.Tile{{ID: 7}}
		b, err := test.Config.New(unusedTiles)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error creating new board", i)
			}
		case err != nil:
			t.Fatalf("Test %v: unwanted error: %v", i, err)
		case len(b.UnusedTileIDs) != 1, b.UnusedTiles[7].ID != 7:
			t.Errorf("Test %v: wanted only unused tile to have id 7, got %v", i, b.UnusedTiles)
		case len(b.UnusedTileIDs) != 1, b.UnusedTileIDs[0] != 7:
			t.Errorf("Test %v: wanted only unused tile id to be 7, got %v", i, b.UnusedTileIDs)
		case len(b.UsedTiles) != 0:
			t.Errorf("Test %v: wanted no used tiles, got %v", i, b.UsedTiles)
		case len(b.UsedTileLocs) != 0:
			t.Errorf("Test %v: wanted no used tiles locs, got %v", i, b.UsedTileLocs)
		}
	}
}

func TestAddTile(t *testing.T) {
	b := Board{
		UnusedTiles:   make(map[tile.ID]tile.Tile),
		UnusedTileIDs: make([]tile.ID, 0, 1),
		UsedTiles:     make(map[tile.ID]tile.Position),
		UsedTileLocs:  make(map[tile.X]map[tile.Y]tile.Tile),
		Config: Config{
			NumCols: 1,
			NumRows: 1,
		},
	}
	tl := tile.Tile{ID: 1}
	err := b.AddTile(tl)
	if err != nil {
		t.Errorf("unwanted error adding tile: %v", err)
	}
	err = b.AddTile(tl)
	if err == nil {
		t.Errorf("unwanted error while adding tile that TileState already has")
	}
	tp := tile.Position{Tile: tl}
	err = b.MoveTiles(map[tile.ID]tile.Position{tl.ID: tp})
	if err != nil {
		t.Errorf("unwanted error moving tile: %v", err)
	}
	err = b.AddTile(tl)
	if err == nil {
		t.Errorf("wanted error when re-adding tile")
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
			want: make([]string, 0),
		},
		{
			usedTiles: map[tile.ID]tile.Position{
				4: {Tile: tile.Tile{ID: 4}, X: 1, Y: 2},
				5: {Tile: tile.Tile{ID: 5}, X: 1, Y: 3},
			},
			usedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
				1: {
					2: {ID: 4},
					3: {ID: 5},
				},
			},
			want: []string{"\u0000\u0000"},
		},
	}
	for i, test := range usedWordsTests {
		b := Board{
			UsedTiles:    test.usedTiles,
			UsedTileLocs: test.usedTileLocs,
		}
		got := b.UsedTileWords()
		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("Test %v: used words not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestCanBeFinished(t *testing.T) {
	canBeFinishedTests := []struct {
		unusedTiles  map[tile.ID]tile.Tile
		usedTiles    map[tile.ID]tile.Position
		usedTileLocs map[tile.X]map[tile.Y]tile.Tile
		want         bool
	}{
		{ // no groups
			want: true,
		},
		{ // no groups, but not all tiles used
			unusedTiles: map[tile.ID]tile.Tile{1: {}},
			want:        false,
		},
		{ // one group
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
		{ // two groups of one tile
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
		{ // one larger group
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
	}
	for i, test := range canBeFinishedTests {
		b := Board{
			UnusedTiles:  test.unusedTiles,
			UsedTiles:    test.usedTiles,
			UsedTileLocs: test.usedTileLocs,
			Config: Config{
				NumCols: 5,
				NumRows: 5,
			},
		}
		got := b.CanBeFinished()
		if test.want != got {
			t.Errorf("Test %v boardCanBeFinished not equal: wanted: %v, got: %v", i, test.want, got)
		}
	}
}

func TestMoveTiles(t *testing.T) {
	t.Run("swap", func(t *testing.T) {
		b := Board{
			UnusedTiles:   make(map[tile.ID]tile.Tile),
			UnusedTileIDs: make([]tile.ID, 0, 2),
			UsedTiles:     make(map[tile.ID]tile.Position),
			UsedTileLocs:  make(map[tile.X]map[tile.Y]tile.Tile),
			Config: Config{
				NumCols: 3,
				NumRows: 3,
			},
		}
		t1 := tile.Tile{ID: 1}
		t2 := tile.Tile{ID: 2}
		b.AddTile(t1)
		b.AddTile(t2)
		b.MoveTiles(map[tile.ID]tile.Position{
			t1.ID: {Tile: t1, X: 1, Y: 1},
			t2.ID: {Tile: t2, X: 2, Y: 2},
		})
		b.MoveTiles(map[tile.ID]tile.Position{
			t1.ID: {Tile: t1, X: 2, Y: 2},
			t2.ID: {Tile: t2, X: 1, Y: 1},
		})
		want, got := 2, len(b.UsedTileLocs)
		if want != got {
			t.Errorf("number of used tiles not equal after swap: wanted %v, got %v", want, got)
		}
	})
	t.Run("errorChecks", func(t *testing.T) {
		errorCheckTests := []struct {
			tilePositions []tile.Position
			board         Board
			wantOk        bool
			want          Board
		}{
			{ // hasTile == false
				tilePositions: []tile.Position{{Tile: tile.Tile{ID: 1}}},
			},
			{ // tiles move to same position
				tilePositions: []tile.Position{
					{Tile: tile.Tile{ID: 1}},
					{Tile: tile.Tile{ID: 2}},
				},
				board: Board{
					UnusedTiles: map[tile.ID]tile.Tile{
						1: {ID: 1},
						2: {ID: 2},
					},
					UnusedTileIDs: []tile.ID{1, 2},
				},
			},
			{ // tile already at desired location
				tilePositions: []tile.Position{{Tile: tile.Tile{ID: 1}, X: 2, Y: 3}},
				board: Board{
					UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
					UnusedTileIDs: []tile.ID{1},
					UsedTiles:     map[tile.ID]tile.Position{4: {Tile: tile.Tile{ID: 4}, X: 2, Y: 3}},
					UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{2: {3: {ID: 4}}},
				},
			},
			{ // tile is moved off board (numRows = 10)
				tilePositions: []tile.Position{{Tile: tile.Tile{ID: 1}, X: 2, Y: 99}},
				board: Board{
					UnusedTiles: map[tile.ID]tile.Tile{
						1: {ID: 1},
					},
					UnusedTileIDs: []tile.ID{1},
				},
			},
			{
				tilePositions: []tile.Position{{Tile: tile.Tile{ID: 1}, X: 2, Y: 3}},
				board: Board{
					UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
					UnusedTileIDs: []tile.ID{1},
					UsedTiles:     map[tile.ID]tile.Position{2: {Tile: tile.Tile{ID: 2}, X: 2, Y: 4}},
					UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{2: {4: {ID: 2}}},
				},
				want: Board{
					UsedTiles: map[tile.ID]tile.Position{
						1: {Tile: tile.Tile{ID: 1}, X: 2, Y: 3},
						2: {Tile: tile.Tile{ID: 1}, X: 2, Y: 4},
					},
					UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
						2: {
							3: {ID: 1},
							4: {ID: 2},
						},
					},
				},
				wantOk: true,
			},
		}
		for i, test := range errorCheckTests {
			test.board.Config = Config{
				NumCols: 10,
				NumRows: 10,
			}
			tilePositionsM := make(map[tile.ID]tile.Position, len(test.tilePositions))
			for _, tp := range test.tilePositions {
				tilePositionsM[tp.Tile.ID] = tp
			}
			err := test.board.MoveTiles(tilePositionsM)
			switch {
			case !test.wantOk:
				if err == nil {
					t.Errorf("Test %v: wanted error moving tiles", i)
				}
			case err != nil:
				t.Errorf("Test %v: unwanted error moving tiles: %v", i, err)
			}
		}
	})
}

func TestRemoveTile(t *testing.T) {
	removeTileTests := []struct {
		removeID tile.ID
		board    Board
		want     Board
		wantOk   bool
	}{
		{},
		{
			removeID: 1,
			board: Board{
				UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
				UnusedTileIDs: []tile.ID{1},
			},
			wantOk: true,
		},
		{
			removeID: 2,
			board: Board{
				UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
				UnusedTileIDs: []tile.ID{1},
				UsedTiles:     map[tile.ID]tile.Position{2: {Tile: tile.Tile{ID: 2}, X: 8, Y: 9}},
				UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{8: {9: {ID: 2}}},
			},
			want: Board{
				UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
				UnusedTileIDs: []tile.ID{1},
			},
			wantOk: true,
		},
		{
			removeID: 3,
			board: Board{
				UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}, 5: {ID: 5}, 3: {ID: 3}},
				UnusedTileIDs: []tile.ID{1, 3, 5},
			},
			want: Board{
				UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}, 5: {ID: 5}},
				UnusedTileIDs: []tile.ID{1, 5},
			},
			wantOk: true,
		},
	}
	for i, test := range removeTileTests {
		tile := tile.Tile{
			ID: test.removeID,
		}
		err := test.board.RemoveTile(tile)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error removing tile", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error removing tile: %v", i, err)
		case !(reflect.DeepEqual(test.want.UnusedTiles, test.board.UnusedTiles) || (test.want.UnusedTiles == nil && len(test.board.UnusedTiles) == 0)),
			!(reflect.DeepEqual(test.want.UnusedTileIDs, test.board.UnusedTileIDs) || (test.want.UnusedTileIDs == nil && len(test.board.UnusedTileIDs) == 0)),
			!(reflect.DeepEqual(test.want.UsedTiles, test.board.UsedTiles) || (test.want.UsedTiles == nil && len(test.board.UsedTiles) == 0)),
			!(reflect.DeepEqual(test.want.UsedTileLocs, test.board.UsedTileLocs) || (test.want.UsedTileLocs == nil && len(test.board.UsedTileLocs) == 0)):
			t.Errorf("Test %v: board not in desired state after tile removal\nwanted %v\ngot    %v", i, test.want, test.board)
		}
	}
}

func TestResize(t *testing.T) {
	resizeTests := []struct {
		deltaNumCols    int
		deltaNumRows    int
		wantTile2Unused bool
	}{
		{}, // do not change board, add board to message
		{
			deltaNumCols:    -10,
			wantTile2Unused: true,
		},
		{
			deltaNumCols: 10,
		},
		{
			deltaNumRows:    -7,
			wantTile2Unused: true,
		},
		{
			deltaNumRows: -4,
		},
	}
	t1 := tile.Tile{
		ID: 1,
		Ch: 'A',
	}
	t2 := tile.Tile{
		ID: 2,
		Ch: 'B',
	}
	t3 := tile.Tile{
		ID: 3,
		Ch: 'C',
	}
	unusedTiles := []tile.Tile{
		t1,
		t2,
		t3,
	}
	tilePositions := []tile.Position{
		{
			Tile: t1,
			X:    1,
			Y:    1,
		},
		{
			Tile: t2,
			X:    15,
			Y:    5,
		},
		{
			Tile: t3,
			X:    2,
			Y:    1,
		},
	}
	tilePositionsM := make(map[tile.ID]tile.Position, len(tilePositions))
	for _, tp := range tilePositions {
		tilePositionsM[tp.Tile.ID] = tp
	}
	for i, test := range resizeTests {
		cfg := Config{
			NumCols: 20,
			NumRows: 10,
		}
		b, err := cfg.New(unusedTiles)
		if err != nil {
			t.Errorf("Test %v: unwanted error creating board to resize: %v", i, err)
		}
		if err = b.MoveTiles(tilePositionsM); err != nil {
			t.Errorf("Test %v: unwanted error moving tiles before board resize: %v", i, err)
		}
		cfg.NumCols += test.deltaNumCols
		cfg.NumRows += test.deltaNumRows
		m, err := b.Resize(cfg)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error resizing board: %v", i, err)
		case b.Config.NumCols != cfg.NumCols, b.Config.NumRows != cfg.NumRows:
			t.Errorf("resizing should update max board dimensions, wanted %v, got %v", cfg, b)
		case test.wantTile2Unused:
			switch {
			case len(b.UnusedTileIDs) != 1, b.UnusedTileIDs[0] != t2.ID,
				len(m.Tiles) != 1, m.Tiles[0].ID != t2.ID:
				t.Errorf("Test %v: wanted tile 2 to be moved back to the unused area now that the board is more narrow/short", i)
			case len(m.Info) == 0:
				t.Errorf("Test %v: wanted info about board resize", i)
			}
		default:
			switch {
			case len(b.UnusedTileIDs) != 0, len(m.Tiles) != 0:
				t.Errorf("Test %v: wanted no unused tiles", i)
			case len(m.Info) != 0:
				t.Errorf("Test %v: wanted no info about board resize", i)
			}
		}
	}
}
