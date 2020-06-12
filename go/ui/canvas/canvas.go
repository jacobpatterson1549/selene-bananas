// +build js

// Package canvas contains the logic to draw the game
package canvas

import (
	"strconv"

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
		GameStatus game.Status // TODO: delete gameInProgress checkbox input
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
		tileIds   map[tile.ID]struct{}
		isSeen    bool
		startX    int
		startY    int
		endX      int
		endY      int
	}

	tileSelection struct {
		tile   tile.Tile
		isUsed bool
		r      int
		c      int
	}
)

const (
	none moveState = iota
	swap
	rect
	drag
	grab
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
			tileIds: make(map[tile.ID]struct{}),
		},
	}
	return c
}

// Redraw draws the canvas
func (c *Canvas) Redraw() {
	c.ctx.ClearRect(0, 0, c.draw.width, c.draw.height)
	c.ctx.SetFillColor("white") // TODO constant color names, maybe a color type
	c.ctx.FillRect(0, 0, c.draw.width, c.draw.height)
	c.ctx.SetStrokeColor("black")
	c.ctx.SetFillColor("black")
	c.ctx.FillText("Unused Tiles", 0, c.draw.unusedMinY-c.draw.textOffset)
	c.drawUnusedTiles(false)
	c.ctx.FillText("Game Area:", 0, c.draw.usedMinY-c.draw.textOffset)
	c.ctx.StrokeRect(c.draw.usedMinX, c.draw.usedMinY,
		c.draw.numCols*c.draw.tileLength, c.draw.numRows*c.draw.tileLength)
	c.drawUsedTiles(false)
	switch {
	case c.selection.moveState == rect:
		c.drawSelectionRectangle()
	case len(c.selection.tileIds) > 0:
		c.ctx.SetStrokeColor("blue")
		c.ctx.SetFillColor("blue")
		c.drawUnusedTiles(true)
		c.drawUsedTiles(true)
	}
	if c.GameStatus == game.NotStarted {
		c.ctx.FillText("Not Started",
			c.draw.usedMinX+2*c.draw.tileLength,
			c.draw.usedMinY+3*c.draw.tileLength-c.draw.textOffset)
	}
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
		if _, ok := c.selection.tileIds[t.ID]; !ok {
			return
		}
		// draw tile with change in location
		x += c.selection.endX - c.selection.startX
		y += c.selection.endY - c.selection.startY
	case c.selection.moveState == drag:
		// do not draw tiles in selection at their original locations
		if _, ok := c.selection.tileIds[t.ID]; ok { // (TODO: similar test as that above)
			return
		}
	}
	c.ctx.StrokeRect(x, y, c.draw.tileLength, c.draw.tileLength)
	c.ctx.FillText(t.Ch.String(), x+c.draw.textOffset, y+c.draw.tileLength-c.draw.textOffset)
}

func (c *Canvas) drawSelectionRectangle() {
	var x, y int
	switch {
	case c.selection.startX < c.selection.endX:
		x = c.selection.startX
	default:
		x = c.selection.endX
	}
	switch {
	case c.selection.startY < c.selection.endY:
		y = c.selection.startY
	default:
		y = c.selection.endY
	}
	width := c.selection.endX - c.selection.startX
	height := c.selection.endY - c.selection.startY
	if width < 0 {
		width *= -1
	}
	if height < 0 {
		height *= -1
	}
	c.ctx.StrokeRect(x, y, width, height)
}

// MoveStart should be called when a move is started to be made at the specified coordinates.
func (c *Canvas) MoveStart(x, y int) {
	if c.GameStatus != game.InProgress {
		return
	}
	c.selection.startX, c.selection.endX = x, x
	c.selection.startY, c.selection.endY = y, y
	tileSelection := c.tileSelection(x, y)
	if c.selection.moveState == swap {
		switch {
		case tileSelection == nil:
			c.selection.setMoveState(none)
			log.Info("swap cancelled")
		default:
			tileId := tileSelection.tile.ID
			c.selection.tileIds[tileId] = struct{}{}
		}
		return
	}
	hasPreviousSelection := len(c.selection.tileIds) > 0
	switch {
	case hasPreviousSelection && tileSelection != nil:
		tileId := tileSelection.tile.ID
		if _, ok := c.selection.tileIds[tileId]; !ok {
			c.selection.tileIds = make(map[tile.ID]struct{})
			c.selection.tileIds[tileId] = struct{}{} // [same as case 3]
		}
		c.selection.setMoveState(drag)
	case hasPreviousSelection:
		c.selection.tileIds = make(map[tile.ID]struct{})
		c.selection.setMoveState(none)
		c.Redraw()
	case tileSelection != nil:
		tileId := tileSelection.tile.ID
		c.selection.tileIds[tileId] = struct{}{}
		c.selection.setMoveState(drag)
	default:
		c.selection.setMoveState(rect)
	}
}

// MoveCursor should be called whenever the cursor moves, regardless of if a move is being made.
func (c *Canvas) MoveCursor(x, y int) {
	if c.GameStatus != game.InProgress {
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
	if c.GameStatus != game.InProgress {
		return
	}
	c.selection.endX = x
	c.selection.endY = y
	switch c.selection.moveState {
	case swap:
		c.selection.setMoveState(none)
		c.swap()
	case rect:
		c.selection.tileIds = c.calculateSelectedTileIds()
		c.selection.setMoveState(none)
		c.selection.startX, c.selection.endX = 0, 0
		c.selection.startY, c.selection.endY = 0, 0
		c.Redraw()
	case drag:
		c.moveSelectedTiles()
		c.selection.tileIds = make(map[tile.ID]struct{})
		c.selection.setMoveState(none)
		c.Redraw()
	}
}

// StartSwap start a swap move
func (c *Canvas) StartSwap() {
	log.Info("click a tile to swap for three others from the pile")
	c.selection.setMoveState(swap)
	c.selection.tileIds = make(map[tile.ID]struct{})
	c.Redraw()
}

// swap trades a tile for some new ones
func (c *Canvas) swap() {
	tileSelection := c.tileSelection(c.selection.endX, c.selection.endY)
	tileWasSelected := func() bool {
		_, ok := c.selection.tileIds[tileSelection.tile.ID]
		return ok
	}
	if tileSelection == nil || !tileWasSelected() {
		log.Info("swap cancelled")
	}
	if err := c.board.RemoveTile(tileSelection.tile); err != nil {
		log.Error("removing tile while swapping: " + err.Error())
	}
	dom.Send(game.Message{
		Type: game.Swap,
		Tiles: []tile.Tile{
			tileSelection.tile,
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
			return &tileSelection{
				tile: t,
			}
		}
	case c.draw.usedMinX <= x && x < c.draw.usedMinX+c.draw.numCols*c.draw.tileLength &&
		c.draw.usedMinY <= y && y < c.draw.usedMinY+c.draw.numRows*c.draw.tileLength:
		col := (x - c.draw.usedMinX) / c.draw.tileLength
		row := (y - c.draw.usedMinY) / c.draw.tileLength
		if yUsedTileLocs, ok := c.board.UsedTileLocs[tile.X(col)]; ok {
			if t, ok := yUsedTileLocs[tile.Y(row)]; ok {
				return &tileSelection{
					tile:   t,
					isUsed: true,
					c:      col,
					r:      row,
				}
			}
		}
	}
	return nil
}

func (c Canvas) calculateSelectedTileIds() map[tile.ID]struct{} {
	var minX, maxX, minY, maxY int // TODO: better assignment
	switch {
	case c.selection.startX < c.selection.endX:
		minX = c.selection.startX
		maxX = c.selection.endX
	default:
		minX = c.selection.endX
		maxX = c.selection.startX
	}
	switch {
	case c.selection.startY < c.selection.endY:
		minY = c.selection.startY
		maxY = c.selection.endY
	default:
		minY = c.selection.endY
		maxY = c.selection.startY
	}
	selectedUnusedTileIds := c.calculateSelectedUnusedTileIds(minX, maxX, minY, maxY)
	selectedUsedTileIds := c.calculateSelectedUsedTileIds(minX, maxX, minY, maxY)
	switch {
	case len(selectedUnusedTileIds) == 0:
		return selectedUsedTileIds
	case len(selectedUsedTileIds) != 0:
		return map[tile.ID]struct{}{} // cannot select used and unused tiles
	default:
		return selectedUnusedTileIds
	}
}

func (c Canvas) calculateSelectedUnusedTileIds(minX, maxX, minY, maxY int) map[tile.ID]struct{} {
	switch {
	case maxX < c.draw.unusedMinX,
		c.draw.unusedMinX+len(c.board.UnusedTileIDs)*c.draw.tileLength <= minX,
		maxY < c.draw.unusedMinY,
		c.draw.unusedMinY+c.draw.tileLength <= minY:
		return map[tile.ID]struct{}{}
	}
	minI := (minX - c.draw.unusedMinX) / c.draw.tileLength
	if minI < 0 {
		minI = 0
	}
	maxI := (maxX - c.draw.unusedMinX) / c.draw.tileLength
	if maxI > len(c.board.UnusedTileIDs) {
		maxI = len(c.board.UnusedTileIDs)
	}
	tileIds := make(map[tile.ID]struct{})
	for _, tileId := range c.board.UnusedTileIDs[minI:maxI] {
		tileIds[tileId] = struct{}{}
	}
	return tileIds
}

func (c Canvas) calculateSelectedUsedTileIds(minX, maxX, minY, maxY int) map[tile.ID]struct{} {
	switch {
	case maxX < c.draw.usedMinX,
		c.draw.usedMinX+c.draw.numCols*c.draw.tileLength <= minX,
		maxY < c.draw.usedMinY,
		c.draw.usedMinY+c.draw.numRows*c.draw.tileLength <= minY:
		return map[tile.ID]struct{}{}
	}
	minCol := tile.X((minX - c.draw.usedMinX) / c.draw.tileLength)
	maxCol := tile.X((maxX - c.draw.usedMinX) / c.draw.tileLength)
	minRow := tile.Y((minY - c.draw.usedMinY) / c.draw.tileLength)
	maxRow := tile.Y((maxY - c.draw.usedMinY) / c.draw.tileLength)
	tileIds := make(map[tile.ID]struct{})
	for col, yUsedTileLocs := range c.board.UsedTileLocs {
		if minCol <= col && col <= maxCol {
			for row, t := range yUsedTileLocs {
				if minRow <= row && row <= maxRow {
					tileIds[t.ID] = struct{}{}
				}
			}
		}
	}
	return tileIds
}

func (c *Canvas) moveSelectedTiles() {
	tilePositions := c.selectionTilePositions()
	if !c.board.CanMoveTiles(tilePositions) {
		return
	}
	if err := c.board.MoveTiles(tilePositions); err != nil {
		log.Error("moving tiles to presumably valid locations: " + err.Error())
		return
	}
	if len(tilePositions) > 0 {
		dom.Send(game.Message{
			Type:          game.TilesMoved,
			TilePositions: tilePositions,
		})
	}
}

func (c Canvas) selectionTilePositions() []tile.Position {
	if len(c.selection.tileIds) == 0 {
		return []tile.Position{}
	}
	midTS := c.tileSelection(c.selection.startX, c.selection.startY)
	endC := (c.selection.endX - c.draw.usedMinX) / c.draw.tileLength
	endR := (c.selection.endY - c.draw.usedMinY) / c.draw.tileLength
	switch {
	case midTS.isUsed:
		return c.selectionUsedTilePositions(midTS.tile, endC, endR)
	default:
		return c.selectionUnusedTilePositions(midTS.tile, endC, endR)
	}
}

func (c Canvas) selectionUnusedTilePositions(midT tile.Tile, endC, endR int) []tile.Position {
	if endR < 0 || c.draw.numRows <= endR {
		return []tile.Position{}
	}
	unusedTileIndex := func(id tile.ID) int { // TODO: make non-anonymous function
		for i, id2 := range c.board.UnusedTileIDs {
			if id == id2 {
				return i
			}
		}
		return -1
	}
	midTIdx := unusedTileIndex(midT.ID)
	tilePositions := make([]tile.Position, 0, len(c.selection.tileIds))
	for id, _ := range c.selection.tileIds {
		tileIdx := unusedTileIndex(id)
		deltaIdx := tileIdx - midTIdx
		col := endC + deltaIdx
		if col < 0 || c.draw.numCols <= col {
			return []tile.Position{}
		}
		t := c.board.UnusedTiles[id]
		tilePositions = append(tilePositions, tile.Position{
			Tile: t,
			X:    tile.X(col),
			Y:    tile.Y(endR),
		})
	}
	return tilePositions
}

func (c Canvas) selectionUsedTilePositions(midT tile.Tile, endC, endR int) []tile.Position {
	tilePositions := make([]tile.Position, 0, len(c.selection.tileIds))
	midTP := c.board.UsedTiles[midT.ID]
	deltaC := endC - int(midTP.X)
	deltaR := endR - int(midTP.Y)
	for id, _ := range c.selection.tileIds {
		tp := c.board.UsedTiles[id]
		col := int(tp.X) + deltaC
		row := int(tp.Y) + deltaR
		switch {
		case col < 0, c.draw.numCols <= col,
			row < 0, c.draw.numRows <= row:
			return []tile.Position{}
		}
		tilePositions = append(tilePositions, tile.Position{
			Tile: tp.Tile,
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
	var minX, maxX, minY, maxY int
	switch {
	case s.startX < s.endX:
		minX = s.startX
		maxX = s.endX
	default:
		minX = s.endX
		maxX = s.startX
	}
	switch {
	case s.startY < s.endY:
		minY = s.startY
		maxY = s.endY
	default:
		minY = s.endY
		maxY = s.startY
	}
	return minX <= x && x < maxX &&
		minY <= y && y < maxY
}
