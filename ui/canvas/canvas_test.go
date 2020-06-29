// +build js,wasm

package canvas

import (
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

type (
	mockContext struct {
		SetFontFunc        func(name string)
		SetLineWidthFunc   func(width float64)
		SetFillColorFunc   func(name string)
		SetStrokeColorFunc func(name string)
		SetOpacityFunc     func(fraction string)
		FillTextFunc       func(text string, x, y int)
		ClearRectFunc      func(x, y, width, height int)
		FillRectFunc       func(x, y, width, height int)
		StrokeRectFunc     func(x, y, width, height int)
	}
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
		StrokeRectFunc:   func(x, y, width, height int) {},
		SetFillColorFunc: func(name string) {},
		FillRectFunc:     func(x, y, width, height int) {},
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
	ta, _ := tile.New(tile.ID(11), 'A')
	tb, _ := tile.New(tile.ID(22), 'B')
	tc, _ := tile.New(tile.ID(33), 'C')
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
				ta.ID: *ta,
				tb.ID: *tb,
				tc.ID: *tc,
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
			tile:  *tb,
			index: 1,
		},
	}
	got := c.calculateSelectedUnusedTiles(minX, maxX, minY, maxY)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal\nwanted: %v\ngot:    %v", want, got)
	}
}

func (ctx *mockContext) SetFont(name string) {
	ctx.SetFontFunc(name)
}

func (ctx *mockContext) SetLineWidth(width float64) {
	ctx.SetLineWidthFunc(width)
}

func (ctx *mockContext) SetFillColor(name string) {
	ctx.SetFillColorFunc(name)
}

func (ctx *mockContext) SetStrokeColor(name string) {
	ctx.SetStrokeColorFunc(name)
}

func (ctx *mockContext) FillText(text string, x, y int) {
	ctx.FillTextFunc(text, x, y)
}

func (ctx *mockContext) ClearRect(x, y, width, height int) {
	ctx.ClearRectFunc(x, y, width, height)
}

func (ctx *mockContext) FillRect(x, y, width, height int) {
	ctx.FillRectFunc(x, y, width, height)
}

func (ctx *mockContext) StrokeRect(x, y, width, height int) {
	ctx.StrokeRectFunc(x, y, width, height)
}
