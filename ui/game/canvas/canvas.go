// +build js,wasm

// Package canvas contains the logic to draw the game.
package canvas

import (
	"context"
	"strconv"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// Canvas is the object which draws the game.
	Canvas struct {
		log        *log.Log
		ctx        Context
		board      *board.Board
		draw       drawMetrics
		selection  selection
		touchPos   pixelPosition
		gameStatus game.Status
		Socket     Socket
		parentDiv  *js.Value
		element    *js.Value
		mainColor  string
		tileColor  string
		dragColor  string
	}

	// Config contains the parameters to create a Canvas.
	Config struct {
		Log        *log.Log
		TileLength int
	}

	// Context handles the drawing of the canvas.
	Context interface {
		SetFont(name string)
		SetLineWidth(width float64)
		SetFillColor(name string)
		SetStrokeColor(name string)
		FillText(text string, x, y int)
		ClearRect(x, y, width, height int)
		FillRect(x, y, width, height int)
		StrokeRect(x, y, width, height int)
	}

	// moveState represents the type of move being made by the cursor.
	moveState int

	// draw contains the drawing properties for the canvas.
	drawMetrics struct {
		width      int
		height     int
		tileLength int
		textOffset int
		unusedMin  pixelPosition
		usedMin    pixelPosition
		numRows    int
		numCols    int
	}

	// selection represents what the cursor has done for the current move.
	selection struct {
		moveState moveState
		tiles     map[tile.ID]tileSelection
		isSeen    bool
		start     pixelPosition
		end       pixelPosition
	}

	// pixelPosition represents a location on the canvas.
	pixelPosition struct {
		log *log.Log
		x   int
		y   int
	}

	// tileSelection represents a tile that the cursor/touch is on.
	// If no negative tilePositions were allowed, a negative X could signify isUsed=false and the Y could be the index, but this would silently be bug-ridden if negative positions were ever allowed.
	tileSelection struct {
		used  bool
		index int
		tile  tile.Tile
		x     tile.X
		y     tile.Y
	}

	// Socket sends messages to the server.
	Socket interface {
		Send(m game.Message)
	}
)

const (
	none moveState = iota
	swap
	rect
	drag
	grab
)

// New Creates a canvas from the config.
func (cfg Config) New(board *board.Board) *Canvas {
	parentDiv := dom.QuerySelector(".game>.canvas")
	element := dom.QuerySelector(".game>.canvas>canvas")
	contextElement := element.Call("getContext", "2d")
	divColor := func(query string) string {
		div := element.Call("querySelector", query)
		color := dom.Color(div)
		return color
	}
	mainColor := divColor(".mainColor")
	dragColor := divColor(".dragColor")
	tileColor := divColor(".tileColor")
	ctx := jsContext{
		ctx: &contextElement,
	}
	c := Canvas{
		log:   cfg.Log,
		ctx:   &ctx,
		board: board,
		selection: selection{
			tiles: make(map[tile.ID]tileSelection),
		},
		parentDiv: &parentDiv,
		element:   &element,
		draw: drawMetrics{
			tileLength: cfg.TileLength,
		},
		mainColor: mainColor,
		dragColor: dragColor,
		tileColor: tileColor,
	}
	return &c
}

// UpdateSize sets the draw properties of the canvas for it's current size in the window.
func (c *Canvas) UpdateSize() {
	c.draw.width = c.parentDiv.Get("offsetWidth").Int()
	padding := 5
	c.draw.textOffset = (c.draw.tileLength * 3) / 20
	c.draw.unusedMin.x = padding
	c.draw.unusedMin.y = c.draw.tileLength
	c.draw.usedMin.x = padding
	c.draw.usedMin.y = c.draw.tileLength * 4
	c.draw.numCols = (c.draw.width - c.draw.usedMin.x - padding) / c.draw.tileLength
	c.draw.numRows = 500 / c.draw.numCols
	c.draw.height = c.draw.usedMin.y + c.draw.numRows*c.draw.tileLength
	c.element.Set("width", c.draw.width)
	c.element.Set("height", c.draw.height)
	c.parentDiv.Set("width", c.draw.width)
	c.parentDiv.Set("height", c.draw.height)
	c.ctx.SetFont(strconv.Itoa(c.draw.tileLength) + "px sans-serif")
	c.ctx.SetLineWidth(float64(c.draw.tileLength) / 10)
}

// SetTileLength sets the drawing size of the tiles length/height.
func (c *Canvas) SetTileLength(tileLength int) {
	c.draw.tileLength = tileLength
	c.UpdateSize()
}

// InitDom registers canvas dom functions by adding an event listeners to the canvas element.
func (c *Canvas) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	funcs := c.createEventFuncs()
	jsFuncs := make(map[string]js.Func, len(funcs))
	options := map[string]interface{}{
		"passive": false,
	}
	for fnName, fn := range funcs {
		jsFunc := c.createEventJsFunc(fnName, fn)
		c.element.Call("addEventListener", fnName, jsFunc, options)
		jsFuncs[fnName] = jsFunc
	}
	wg.Add(1)
	go dom.ReleaseJsFuncsOnDone(ctx, wg, jsFuncs)
}

// createEventFuncs creates the event listener functions for mouse/touch interaction.
func (c *Canvas) createEventFuncs() map[string]func(event js.Value) {
	mousePP := c.newPixelPosition()
	touchPP := c.newPixelPosition()
	funcs := map[string]func(event js.Value){
		"mousedown": func(event js.Value) {
			c.moveStart(mousePP.fromMouse(event))
		},
		"mouseup": func(event js.Value) {
			c.moveEnd(mousePP.fromMouse(event))
		},
		"mousemove": func(event js.Value) {
			c.moveCursor(mousePP.fromMouse(event))
		},
		"touchstart": func(event js.Value) {
			c.moveStart(touchPP.fromTouch(event))
		},
		"touchend": func(event js.Value) {
			// the event has no touches, use previous touchPos
			c.moveEnd(*touchPP)
		},
		"touchmove": func(event js.Value) {
			c.moveCursor(touchPP.fromTouch(event))
		},
	}
	return funcs
}

// createEventJsFunc creates a jsFunc from the function.
// This is not inlined to prevent the runtime from overwriting all functions with the last one.
func (Canvas) createEventJsFunc(fnName string, fn func(event js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		fn(event)
		return nil
	})
}

// Redraw draws the canvas
func (c *Canvas) Redraw() {
	c.ctx.ClearRect(0, 0, c.draw.width, c.draw.height)
	c.ctx.SetStrokeColor(c.mainColor)
	c.ctx.SetFillColor(c.mainColor)
	c.ctx.FillText("Unused Tiles", 0, c.draw.unusedMin.y-c.draw.textOffset)
	c.ctx.FillText("Game Area:", 0, c.draw.usedMin.y-c.draw.textOffset)
	c.ctx.StrokeRect(c.draw.usedMin.x, c.draw.usedMin.y,
		c.draw.numCols*c.draw.tileLength, c.draw.numRows*c.draw.tileLength)
	c.drawUnusedTiles(false)
	c.drawUsedTiles(false)
	switch {
	case c.gameStatus == game.NotStarted:
		c.ctx.FillText("Not Started",
			c.draw.usedMin.x+2*c.draw.tileLength,
			c.draw.usedMin.y+3*c.draw.tileLength-c.draw.textOffset)
	case c.selection.moveState == rect:
		c.drawSelectionRectangle()
	case len(c.selection.tiles) > 0:
		c.ctx.SetStrokeColor(c.dragColor)
		c.drawUnusedTiles(true)
		c.drawUsedTiles(true)
	}
}

// SetGameStatus sets the gameStatus for the canvas.  The canvas is redrawn afterwards to clean up drawing artifacts
func (c *Canvas) SetGameStatus(s game.Status) {
	c.gameStatus = s
	c.selection.setMoveState(none)
	c.selection.tiles = make(map[tile.ID]tileSelection)
}

// drawUsedTiles pants the unused tiles.
func (c *Canvas) drawUnusedTiles(fromSelection bool) {
	for i, id := range c.board.UnusedTileIDs {
		x := c.draw.unusedMin.x + i*c.draw.tileLength
		y := c.draw.unusedMin.y
		t := c.board.UnusedTiles[id]
		c.drawTile(x, y, t, fromSelection)
	}
}

// drawUsedTiles pants the used tiles.
func (c *Canvas) drawUsedTiles(fromSelection bool) {
	for xCol, yUsedTileLocs := range c.board.UsedTileLocs {
		for yRow, t := range yUsedTileLocs {
			x := c.draw.usedMin.x + int(xCol)*c.draw.tileLength
			y := c.draw.usedMin.y + int(yRow)*c.draw.tileLength
			c.drawTile(x, y, t, fromSelection)
		}
	}
}

// drawTile paints the tile at the specified top-left coordinate.
// The fromSelection flag specifies whether tiles from the selection or those not from the selection are being drawn.
func (c *Canvas) drawTile(x, y int, t tile.Tile, fromSelection bool) {
	lineColor := c.mainColor
	switch {
	case fromSelection:
		// only draw selected tiles
		if _, ok := c.selection.tiles[t.ID]; !ok {
			return
		}
		lineColor = c.dragColor
		// draw tile with change in location
		x += c.selection.end.x - c.selection.start.x
		y += c.selection.end.y - c.selection.start.y
	case c.selection.moveState == drag:
		// do not draw tiles in selection at their original locations
		if _, ok := c.selection.tiles[t.ID]; ok {
			return
		}
	}
	c.ctx.SetFillColor(c.tileColor)
	c.ctx.FillRect(x, y, c.draw.tileLength, c.draw.tileLength)
	c.ctx.SetFillColor(lineColor)
	c.ctx.StrokeRect(x, y, c.draw.tileLength, c.draw.tileLength)
	c.ctx.SetFillColor(lineColor)
	c.ctx.FillText(string(t.Ch), x+c.draw.textOffset, y+c.draw.tileLength-c.draw.textOffset)
}

// drawSelectionRectangle draws the outline of the selection.
func (c *Canvas) drawSelectionRectangle() {
	minX, maxX := sort(c.selection.start.x, c.selection.end.x)
	minY, maxY := sort(c.selection.start.y, c.selection.end.y)
	width := maxX - minX
	height := maxY - minY
	c.ctx.StrokeRect(minX, minY, width, height)
}

// moveStart should be called when a move is started to be made at the specified coordinates.
func (c *Canvas) moveStart(pp pixelPosition) {
	if !c.canMove() {
		return
	}
	c.selection.start, c.selection.end = pp, pp
	ts := c.tileSelection(pp)
	if c.selection.moveState == swap {
		switch {
		case ts == nil:
			c.selection.setMoveState(none)
			c.log.Info("swap cancelled")
		default:
			tileID := ts.tile.ID
			c.selection.tiles[tileID] = *ts
		}
		return
	}
	hasPreviousSelection := len(c.selection.tiles) > 0
	switch {
	case hasPreviousSelection && ts != nil:
		tileID := ts.tile.ID
		if _, ok := c.selection.tiles[tileID]; !ok {
			c.selection.tiles = make(map[tile.ID]tileSelection)
			c.selection.tiles[tileID] = *ts
		}
		c.selection.setMoveState(drag)
	case hasPreviousSelection: // && ts == nil
		c.selection.tiles = make(map[tile.ID]tileSelection)
		c.selection.setMoveState(rect)
		c.Redraw()
	case ts != nil: // && !hasPreviousSelection
		tileID := ts.tile.ID
		c.selection.tiles[tileID] = *ts
		c.selection.setMoveState(drag)
	default: // !hasPerviousSelection && ts == nil
		c.selection.setMoveState(rect)
	}
}

// moveCursor should be called whenever the cursor moves, regardless of if a move is being made.
func (c *Canvas) moveCursor(pp pixelPosition) {
	if !c.canMove() {
		return
	}
	switch c.selection.moveState {
	case drag, rect:
		c.selection.end = pp
		c.Redraw()
		return
	case grab:
		if c.tileSelection(pp) == nil {
			c.selection.setMoveState(none)
		}
	case none:
		if c.tileSelection(pp) != nil {
			c.selection.setMoveState(grab)
		}
	}
}

// moveEnd should be called when a move is done being made at the specified coordinates.
func (c *Canvas) moveEnd(pp pixelPosition) {
	if !c.canMove() {
		return
	}
	c.selection.end = pp
	switch c.selection.moveState {
	case swap:
		c.selection.setMoveState(none)
		c.swap()
	case rect:
		c.selection.tiles = c.calculateSelectedTiles()
		c.selection.setMoveState(none)
		c.selection.start = pixelPosition{}
		c.selection.end = pixelPosition{}
		c.Redraw()
	case drag:
		c.moveSelectedTiles()
		c.selection.tiles = make(map[tile.ID]tileSelection)
		c.selection.setMoveState(none)
		c.Redraw()
	}
}

func (c Canvas) canMove() bool {
	switch c.gameStatus {
	case game.InProgress, game.FinishedAllowMove:
		return true
	}
	return false
}

// StartSwap start a swap move.
func (c *Canvas) StartSwap() {
	c.log.Info("click a tile to swap for three others from the pile")
	c.selection.setMoveState(swap)
	c.selection.tiles = make(map[tile.ID]tileSelection)
	c.Redraw()
}

// swap trades a tile for some new ones.
func (c *Canvas) swap() {
	endTS := c.tileSelection(c.selection.end)
	endTileWasSelected := func() bool {
		_, ok := c.selection.tiles[endTS.tile.ID]
		return ok
	}
	if endTS == nil || !endTileWasSelected() {
		c.log.Info("swap cancelled")
	}
	if err := c.board.RemoveTile(endTS.tile); err != nil {
		c.log.Error("removing tile while swapping: " + err.Error())
	}
	c.Socket.Send(game.Message{
		Type: game.Swap,
		Tiles: []tile.Tile{
			endTS.tile,
		},
	})
}

// getTileSelection returns the tile at the specified coordinates on the canvas or nil if none exists.
func (c Canvas) tileSelection(pp pixelPosition) *tileSelection {
	switch {
	case c.draw.unusedMin.x <= pp.x && pp.x < c.draw.unusedMin.x+len(c.board.UnusedTileIDs)*c.draw.tileLength &&
		c.draw.unusedMin.y <= pp.y && pp.y < c.draw.unusedMin.y+c.draw.tileLength:
		idx := (pp.x - c.draw.unusedMin.x) / c.draw.tileLength
		id := c.board.UnusedTileIDs[idx]
		if t, ok := c.board.UnusedTiles[id]; ok {
			var ts tileSelection
			ts.index = idx
			ts.tile = t
			return &ts
		}
	case c.draw.usedMin.x <= pp.x && pp.x < c.draw.usedMin.x+c.draw.numCols*c.draw.tileLength &&
		c.draw.usedMin.y <= pp.y && pp.y < c.draw.usedMin.y+c.draw.numRows*c.draw.tileLength:
		col := tile.X((pp.x - c.draw.usedMin.x) / c.draw.tileLength)
		row := tile.Y((pp.y - c.draw.usedMin.y) / c.draw.tileLength)
		if yUsedTileLocs, ok := c.board.UsedTileLocs[col]; ok {
			if t, ok := yUsedTileLocs[row]; ok {
				var ts tileSelection
				ts.used = true
				ts.tile = t
				ts.x = col
				ts.y = row
				return &ts
			}
		}
	}
	return nil
}

// calculateSelectedTiles determines which tiles are selected from the selection.
func (c Canvas) calculateSelectedTiles() map[tile.ID]tileSelection {
	minX, maxX := sort(c.selection.start.x, c.selection.end.x)
	minY, maxY := sort(c.selection.start.y, c.selection.end.y)
	selectedUnusedTileIds := c.calculateSelectedUnusedTiles(minX, maxX, minY, maxY)
	selectedUsedTileIds := c.calculateSelectedUsedTiles(minX, maxX, minY, maxY)
	switch {
	case len(selectedUnusedTileIds) == 0:
		return selectedUsedTileIds
	case len(selectedUsedTileIds) != 0:
		return nil // cannot select used and unused tiles
	default:
		return selectedUnusedTileIds
	}
}

// calculateSelectedUnusedTiles determines which unused tiles are selected from the selection.
func (c Canvas) calculateSelectedUnusedTiles(minX, maxX, minY, maxY int) map[tile.ID]tileSelection {
	switch {
	case maxX < c.draw.unusedMin.x,
		c.draw.unusedMin.x+len(c.board.UnusedTileIDs)*c.draw.tileLength <= minX,
		maxY < c.draw.unusedMin.y,
		c.draw.unusedMin.y+c.draw.tileLength <= minY:
		return make(map[tile.ID]tileSelection)
	}
	minI := (minX - c.draw.unusedMin.x) / c.draw.tileLength
	if minI < 0 {
		minI = 0
	}
	maxI := (maxX - c.draw.unusedMin.x) / c.draw.tileLength
	if maxI >= len(c.board.UnusedTileIDs) {
		maxI = len(c.board.UnusedTileIDs) - 1
	}
	tiles := make(map[tile.ID]tileSelection)
	for i, id := range c.board.UnusedTileIDs[minI : maxI+1] {
		t := c.board.UnusedTiles[id]
		tiles[id] = tileSelection{
			used:  false,
			tile:  t,
			index: minI + i,
		}
	}
	return tiles
}

// calculateSelectedUsedTiles determines which used tiles are selected from the selection.
func (c Canvas) calculateSelectedUsedTiles(minX, maxX, minY, maxY int) map[tile.ID]tileSelection {
	switch {
	case maxX < c.draw.usedMin.x,
		c.draw.usedMin.x+c.draw.numCols*c.draw.tileLength <= minX,
		maxY < c.draw.usedMin.y,
		c.draw.usedMin.y+c.draw.numRows*c.draw.tileLength <= minY:
		return make(map[tile.ID]tileSelection)
	}
	minCol := tile.X((minX - c.draw.usedMin.x) / c.draw.tileLength)
	maxCol := tile.X((maxX - c.draw.usedMin.x) / c.draw.tileLength)
	minRow := tile.Y((minY - c.draw.usedMin.y) / c.draw.tileLength)
	maxRow := tile.Y((maxY - c.draw.usedMin.y) / c.draw.tileLength)
	tileIds := make(map[tile.ID]tileSelection)
	for col, yUsedTileLocs := range c.board.UsedTileLocs {
		if minCol <= col && col <= maxCol {
			for row, t := range yUsedTileLocs {
				if minRow <= row && row <= maxRow {
					tileIds[t.ID] = tileSelection{
						used: true,
						tile: t,
						x:    col,
						y:    row,
					}
				}
			}
		}
	}
	return tileIds
}

// moveSelectedTiles determines what tiles to move from the selection, changes the local board, and sends the update to the server.
func (c *Canvas) moveSelectedTiles() {
	tilePositions := c.selectionTilePositions()
	if len(tilePositions) == 0 {
		return
	}
	if !c.board.CanMoveTiles(tilePositions) {
		return
	}
	if err := c.board.MoveTiles(tilePositions); err != nil {
		c.log.Error("moving tiles to presumably valid locations: " + err.Error())
		return
	}
	if c.gameStatus == game.InProgress {
		c.Socket.Send(game.Message{
			Type:          game.TilesMoved,
			TilePositions: tilePositions,
		})
	}
}

// selectionTilePositions calculates the new positions of the selected tiles.
func (c Canvas) selectionTilePositions() []tile.Position {
	startTS := c.tileSelection(c.selection.start)
	endC := (c.selection.end.x - c.draw.usedMin.x) / c.draw.tileLength
	endR := (c.selection.end.y - c.draw.usedMin.y) / c.draw.tileLength
	switch {
	case startTS == nil:
		c.log.Error("no tile position at start position")
	case int(startTS.x) == endC && int(startTS.y) == endR:
		// NOOP
	case startTS.used:
		return c.selectionUsedTilePositions(*startTS, endC, endR)
	default:
		return c.selectionUnusedTilePositions(*startTS, endC, endR)
	}
	return nil
}

// selectionUsedTilePositions computes the unused tile positions of the selection.
func (c Canvas) selectionUnusedTilePositions(startTS tileSelection, endC, endR int) []tile.Position {
	if endR < 0 || c.draw.numRows <= endR {
		return nil
	}
	tilePositions := make([]tile.Position, 0, len(c.selection.tiles))
	y := tile.Y(endR)
	for _, ts := range c.selection.tiles {
		deltaIdx := ts.index - startTS.index
		col := endC + deltaIdx
		switch {
		case col < 0, c.draw.numCols <= col:
			return nil
		}
		tilePositions = append(tilePositions, tile.Position{
			Tile: ts.tile,
			X:    tile.X(col),
			Y:    y,
		})
	}
	return tilePositions
}

// selectionUsedTilePositions computes the used tile positions of the selection.
func (c Canvas) selectionUsedTilePositions(startTS tileSelection, endC, endR int) []tile.Position {
	tilePositions := make([]tile.Position, 0, len(c.selection.tiles))
	deltaC := endC - int(startTS.x)
	deltaR := endR - int(startTS.y)
	for _, ts := range c.selection.tiles {
		col := int(ts.x) + deltaC
		row := int(ts.y) + deltaR
		switch {
		case col < 0, c.draw.numCols <= col,
			row < 0, c.draw.numRows <= row:
			return nil
		}
		tilePositions = append(tilePositions, tile.Position{
			Tile: ts.tile,
			X:    tile.X(col),
			Y:    tile.Y(row),
		})
	}
	return tilePositions
}

// setMoveState updates the moveState and checks the appropriate dom element for it.
func (s *selection) setMoveState(ms moveState) {
	s.moveState = ms
	switch ms {
	case none:
		dom.SetChecked(".game>.canvas>.move-state.none", true)
	case swap:
		dom.SetChecked(".game>.canvas>.move-state.swap", true)
	case rect:
		dom.SetChecked(".game>.canvas>.move-state.rect", true)
	case drag:
		dom.SetChecked(".game>.canvas>.move-state.drag", true)
	case grab:
		dom.SetChecked(".game>.canvas>.move-state.grab", true)
	}
}

// inRect determines if the specified coordinates are in the rectangle.
// The left and top edges (min-valued) are inclusive and the right and bottom (max-valued) edges are exclusive.
func (s selection) inRect(x, y int) bool {
	minX, maxX := sort(s.start.x, s.end.x)
	minY, maxY := sort(s.start.y, s.end.y)
	return minX <= x && x < maxX &&
		minY <= y && y < maxY
}

// newPixelPosition creates a new PixelPosition with the log of the canvas.
func (c *Canvas) newPixelPosition() *pixelPosition {
	pp := pixelPosition{
		log: c.log,
	}
	return &pp
}

// fromMouse updates the pixelPosition for the mouse event and returns it
func (pp *pixelPosition) fromMouse(event js.Value) pixelPosition {
	pp.x = event.Get("offsetX").Int()
	pp.y = event.Get("offsetY").Int()
	return *pp
}

// fromTouch updates the pixelPosition for the first touch of the touch event and returns it
func (pp *pixelPosition) fromTouch(event js.Value) pixelPosition {
	event.Call("preventDefault")
	touches := event.Get("touches")
	if touches.Length() == 0 {
		pp.log.Error("no touches for touch event, using previous touch location")
		return *pp
	}
	touch := touches.Index(0)
	canvasRect := event.Get("target").Call("getBoundingClientRect")
	pp.x = touch.Get("clientX").Int() - canvasRect.Get("left").Int()
	pp.y = touch.Get("clientY").Int() - canvasRect.Get("top").Int()
	return *pp
}

// sort returns the elements in increasing order.
// If a < b, the returned values are [a, b].
// If a > b, the returned values are [b, a].
func sort(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}

// NumCols the number of columns the canvas can draw.
func (c Canvas) NumCols() int {
	return c.draw.numCols
}

// NumRows the number of rows the canvas can draw.
func (c Canvas) NumRows() int {
	return c.draw.numRows
}
