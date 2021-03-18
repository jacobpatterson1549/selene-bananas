// +build js,wasm

package canvas

import (
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestDrawTile(t *testing.T) {
	drawTileID := tile.ID(1)
	drawTileTests := []struct {
		s             selection // has moveState
		fromSelection bool
		wantDrawn     bool
	}{
		{ // draw a tile if fromSelection == false and moveState != drag
			wantDrawn: true,
		},
		{ // draw a tile if tiles from the selection are being drawn an the tile is in the selection
			s: selection{
				tiles: map[tile.ID]tileSelection{
					drawTileID: {},
				},
			},
			fromSelection: true,
			wantDrawn:     true,
		},
		{ // do NOT draw a tile if tiles from the selection are being drawn an the tile is NOT in the selection
			s: selection{
				tiles: map[tile.ID]tileSelection{},
			},
			fromSelection: true,
			wantDrawn:     false,
		},
		{ // draw a tile if the user is dragging other tiles (the tile is not in the selection of tiles being dragged)
			s: selection{
				tiles:     map[tile.ID]tileSelection{},
				moveState: drag,
			},
			fromSelection: false,
			wantDrawn:     true,
		},
		{ // do NOT draw a tile if the user is dragging it and it's being drawn at the original location (fromSelection=false)
			s: selection{
				tiles: map[tile.ID]tileSelection{
					drawTileID: {},
				},
				moveState: drag,
			},
			fromSelection: false,
			wantDrawn:     false,
		},
	}
	var gotDrawn bool

	ctx := mockContext{
		FillTextFunc: func(text string, x, y int) {
			gotDrawn = true
		},
		StrokeRectFunc: func(x, y, width, height int) {
			// not tested
		},
		SetFillColorFunc: func(name string) {
			// not tested
		},
		FillRectFunc: func(x, y, width, height int) {
			// not tested
		},
	}
	for i, test := range drawTileTests {
		gotDrawn = false
		c := Canvas{
			ctx:       &ctx,
			selection: test.s,
		}
		tile := tile.Tile{
			ID: drawTileID,
		}
		c.drawTile(0, 0, tile, test.fromSelection)
		if test.wantDrawn != gotDrawn {
			t.Errorf("Test %v: wanted tile to be drawn: %v", i, test.wantDrawn)
		}
	}
}

func TestCalculateSelectedUnusedTiles(t *testing.T) {
	ta := tile.Tile{
		ID: 11,
		Ch: 'A',
	}
	tb := tile.Tile{
		ID: 22,
		Ch: 'B',
	}
	tc := tile.Tile{
		ID: 33,
		Ch: 'C',
	}
	c := Canvas{
		draw: drawMetrics{
			tileLength: 2,
			unusedMin: pixelPosition{
				x: 1,
				y: 8,
			},
		},
		board: &board.Board{
			UnusedTiles: map[tile.ID]tile.Tile{
				ta.ID: ta,
				tb.ID: tb,
				tc.ID: tc,
			},
			UnusedTileIDs: []tile.ID{
				ta.ID,
				tb.ID,
				tc.ID,
			},
		},
	}
	minX := 4
	minY := 9
	maxX := 4
	maxY := 9
	want := map[tile.ID]tileSelection{
		tb.ID: {
			used:  false,
			tile:  tb,
			index: 1,
		},
	}
	got := c.calculateSelectedUnusedTiles(minX, maxX, minY, maxY)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestSwap(t *testing.T) {
	st := tile.Tile{
		ID: 8,
	}
	messageSent := false
	c := Canvas{
		board: board.New([]tile.Tile{st}, nil),
		selection: selection{
			end: pixelPosition{
				x: 1,
				y: 1,
			},
			tiles: map[tile.ID]tileSelection{
				8: {
					tile: st,
				},
			},
		},
		draw: drawMetrics{
			tileLength: 2,
		},
		Socket: mockSocket{
			SendFunc: func(m message.Message) {
				switch {
				case m.Type != message.SwapGameTile, m.Game.Board.UnusedTileIDs[0] != st.ID:
					t.Errorf("wanted message to swap tile 8, got: %v", m)
				}
				messageSent = true
			},
		},
	}
	c.swap()
	switch {
	case len(c.board.UnusedTiles) != 0:
		t.Error("wanted tile to be swapped")
	case !messageSent:
		t.Error("wanted message to be sent")
	case len(c.selection.tiles) != 0:
		t.Error("wanted no selected tiles after swap")
	}
}
