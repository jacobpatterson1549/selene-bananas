// +build js,wasm

// Package canvas contains the logic to draw the game
package canvas

import (
	"context"
	"strconv"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

type (
	// Canvas is the object which draws the game
	Canvas struct {
		ctx        Context
		board      *board.Board
		draw       drawMetrics
		selection  selection
		touchPos   pixelPosition
		gameStatus game.Status
	}

	// Config contains the parameters to create a Canvas
	Config struct {
		Width      int
		Height     int
		TileLength int
		FontName   string
	}

	// Context handles the drawing of the canvas
	Context interface {
		SetFont(name string)
		SetLineWidth(width int)
		SetFillColor(name string)
		SetStrokeColor(name string)
		FillText(text string, x, y int)
		ClearRect(x, y, width, height int)
		FillRect(x, y, width, height int)
		StrokeRect(x, y, width, height int)
	}

	// moveState represents the type of move being made by the cursor
	moveState int

	// draw contains the drawing properties for the canvas
	drawMetrics struct {
		width      int
		height     int
		tileLength int
		textOffset int
		unusedMinX int
		unusedMinY int
		usedMinX   int
		usedMinY   int
		numRows    int
		numCols    int
	}

	// selection represents what the cursor has done for the current move
	selection struct {
		moveState moveState
		tiles     map[tile.ID]tileSelection
		isSeen    bool
		start     pixelPosition
		end       pixelPosition
	}

	// pixelPosition represents a location on the canvas
	pixelPosition struct {
		x int
		y int
	}

	// tileSelection represents a tile that the cursor/touch is on
	// If no negative tilePositions were allowed, a negative X could signify isUsed=false and the Y could be the index, but this would silently be bug-ridden if negative positions were ever allowed.
	tileSelection struct {
		used  bool
		index int
		tile  tile.Tile
		x     tile.X
		y     tile.Y
	}
)

const (
	none moveState = iota
	swap
	rect
	drag
	grab
	mainColor       = "black"
	backgroundColor = "white"
	dragColor       = "blue"
)

var moveStateRadioQueries = map[moveState]string{
	none: "#game>input.move-state.none",
	swap: "#game>input.move-state.swap",
	rect: "#game>input.move-state.rect",
	drag: "#game>input.move-state.drag",
	grab: "#game>input.move-state.grab",
}

// New Creates a canvas from the config.
func (cfg Config) New(ctx Context, board *board.Board) Canvas {
	font := strconv.Itoa(cfg.TileLength) + "px " + cfg.FontName
	ctx.SetFont(font)
	ctx.SetLineWidth(2)
	textOffset := (cfg.TileLength * 3) / 20
	padding := 5
	unusedMinX := padding
	unusedMinY := cfg.TileLength
	usedMinX := padding
	usedMinY := cfg.TileLength * 4
	usedMaxX := cfg.Width - padding
	usedMaxY := cfg.Height - padding
	numRows := (usedMaxY - usedMinY) / cfg.TileLength
	numCols := (usedMaxX - usedMinX) / cfg.TileLength
	c := Canvas{
		ctx:   ctx,
		board: board,
		draw: drawMetrics{
			width:      cfg.Width,
			height:     cfg.Height,
			tileLength: cfg.TileLength,
			textOffset: textOffset,
			unusedMinX: unusedMinX,
			unusedMinY: unusedMinY,
			usedMinX:   usedMinX,
			usedMinY:   usedMinY,
			numRows:    numRows,
			numCols:    numCols,
		},
		selection: selection{
			tiles: make(map[tile.ID]tileSelection),
		},
	}
	return c
}

// InitDom regesters canvas dom functions.
func (c *Canvas) InitDom(ctx context.Context, wg *sync.WaitGroup, canvasElement js.Value) {
	wg.Add(1)
	redrawJsFunc := dom.NewJsFunc(c.Redraw)
	swapTileJsFunc := dom.NewJsFunc(c.StartSwap)
	var mousePP pixelPosition
	mouseDownFunc := func(event js.Value) {
		c.MoveStart(mousePP.fromMouse(event))
	}
	mouseUpFunc := func(event js.Value) {
		c.MoveEnd(mousePP.fromMouse(event))
		println("move end")
	}
	mouseMoveFunc := func(event js.Value) {
		c.MoveCursor(mousePP.fromMouse(event))
	}
	var touchPP pixelPosition
	touchStartFunc := func(event js.Value) {
		c.MoveStart(touchPP.fromTouch(event))
	}
	touchEndFunc := func(event js.Value) {
		// the event has no touches, use previous touchPos
		c.MoveEnd(touchPP)
	}
	touchMoveFunc := func(event js.Value) {
		c.MoveCursor(touchPP.fromTouch(event))
	}
	dom.RegisterFunc("canvas", "redraw", redrawJsFunc)
	dom.RegisterFunc("canvas", "swapTile", swapTileJsFunc)
	mouseDownJsFunc := dom.RegisterEventListenerFunc(canvasElement, "mousedown", mouseDownFunc)
	mouseUpJsFunc := dom.RegisterEventListenerFunc(canvasElement, "mouseup", mouseUpFunc)
	mouseMoveJsFunc := dom.RegisterEventListenerFunc(canvasElement, "mousemove", mouseMoveFunc)
	touchStartJsFunc := dom.RegisterEventListenerFunc(canvasElement, "touchstart", touchStartFunc)
	touchEndJsFunc := dom.RegisterEventListenerFunc(canvasElement, "touchend", touchEndFunc)
	touchMoveJsFunc := dom.RegisterEventListenerFunc(canvasElement, "touchmove", touchMoveFunc)
	go func() {
		<-ctx.Done()
		redrawJsFunc.Release()
		swapTileJsFunc.Release()
		mouseDownJsFunc.Release()
		mouseUpJsFunc.Release()
		mouseMoveJsFunc.Release()
		touchStartJsFunc.Release()
		touchEndJsFunc.Release()
		touchMoveJsFunc.Release()
		wg.Done()
	}()
}

// Redraw draws the canvas
func (c *Canvas) Redraw() {
	c.ctx.ClearRect(0, 0, c.draw.width, c.draw.height)
	c.ctx.SetFillColor(backgroundColor)
	c.ctx.FillRect(0, 0, c.draw.width, c.draw.height)
	c.ctx.SetStrokeColor(mainColor)
	c.ctx.SetFillColor(mainColor)
	c.ctx.FillText("Unused Tiles", 0, c.draw.unusedMinY-c.draw.textOffset)
	c.drawUnusedTiles(false)
	c.ctx.FillText("Game Area:", 0, c.draw.usedMinY-c.draw.textOffset)
	c.ctx.StrokeRect(c.draw.usedMinX, c.draw.usedMinY,
		c.draw.numCols*c.draw.tileLength, c.draw.numRows*c.draw.tileLength)
	c.drawUsedTiles(false)
	switch {
	case c.gameStatus == game.NotStarted:
		c.ctx.FillText("Not Started",
			c.draw.usedMinX+2*c.draw.tileLength,
			c.draw.usedMinY+3*c.draw.tileLength-c.draw.textOffset)
	case c.selection.moveState == rect:
		c.drawSelectionRectangle()
	case len(c.selection.tiles) > 0:
		c.ctx.SetStrokeColor(dragColor)
		c.ctx.SetFillColor(dragColor)
		c.drawUnusedTiles(true)
		c.drawUsedTiles(true)
	}
}

// GameStatus sets the gameStatus for the canvas.  The canvas is redrawn afterwards to clean up drawing artifacts
func (c *Canvas) GameStatus(s game.Status) {
	c.gameStatus = s
	c.selection.setMoveState(none)
	c.selection.tiles = map[tile.ID]tileSelection{}
	c.Redraw()
}

func (c *Canvas) drawUnusedTiles(fromSelection bool) {
	for i, id := range c.board.UnusedTileIDs {
		x := c.draw.unusedMinX + i*c.draw.tileLength
		y := c.draw.unusedMinY
		t := c.board.UnusedTiles[id]
		c.drawTile(x, y, t, fromSelection)
	}
}

func (c *Canvas) drawUsedTiles(fromSelection bool) {
	for xCol, yUsedTileLocs := range c.board.UsedTileLocs {
		for yRow, t := range yUsedTileLocs {
			x := c.draw.usedMinX + int(xCol)*c.draw.tileLength
			y := c.draw.usedMinY + int(yRow)*c.draw.tileLength
			c.drawTile(x, y, t, fromSelection)
		}
	}
}

func (c *Canvas) drawTile(x, y int, t tile.Tile, fromSelection bool) {
	switch {
	case fromSelection:
		// TODO: write test to show that tiles that are not selected should not be drawn if the drawing mode is for selected tiles only.
		// only draw selected tiles
		if _, ok := c.selection.tiles[t.ID]; !ok {
			return
		}
		// draw tile with change in location
		x += c.selection.end.x - c.selection.start.x
		y += c.selection.end.y - c.selection.start.y
	case c.selection.moveState == drag:
		// do not draw tiles in selection at their original locations
		if _, ok := c.selection.tiles[t.ID]; ok { // (TODO: similar test as that above)
			return
		}
	}
	c.ctx.StrokeRect(x, y, c.draw.tileLength, c.draw.tileLength)
	c.ctx.FillText(t.Ch.String(), x+c.draw.textOffset, y+c.draw.tileLength-c.draw.textOffset)
}

func (c *Canvas) drawSelectionRectangle() {
	minX, maxX := sort(c.selection.start.x, c.selection.end.x)
	minY, maxY := sort(c.selection.start.y, c.selection.end.y)
	width := maxX - minX
	height := maxY - minY
	c.ctx.StrokeRect(minX, minY, width, height)
}

// MoveStart should be called when a move is started to be made at the specified coordinates.
func (c *Canvas) MoveStart(pp pixelPosition) {
	if c.gameStatus != game.InProgress {
		return
	}
	c.selection.start, c.selection.end = pp, pp
	ts := c.tileSelection(pp)
	if c.selection.moveState == swap {
		switch {
		case ts == nil:
			c.selection.setMoveState(none)
			log.Info("swap cancelled")
		default:
			tileId := ts.tile.ID
			c.selection.tiles[tileId] = *ts
		}
		return
	}
	hasPreviousSelection := len(c.selection.tiles) > 0
	switch {
	case hasPreviousSelection && ts != nil:
		tileId := ts.tile.ID
		if _, ok := c.selection.tiles[tileId]; !ok {
			c.selection.tiles = make(map[tile.ID]tileSelection)
			c.selection.tiles[tileId] = *ts
		}
		c.selection.setMoveState(drag)
	case hasPreviousSelection:
		c.selection.tiles = make(map[tile.ID]tileSelection)
		c.selection.setMoveState(none)
		c.Redraw()
	case ts != nil:
		tileId := ts.tile.ID
		c.selection.tiles[tileId] = *ts
		c.selection.setMoveState(drag)
	default:
		c.selection.setMoveState(rect)
	}
}

// MoveCursor should be called whenever the cursor moves, regardless of if a move is being made.
func (c *Canvas) MoveCursor(pp pixelPosition) {
	if c.gameStatus != game.InProgress {
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

// MoveEnd should be called when a move is done being made at the specified coordinates.
func (c *Canvas) MoveEnd(pp pixelPosition) {
	if c.gameStatus != game.InProgress {
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

// StartSwap start a swap move
func (c *Canvas) StartSwap() {
	log.Info("click a tile to swap for three others from the pile")
	c.selection.setMoveState(swap)
	c.selection.tiles = make(map[tile.ID]tileSelection)
	c.Redraw()
}

// swap trades a tile for some new ones
func (c *Canvas) swap() {
	endTS := c.tileSelection(c.selection.end)
	endTileWasSelected := func() bool {
		_, ok := c.selection.tiles[endTS.tile.ID]
		return ok
	}
	if endTS == nil || !endTileWasSelected() {
		log.Info("swap cancelled")
	}
	if err := c.board.RemoveTile(endTS.tile); err != nil {
		log.Error("removing tile while swapping: " + err.Error())
	}
	dom.Send(game.Message{
		Type: game.Swap,
		Tiles: []tile.Tile{
			endTS.tile,
		},
	})
}

// getTileSelection returns the tile at the specified coordinates on the canvas or nil if none exists
func (c Canvas) tileSelection(pp pixelPosition) *tileSelection {
	switch {
	case c.draw.unusedMinX <= pp.x && pp.x < c.draw.unusedMinX+len(c.board.UnusedTileIDs)*c.draw.tileLength &&
		c.draw.unusedMinY <= pp.y && pp.y < c.draw.unusedMinY+c.draw.tileLength:
		idx := (pp.x - c.draw.unusedMinX) / c.draw.tileLength
		id := c.board.UnusedTileIDs[idx]
		if t, ok := c.board.UnusedTiles[id]; ok {
			var ts tileSelection
			ts.index = idx
			ts.tile = t
			return &ts
		}
	case c.draw.usedMinX <= pp.x && pp.x < c.draw.usedMinX+c.draw.numCols*c.draw.tileLength &&
		c.draw.usedMinY <= pp.y && pp.y < c.draw.usedMinY+c.draw.numRows*c.draw.tileLength:
		col := tile.X((pp.x - c.draw.usedMinX) / c.draw.tileLength)
		row := tile.Y((pp.y - c.draw.usedMinY) / c.draw.tileLength)
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

func (c Canvas) calculateSelectedTiles() map[tile.ID]tileSelection {
	minX, maxX := sort(c.selection.start.x, c.selection.end.x)
	minY, maxY := sort(c.selection.start.y, c.selection.end.y)
	selectedUnusedTileIds := c.calculateSelectedUnusedTiles(minX, maxX, minY, maxY)
	selectedUsedTileIds := c.calculateSelectedUsedTiles(minX, maxX, minY, maxY)
	switch {
	case len(selectedUnusedTileIds) == 0:
		return selectedUsedTileIds
	case len(selectedUsedTileIds) != 0:
		return map[tile.ID]tileSelection{} // cannot select used and unused tiles
	default:
		return selectedUnusedTileIds
	}
}

func (c Canvas) calculateSelectedUnusedTiles(minX, maxX, minY, maxY int) map[tile.ID]tileSelection {
	switch {
	case maxX < c.draw.unusedMinX,
		c.draw.unusedMinX+len(c.board.UnusedTileIDs)*c.draw.tileLength <= minX,
		maxY < c.draw.unusedMinY,
		c.draw.unusedMinY+c.draw.tileLength <= minY:
		return map[tile.ID]tileSelection{}
	}
	minI := (minX - c.draw.unusedMinX) / c.draw.tileLength
	if minI < 0 {
		minI = 0
	}
	maxI := (maxX - c.draw.unusedMinX) / c.draw.tileLength
	if maxI > len(c.board.UnusedTileIDs) {
		maxI = len(c.board.UnusedTileIDs)
	}
	tiles := make(map[tile.ID]tileSelection)
	for i, id := range c.board.UnusedTileIDs[minI:maxI] {
		t := c.board.UnusedTiles[id]
		tiles[id] = tileSelection{
			used:  false,
			tile:  t,
			index: minI + i,
		}
	}
	return tiles
}

func (c Canvas) calculateSelectedUsedTiles(minX, maxX, minY, maxY int) map[tile.ID]tileSelection {
	switch {
	case maxX < c.draw.usedMinX,
		c.draw.usedMinX+c.draw.numCols*c.draw.tileLength <= minX,
		maxY < c.draw.usedMinY,
		c.draw.usedMinY+c.draw.numRows*c.draw.tileLength <= minY:
		return map[tile.ID]tileSelection{}
	}
	minCol := tile.X((minX - c.draw.usedMinX) / c.draw.tileLength)
	maxCol := tile.X((maxX - c.draw.usedMinX) / c.draw.tileLength)
	minRow := tile.Y((minY - c.draw.usedMinY) / c.draw.tileLength)
	maxRow := tile.Y((maxY - c.draw.usedMinY) / c.draw.tileLength)
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

func (c *Canvas) moveSelectedTiles() {
	tilePositions := c.selectionTilePositions()
	if len(tilePositions) == 0 {
		return
	}
	if !c.board.CanMoveTiles(tilePositions) {
		return
	}
	if err := c.board.MoveTiles(tilePositions); err != nil {
		log.Error("moving tiles to presumably valid locations: " + err.Error())
		return
	}
	dom.Send(game.Message{
		Type:          game.TilesMoved,
		TilePositions: tilePositions,
	})
}

// selectionTilePositions calculates the new positions of the selected tiles.
func (c Canvas) selectionTilePositions() []tile.Position {
	if len(c.selection.tiles) == 0 {
		return []tile.Position{}
	}
	startTS := c.tileSelection(c.selection.start)
	endC := (c.selection.end.x - c.draw.usedMinX) / c.draw.tileLength
	endR := (c.selection.end.y - c.draw.usedMinY) / c.draw.tileLength
	switch {
	case startTS == nil:
		log.Error("no tile position at start position")
		return []tile.Position{}
	case startTS.used:
		return c.selectionUsedTilePositions(*startTS, endC, endR)
	default:
		return c.selectionUnusedTilePositions(*startTS, endC, endR)
	}
}

func (c Canvas) selectionUnusedTilePositions(startTS tileSelection, endC, endR int) []tile.Position {
	if endR < 0 || c.draw.numRows <= endR {
		return []tile.Position{}
	}
	tilePositions := make([]tile.Position, 0, len(c.selection.tiles))
	y := tile.Y(endR)
	for _, ts := range c.selection.tiles {
		deltaIdx := ts.index - startTS.index
		col := endC + deltaIdx
		switch {
		case col < 0, c.draw.numCols <= col:
			return []tile.Position{}
		}
		tilePositions = append(tilePositions, tile.Position{
			Tile: ts.tile,
			X:    tile.X(col),
			Y:    y,
		})
	}
	return tilePositions
}

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
			return []tile.Position{}
		}
		tilePositions = append(tilePositions, tile.Position{
			Tile: ts.tile,
			X:    tile.X(col),
			Y:    tile.Y(row),
		})
	}
	return tilePositions
}

func (s *selection) setMoveState(ms moveState) {
	s.moveState = ms
	if query, ok := moveStateRadioQueries[ms]; ok {
		dom.SetCheckedQuery(query, true)
	}
}

func (s selection) inRect(x, y int) bool {
	minX, maxX := sort(s.start.x, s.end.x)
	minY, maxY := sort(s.start.y, s.end.y)
	return minX <= x && x < maxX &&
		minY <= y && y < maxY
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
		log.Error("no touches for touch event, using previous touch location")
		return *pp
	}
	touch := touches.Index(0)
	canvasRect := event.Get("target").Call("getBoundingClientRect")
	pp.x = touch.Get("pageX").Int() - canvasRect.Get("left").Int()
	pp.y = touch.Get("pageY").Int() - canvasRect.Get("top").Int()
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
