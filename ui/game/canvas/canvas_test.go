//go:build js && wasm

package canvas

import (
	"reflect"
	"strings"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestSetGameStatus(t *testing.T) {
	c := Canvas{
		gameStatus: game.NotStarted,
		selection: selection{
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
			},
			tiles: map[tile.ID]tileSelection{1: {}},
		},
	}
	want := game.InProgress
	c.SetGameStatus(want)
	switch {
	case c.gameStatus != want:
		t.Errorf("game status not set: wanted %v, got %v", want, c.gameStatus)
	case c.selection.moveState != none:
		t.Errorf("wanted moveState to be none (%v), got %v", none, c.selection.moveState)
	case len(c.selection.tiles) != 0:
		t.Errorf("wanted selection tiles to be cleared")
	}
}

func TestDrawErrorMessage(t *testing.T) {
	fillColorSet, textFilled := false, false
	c := Canvas{
		errorColor: "deep_red",
		ctx: &mockContext{
			SetFillColorFunc: func(name string) {
				if want, got := "deep_red", name; want != got {
					t.Errorf("wanted error color to be %v, got %v", want, got)
				}
				fillColorSet = true
			},
			FillTextFunc: func(text string, x, y int) {
				textFilled = true
			},
		},
	}
	c.drawErrorMessage("any message")
	switch {
	case !fillColorSet:
		t.Errorf("fill color not set")
	case !textFilled:
		t.Errorf("next not drawn")
	}
}

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
			fromSelection: true,
			wantDrawn:     false,
		},
		{ // draw a tile if the user is dragging other tiles (the tile is not in the selection of tiles being dragged)
			s: selection{
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

func TestCanMove(t *testing.T) {
	tests := map[game.Status]bool{
		game.NotStarted: false,
		game.InProgress: true,
		game.Finished:   true,
		game.Deleted:    false,
	}
	for s, want := range tests {
		c := Canvas{
			gameStatus: s,
		}
		got := c.canMove()
		if want != got {
			t.Errorf("wanted canvas.canMove() to be %v when game Status is %v", want, s)
		}
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

func TestSetMoveState(t *testing.T) {
	tests := []struct {
		ms   moveState
		want string
	}{
		{none, "none"},
		{swap, "swap"},
		{rect, "rect"},
		{drag, "drag"},
		{grab, "grab"},
	}
	for i, test := range tests {
		setCheckedCalled := false
		c := Canvas{
			selection: selection{
				dom: &mockDOM{
					SetCheckedFunc: func(query string, checked bool) {
						if !strings.HasSuffix(query, test.want) {
							t.Errorf("Test %v: wanted query to end with %v: %v", i, test.want, query)
						}
						if !checked {
							t.Errorf("Test %v: wanted checked to be true for query: %v", i, query)
						}
						setCheckedCalled = true
					},
				},
			},
		}
		c.selection.setMoveState(test.ms)
		if !setCheckedCalled {
			t.Errorf("Test %v: wanted dom.SetChecked to be called to set the move state", i)
		}
	}
}

func TestPixelPositionFromMouse(t *testing.T) {
	parent := pixelPosition{
		x: 1,
		y: 2,
	}
	event := js.ValueOf(map[string]interface{}{
		"offsetX": 4,
		"offsetY": 8,
	})
	child := parent.fromMouse(event)
	want := pixelPosition{
		x: 4,
		y: 8,
	}
	switch {
	case child != want:
		t.Errorf("child values incorrect: wanted %v, got %v", want, child)
	case parent != want:
		t.Errorf("parent should also be modified: wanted %v, got %v", want, parent)
	}
}

func TestPixelPositionFromTouch(t *testing.T) {
	parent := pixelPosition{
		x: 9,
		y: 8,
	}
	defaultPrevented := false
	preventDefault := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defaultPrevented = true
		return nil
	})
	getBoundingClientRect := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return map[string]interface{}{
			"left": 100,
			"top":  500,
		}
	})
	event := js.ValueOf(map[string]interface{}{
		"preventDefault": preventDefault,
		"touches": []interface{}{
			map[string]interface{}{
				"clientX": 125,
				"clientY": 672,
			},
		},
		"target": map[string]interface{}{
			"getBoundingClientRect": getBoundingClientRect,
		},
	})
	child := parent.fromTouch(event)
	getBoundingClientRect.Release()
	preventDefault.Release()
	want := pixelPosition{
		x: 25,
		y: 172,
	}
	switch {
	case child != want:
		t.Errorf("child values incorrect: wanted %v, got %v", want, child)
	case parent != want:
		t.Errorf("parent should also be modified: wanted %v, got %v", want, parent)
	case !defaultPrevented:
		t.Errorf("wanted preventDefault to be called on event")
	}
}

func TestPixelPositionClear(t *testing.T) {
	pp := pixelPosition{
		x: 1,
		y: 2,
	}
	pp.clear()
	want := pixelPosition{
		x: 0,
		y: 0,
	}
	if pp != want {
		t.Errorf("pixel position not cleared: wanted %v, got %v", want, pp)
	}

}

func TestSort(t *testing.T) {
	tests := []struct {
		a     int
		b     int
		wantA int
		wantB int
	}{
		{},
		{7, 8, 7, 8},
		{15, 13, 13, 15},
		{9, 9, 9, 9},
	}
	for i, test := range tests {
		gotA, gotB := sort(test.a, test.b)
		if test.wantA != gotA || test.wantB != gotB {
			t.Errorf("Test %v: sorts not equal: wanted %v, %v, got %v, %v", i, test.wantA, test.wantB, gotA, gotB)
		}
	}
}

func TestNumCols(t *testing.T) {
	want := 17
	c := Canvas{
		draw: drawMetrics{
			numCols: want,
		},
	}
	got := c.NumCols()
	if want != got {
		t.Errorf("numCols not equal: wanted %v, got %v", want, got)
	}
}

func TestNumCRows(t *testing.T) {
	want := 12
	c := Canvas{
		draw: drawMetrics{
			numRows: want,
		},
	}
	got := c.NumRows()
	if want != got {
		t.Errorf("numRows not equal: wanted %v, got %v", want, got)
	}
}
