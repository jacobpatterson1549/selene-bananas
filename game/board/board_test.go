package board

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestNew(t *testing.T) {
	cfg := Config{
		NumCols: 10,
		NumRows: 10,
	}
	b, err := cfg.New([]tile.Tile{{ID: 1}})
	if err != nil || b == nil {
		t.Fatalf("unwanted error: %v", err)
	}
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

func TestNewFromBoard(t *testing.T) {
	tiles := []tile.Tile{
		{
			ID: 1,
			Ch: 'A',
		},
	}
	tilePositions := []tile.Position{
		{
			X: 3,
			Y: 4,
			Tile: tile.Tile{
				ID: 2,
				Ch: 'B',
			},
		},
	}
	want := &Board{
		UnusedTiles: map[tile.ID]tile.Tile{
			1: {
				ID: 1,
				Ch: 'A',
			},
		},
		UnusedTileIDs: []tile.ID{
			1,
		},
		UsedTiles: map[tile.ID]tile.Position{
			2: {
				Tile: tile.Tile{
					ID: 2,
					Ch: 'B',
				},
				X: 3,
				Y: 4,
			},
		},
		UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
			3: {
				4: {
					ID: 2,
					Ch: 'B',
				},
			},
		},
	}
	got := New(tiles, tilePositions)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestNewInvalidBoards(t *testing.T) {
	invalidBoardConfigs := []Config{
		{},
		{
			NumRows: -1,
		},
		{
			NumRows: 125,
			NumCols: 8,
		},
	}
	for i, cfg := range invalidBoardConfigs {
		_, err := cfg.New(nil)
		if err == nil {
			t.Errorf("Test %v: wanted error", i)
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
		t.Errorf("unwanted error: %v", err)
	}
	err = b.AddTile(tl)
	if err == nil {
		t.Errorf("unwanted error while adding tile that TileState already has")
	}
	tp := tile.Position{Tile: tl}
	err = b.MoveTiles(map[tile.ID]tile.Position{tl.ID: tp})
	if err != nil {
		t.Errorf("unwanted error: %v", err)
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
			want: []string{},
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
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
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
			t.Errorf("Test %v: wanted: %v, got: %v", i, test.want, got)
		}
	}
}

func TestMoveTilesSwap(t *testing.T) {
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
		t.Errorf("wanted %v, got %v", want, got)
	}
}

// boardsEqualForTesting is a test function that determines if boards are equal by allowing the nils in fields for the wanted board,
// but only if the other board has that field with a zero length value
func boardsEqualForTesting(want, got Board) bool {
	return (reflect.DeepEqual(want.UnusedTiles, got.UnusedTiles) || (want.UnusedTiles == nil && len(got.UnusedTiles) == 0)) ||
		(reflect.DeepEqual(want.UnusedTileIDs, got.UnusedTileIDs) || (want.UnusedTileIDs == nil && len(got.UnusedTileIDs) == 0)) ||
		(reflect.DeepEqual(want.UsedTiles, got.UsedTiles) || (want.UsedTiles == nil && len(got.UsedTiles) == 0)) ||
		(reflect.DeepEqual(want.UsedTileLocs, got.UsedTileLocs) || (want.UsedTileLocs == nil && len(got.UsedTileLocs) == 0))
}

func TestMoveTiles(t *testing.T) {
	moveTilesErrTests := []struct {
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
				UsedTiles:     map[tile.ID]tile.Position{},
				UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{},
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
	for i, test := range moveTilesErrTests {
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
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		}
	}
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
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
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
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
		if err = b.MoveTiles(tilePositionsM); err != nil {
			t.Errorf("Test %v: unwanted error: %v", i, err)
		}
		cfg.NumCols += test.deltaNumCols
		cfg.NumRows += test.deltaNumRows
		m, err := b.Resize(cfg)
		switch {
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
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

func TestMarshal(t *testing.T) {
	b := Board{
		UnusedTiles: map[tile.ID]tile.Tile{
			1: {
				ID: 1,
				Ch: 'A',
			},
		},
		UnusedTileIDs: []tile.ID{
			1,
		},
		UsedTiles: map[tile.ID]tile.Position{
			2: {
				Tile: tile.Tile{
					ID: 2,
					Ch: 'B',
				},
				X: 3,
				Y: 4,
			},
		},
		UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
			3: {
				4: {
					ID: 2,
					Ch: 'B',
				},
			},
		},
		Config: Config{
			NumRows: 17,
			NumCols: 22,
		},
	}
	want := `{"tiles":[{"id":1,"ch":"A"}],"tilePositions":[{"t":{"id":2,"ch":"B"},"x":3,"y":4}],"config":{"r":17,"c":22}}`
	got, err := json.Marshal(b)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case want != string(got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, string(got))
	}
}

func TestMarshalOrderedTiles(t *testing.T) {
	t1 := tile.Tile{ID: 1, Ch: 'A'}
	t2 := tile.Tile{ID: 2, Ch: 'B'}
	t3 := tile.Tile{ID: 3, Ch: 'C'}
	t4 := tile.Tile{ID: 4, Ch: 'D'}
	t5 := tile.Tile{ID: 5, Ch: 'E'}
	t6 := tile.Tile{ID: 6, Ch: 'F'}
	t7 := tile.Tile{ID: 7, Ch: 'G'}
	t8 := tile.Tile{ID: 8, Ch: 'H'}
	t9 := tile.Tile{ID: 9, Ch: 'I'}
	b := Board{
		UsedTiles: map[tile.ID]tile.Position{
			2: {Tile: t2, X: 2, Y: 3},
			3: {Tile: t3, X: 3, Y: 2},
			1: {Tile: t1, X: 1, Y: 4},
			5: {Tile: t5, X: 2, Y: 2},
			6: {Tile: t6, X: 3, Y: 1},
			4: {Tile: t4, X: 2, Y: 1},
		},
		UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
			3: {
				1: t6,
				6: t3,
			},
			1: {
				4: t1,
			},
			2: {
				1: t4,
				3: t2,
				2: t5,
			},
		},
		UnusedTiles: map[tile.ID]tile.Tile{
			7: t7,
			9: t9,
			8: t8,
		},
		UnusedTileIDs: []tile.ID{
			9,
			7,
			8,
		},
	}
	unusedTiles := "" + // The tiles should be ordered by the UnusedTilesIDs array.
		`{"id":9,"ch":"I"},` +
		`{"id":7,"ch":"G"},` +
		`{"id":8,"ch":"H"}`
	usedTiles := "" + // The tiles should be ordered by x position then y position, not id or letter.
		`{"t":{"id":1,"ch":"A"},"x":1,"y":4},` +
		`{"t":{"id":4,"ch":"D"},"x":2,"y":1},` +
		`{"t":{"id":5,"ch":"E"},"x":2,"y":2},` +
		`{"t":{"id":2,"ch":"B"},"x":2,"y":3},` +
		`{"t":{"id":6,"ch":"F"},"x":3,"y":1},` +
		`{"t":{"id":3,"ch":"C"},"x":3,"y":2}`
	want := `{"tiles":[` + unusedTiles + `],"tilePositions":[` + usedTiles + `]}`
	got, err := json.Marshal(b)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case want != string(got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, string(got))
	}
}

func TestUnMarshal(t *testing.T) {
	j := `{"tiles":[{"id":1,"ch":"A"}],"tilePositions":[{"t":{"id":2,"ch":"B"},"x":3,"y":4}],"config":{"r":17,"c":22}}`
	want := Board{
		UnusedTiles: map[tile.ID]tile.Tile{
			1: {
				ID: 1,
				Ch: 'A',
			},
		},
		UnusedTileIDs: []tile.ID{
			1,
		},
		UsedTiles: map[tile.ID]tile.Position{
			2: {
				Tile: tile.Tile{
					ID: 2,
					Ch: 'B',
				},
				X: 3,
				Y: 4,
			},
		},
		UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{
			3: {
				4: {
					ID: 2,
					Ch: 'B',
				},
			},
		},
		Config: Config{
			NumRows: 17,
			NumCols: 22,
		},
	}
	var got Board
	err := json.Unmarshal([]byte(j), &got)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestMarshalBadJson(t *testing.T) {
	j := `{"tiles":"NOT_AN_ARRAY"}`
	var got Board
	err := json.Unmarshal([]byte(j), &got)
	if err == nil {
		t.Error("wanted error when unmarshaling bad json")
	}
}
