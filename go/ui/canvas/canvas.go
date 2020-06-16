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
		ctx       Context
		board     *board.Board
		draw      drawMetrics
		selection selection
		touchPos
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
		startX    int
		startY    int
		endX      int
		endY      int
	}

	// touchPos represents the position of the previous screen touch
	// TODO: rename to position, replace all "Loc" with "Pos", move into selection, replace start&end there, compose in tile.Position, compose tile in tile.Position
	touchPos struct {
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
	offsetFunc := func(event js.Value) (int, int) {
		x := event.Get("offsetX").Int()
		y := event.Get("offsetY").Int()
		return x, y
	}
	mouseDownFunc := func(event js.Value) {
		c.MoveStart(offsetFunc(event))
	}
	mouseUpFunc := func(event js.Value) {
		c.MoveEnd(offsetFunc(event))
	}
	mouseMoveFunc := func(event js.Value) {
		c.MoveCursor(offsetFunc(event))
	}
	touchStartFunc := func(event js.Value) {
		c.touchPos.update(event)
		c.MoveStart(c.touchPos.x, c.touchPos.y)
	}
	touchEndFunc := func(event js.Value) {
		// the event has no touches, use previous touchPos
		c.MoveEnd(c.touchPos.x, c.touchPos.y)
	}
	touchMoveFunc := func(event js.Value) {
		c.touchPos.update(event)
		c.MoveCursor(c.touchPos.x, c.touchPos.y)
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
		x += c.selection.endX - c.selection.startX
		y += c.selection.endY - c.selection.startY
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
	minX, maxX := sort(c.selection.startX, c.selection.endX)
	minY, maxY := sort(c.selection.startY, c.selection.endY)
	width := maxX - minX
	height := maxY - minY
	c.ctx.StrokeRect(minX, minY, width, height)
}

// MoveStart should be called when a move is started to be made at the specified coordinates.
func (c *Canvas) MoveStart(x, y int) {
	if c.gameStatus != game.InProgress {
		return
	}
	c.selection.startX, c.selection.endX = x, x
	c.selection.startY, c.selection.endY = y, y
	ts := c.tileSelection(x, y)
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
func (c *Canvas) MoveCursor(x, y int) {
	if c.gameStatus != game.InProgress {
		return
	}
	switch c.selection.moveState {
	case drag, rect:
		c.selection.endX = x
		c.selection.endY = y
		c.Redraw()
		return
	case grab:
		if c.tileSelection(x, y) == nil {
			c.selection.setMoveState(none)
		}
	case none:
		if c.tileSelection(x, y) != nil {
			c.selection.setMoveState(grab)
		}
	}
}

// MoveEnd should be called when a move is done being made at the specified coordinates.
func (c *Canvas) MoveEnd(x, y int) {
	if c.gameStatus != game.InProgress {
		return
	}
	c.selection.endX = x
	c.selection.endY = y
	switch c.selection.moveState {
	case swap:
		c.selection.setMoveState(none)
		c.swap()
	case rect:
		c.selection.tiles = c.calculateSelectedTiles()
		c.selection.setMoveState(none)
		c.selection.startX, c.selection.endX = 0, 0
		c.selection.startY, c.selection.endY = 0, 0
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
	endTS := c.tileSelection(c.selection.endX, c.selection.endY)
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
func (c Canvas) tileSelection(x, y int) *tileSelection {
	switch {
	case c.draw.unusedMinX <= x && x < c.draw.unusedMinX+len(c.board.UnusedTileIDs)*c.draw.tileLength &&
		c.draw.unusedMinY <= y && y < c.draw.unusedMinY+c.draw.tileLength:
		idx := (x - c.draw.unusedMinX) / c.draw.tileLength
		id := c.board.UnusedTileIDs[idx]
		if t, ok := c.board.UnusedTiles[id]; ok {
			var ts tileSelection
			ts.index = idx
			ts.tile = t
			return &ts
		}
	case c.draw.usedMinX <= x && x < c.draw.usedMinX+c.draw.numCols*c.draw.tileLength &&
		c.draw.usedMinY <= y && y < c.draw.usedMinY+c.draw.numRows*c.draw.tileLength:
		col := tile.X((x - c.draw.usedMinX) / c.draw.tileLength)
		row := tile.Y((y - c.draw.usedMinY) / c.draw.tileLength)
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
	minX, maxX := sort(c.selection.startX, c.selection.endX)
	minY, maxY := sort(c.selection.startY, c.selection.endY)
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
		tiles[id] = tileSelection{
			used: false,
			tile: tile.Tile{
				ID: id,
			},
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
	midTS := c.tileSelection(c.selection.startX, c.selection.startY)
	endC := (c.selection.endX - c.draw.usedMinX) / c.draw.tileLength
	endR := (c.selection.endY - c.draw.usedMinY) / c.draw.tileLength
	switch {
	case midTS == nil:
		log.Error("no tile position at start position")
		return []tile.Position{}
	case midTS.used:
		midTP := tile.Position{
			// TODO: selectionUsedTilePositions should only accept 'Position' because only x, y are needed, not tile
			X: midTS.x,
			Y: midTS.y,
		}
		return c.selectionUsedTilePositions(midTP, endC, endR)
	default:
		return c.selectionUnusedTilePositions(midTS.index, endC, endR)
	}
}

func (c Canvas) selectionUnusedTilePositions(midIndex int, endC, endR int) []tile.Position {
	if endR < 0 || c.draw.numRows <= endR {
		return []tile.Position{}
	}
	tilePositions := make([]tile.Position, 0, len(c.selection.tiles))
	y := tile.Y(endR)
	for _, ts := range c.selection.tiles {
		deltaIdx := ts.index - midIndex
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

func (c Canvas) selectionUsedTilePositions(midTP tile.Position, endC, endR int) []tile.Position {
	tilePositions := make([]tile.Position, 0, len(c.selection.tiles))
	deltaC := endC - int(midTP.X)
	deltaR := endR - int(midTP.Y)
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
	minX, maxX := sort(s.startX, s.endX)
	minY, maxY := sort(s.startY, s.endY)
	return minX <= x && x < maxX &&
		minY <= y && y < maxY
}

func (tp *touchPos) update(event js.Value) {
	event.Call("preventDefault")
	touches := event.Get("touches")
	if touches.Length() == 0 {
		return
	}
	touch := touches.Index(0)
	canvasRect := event.Get("target").Call("getBoundingClientRect")
	tp.x = touch.Get("pageX").Int() - canvasRect.Get("left").Int()
	tp.y = touch.Get("pageY").Int() - canvasRect.Get("top").Int()
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
