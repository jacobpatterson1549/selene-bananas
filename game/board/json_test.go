package board

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

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

func TestMarshal(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
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
			t.Errorf("not equal:\nwanted: %v\ngot:    %s", want, got)
		}
	})
	t.Run("orderedTiles", func(t *testing.T) {
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
			t.Errorf("not equal:\nwanted: %v\ngot:    %s", want, got)
		}
	})
}

func TestUnmarshal(t *testing.T) {
	unmarshalTests := []struct {
		j      string
		wantOk bool
		want   Board
	}{
		{
			j: `{"tiles":"NOT_AN_ARRAY"}`,
		},
		{
			j:      `{"tiles":[{"id":1,"ch":"A"}],"tilePositions":[{"t":{"id":2,"ch":"B"},"x":3,"y":4}],"config":{"r":17,"c":22}}`,
			wantOk: true,
			want: Board{
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
			},
		},
	}
	for i, test := range unmarshalTests {
		var got Board
		err := json.Unmarshal([]byte(test.j), &got)
		switch {
		case !test.wantOk:
			if err == nil {
				t.Errorf("Test %v: wanted error when unmarshalling bad json", i)
			}
		case err != nil:
			t.Errorf("Test %v: unwanted error: %v", i, err)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v: not equal:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}
