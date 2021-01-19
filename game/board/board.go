// Package board stores the tiles for a game and handles queries to read and update tiles
package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

type (
	// Board represents the positioning of a player's tiles from a game.
	// (Each player has their own board)
	Board struct {
		UnusedTiles   map[tile.ID]tile.Tile
		UnusedTileIDs []tile.ID
		UsedTiles     map[tile.ID]tile.Position
		UsedTileLocs  map[tile.X]map[tile.Y]tile.Tile
		Config
	}

	// Config stores fields for creating a board.
	Config struct {
		NumRows int `json:"r"`
		NumCols int `json:"c"`
	}

	// ResizeResult contains the result of a successful board resize operation.
	ResizeResult struct {
		Info          string
		Tiles         []tile.Tile
		TilePositions []tile.Position
	}

	// jsonBoard is used for serialization with the json/encoding package
	jsonBoard struct {
		UnusedTiles []tile.Tile
		UsedTiles   []tile.Position
		Config
	}
)

const (
	minCols = 10
	minRows = 10
)

// New creates a new board with the unused tiles.
func (cfg Config) New(unusedTiles []tile.Tile) (*Board, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.New("creating board: validation: " + err.Error())
	}
	unusedTilesByID := make(map[tile.ID]tile.Tile, len(unusedTiles))
	unusedTileIDs := make([]tile.ID, len(unusedTiles))
	for i, t := range unusedTiles {
		unusedTilesByID[t.ID] = t
		unusedTileIDs[i] = t.ID
	}
	usedTiles := make(map[tile.ID]tile.Position)
	usedTileLocs := make(map[tile.X]map[tile.Y]tile.Tile)
	b := Board{
		UnusedTiles:   unusedTilesByID,
		UnusedTileIDs: unusedTileIDs,
		UsedTiles:     usedTiles,
		UsedTileLocs:  usedTileLocs,
		Config:        cfg,
	}
	return &b, nil
}

// Validate returns an error if the number of rows or columns is invalid.
func (cfg Config) Validate() error {
	switch {
	case cfg.NumRows < minRows:
		return errors.New("not enough rows on board, must be >= " + strconv.Itoa(minRows))
	case cfg.NumCols < minCols:
		return errors.New("not enough columns on board, must be >= " + strconv.Itoa(minCols))
	}
	return nil
}

// MarshalJSON implements the encoding/json.Marshaler interface.
// Returns an object containing the array of unused tiles, map of tile positions, and the board config.
func (b Board) MarshalJSON() ([]byte, error) {
	unusedTiles := b.sortedUnusedTiles()
	usedTiles := b.sortUsedTiles()
	jb := jsonBoard{
		UnusedTiles: unusedTiles,
		UsedTiles:   usedTiles,
		Config:      b.Config,
	}
	return json.Marshal(jb)
}

// UnmarshalJSON implements the encoding/json.Unmarshaler interface.
// UsedTiles and UnusedTiles are read from arrays and converted into maps for quick lookup.
func (b *Board) UnmarshalJSON(d []byte) error {
	var jb jsonBoard
	if err := json.Unmarshal(d, &jb); err != nil {
		return err
	}
	b.UnusedTiles = make(map[tile.ID]tile.Tile, len(jb.UnusedTiles))
	b.UnusedTileIDs = make([]tile.ID, 0, len(jb.UnusedTiles))
	for _, t := range jb.UnusedTiles {
		b.UnusedTiles[t.ID] = t
		b.UnusedTileIDs = append(b.UnusedTileIDs, t.ID)
	}
	b.UsedTiles = make(map[tile.ID]tile.Position, len(jb.UsedTiles))
	b.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
	for _, tp := range jb.UsedTiles {
		b.UsedTiles[tp.Tile.ID] = tp
		if _, ok := b.UsedTileLocs[tp.X]; !ok {
			b.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
		}
		b.UsedTileLocs[tp.X][tp.Y] = tp.Tile
	}
	b.Config = jb.Config
	return nil
}

// sortedUnusedTiles returns a new array of the unused tiles, sorted by the UnusedTileIDs array.
func (b Board) sortedUnusedTiles() []tile.Tile {
	unusedTiles := make([]tile.Tile, 0, len(b.UnusedTiles))
	for _, id := range b.UnusedTileIDs {
		t := b.UnusedTiles[id]
		unusedTiles = append(unusedTiles, t)
	}
	return unusedTiles
}

// sortUsedTiles returns a new array of the used tiles, sorted by x position, then y position.
func (b Board) sortUsedTiles() []tile.Position {
	usedTiles := make([]tile.Position, 0, len(b.UsedTiles))
	xPositions := make([]int, 0)
	for x := range b.UsedTileLocs {
		xPositions = append(xPositions, int(x))
	}
	sort.Ints(xPositions)
	for _, x := range xPositions {
		yLocs := b.UsedTileLocs[tile.X(x)]
		yPositions := make([]int, 0, len(yLocs))
		for y := range yLocs {
			yPositions = append(yPositions, int(y))
		}
		sort.Ints(yPositions)
		for _, y := range yPositions {
			t := yLocs[tile.Y(y)]
			tp := b.UsedTiles[t.ID]
			usedTiles = append(usedTiles, tp)
		}
	}
	return usedTiles
}

// AddTile adds a tile to the board's unused tiles.
// An error is returned and the tile is not added if the player already has it.
func (b *Board) AddTile(t tile.Tile) error {
	if b.hasTile(t) {
		return errors.New("player already has tile with id " + strconv.Itoa(int(t.ID)))
	}
	b.UnusedTiles[t.ID] = t
	b.UnusedTileIDs = append(b.UnusedTileIDs, t.ID)
	return nil
}

// RemoveTile removes a single tile from the board's tiles.
// An error is returned if the board does not have the tile.
func (b *Board) RemoveTile(t tile.Tile) error {
	if !b.hasTile(t) {
		return errors.New("player does not have tile with id " + strconv.Itoa(int(t.ID)))
	}
	_, ok := b.UnusedTiles[t.ID]
	switch {
	case ok:
		b.removeUnusedTile(t)
	default:
		b.removeUsedTile(t)
	}
	return nil
}

// removeUnusedTile removes a tile from the unused tiles.
func (b *Board) removeUnusedTile(t tile.Tile) {
	delete(b.UnusedTiles, t.ID)
	for i, id2 := range b.UnusedTileIDs {
		if t.ID == id2 {
			b.UnusedTileIDs = append(b.UnusedTileIDs[:i], b.UnusedTileIDs[i+1:]...)
			return
		}
	}
}

// removeUsedTile removes a tile from the used tiles.
func (b *Board) removeUsedTile(t tile.Tile) {
	tp := b.UsedTiles[t.ID]
	delete(b.UsedTiles, t.ID)
	delete(b.UsedTileLocs[tp.X], tp.Y)
	if len(b.UsedTileLocs[tp.X]) == 0 {
		delete(b.UsedTileLocs, tp.X)
	}
}

// MoveTiles moves the tiles to the specified positions.
// No action is taken and an error is returned if the tiles cannot be moved.
func (b *Board) MoveTiles(tilePositions []tile.Position) error {
	if !b.CanMoveTiles(tilePositions) {
		return errors.New("cannot move tiles that the player does not have or cannot move tiles to the same spot as others")
	}
	for _, tp := range tilePositions {
		_, tileUnused := b.UnusedTiles[tp.Tile.ID]
		switch {
		case tileUnused:
			b.removeUnusedTile(tp.Tile)
		default:
			oldTp := b.UsedTiles[tp.Tile.ID]
			if b.UsedTileLocs[oldTp.X][oldTp.Y].ID == tp.Tile.ID {
				delete(b.UsedTileLocs[oldTp.X], oldTp.Y)
				if len(b.UsedTileLocs[oldTp.X]) == 0 {
					delete(b.UsedTileLocs, oldTp.X)
				}
			}
		}
		if _, ok := b.UsedTileLocs[tp.X]; !ok {
			b.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile, 1)
		}
		b.UsedTileLocs[tp.X][tp.Y] = tp.Tile
		b.UsedTiles[tp.Tile.ID] = tp
	}
	return nil
}

// CanMoveTiles determines if the player's tiles can be moved to/in the used area
// without overlapping any other tiles
func (b Board) CanMoveTiles(tilePositions []tile.Position) bool {
	ids := make(map[tile.ID]struct{}, len(tilePositions))
	positions := make(map[tile.X]map[tile.Y]struct{}, len(b.UsedTileLocs))
	for _, tp := range tilePositions {
		// ensure the tile position and id is valid
		switch {
		case tp.X < 0, tp.Y < 0, int(tp.X) >= b.NumCols, int(tp.Y) >= b.NumRows,
			!b.hasTile(tp.Tile):
			return false
		}
		if _, ok := ids[tp.Tile.ID]; ok {
			return false
		}
		ids[tp.Tile.ID] = struct{}{}
		// ensure no other tile is moved to the same spot
		if _, ok := positions[tp.X]; !ok {
			positions[tp.X] = make(map[tile.Y]struct{}, 1)
		} else if _, ok := positions[tp.X][tp.Y]; ok {
			return false
		}
		positions[tp.X][tp.Y] = struct{}{}
	}
	return b.canLeaveUsedTiles(ids, positions)
}

// canLeaveUsedTiles determines if all of the used tiles on the board that are not in movedTilesIDs
// will not be replaced by any of the tiles being moved.
func (b Board) canLeaveUsedTiles(movedTileIDs map[tile.ID]struct{}, movedTilePositions map[tile.X]map[tile.Y]struct{}) bool {
	for t2ID, tp2 := range b.UsedTiles {
		if _, ok := movedTileIDs[t2ID]; !ok {
			if movedTilePositionsY, ok := movedTilePositions[tp2.X]; ok {
				if _, ok := movedTilePositionsY[tp2.Y]; ok {
					return false
				}
			}
		}
	}
	return true
}

// HasSingleUsedGroup determines if the player's tiles form a single group,
// with all tiles connected via immediate horizontal and vertical neighbors
func (b *Board) HasSingleUsedGroup() bool {
	if len(b.UsedTiles) == 0 {
		return false
	}
	seenTileIds := make(map[tile.ID]struct{})
	for x, yTiles := range b.UsedTileLocs {
		for y, t := range yTiles {
			b.addSeenTileIDs(int(x), int(y), t, seenTileIds)
			break // only check one tile's surrounding tilePositions
		}
		break
	}
	return len(seenTileIds) == len(b.UsedTiles)
}

// UsedTileWords computes all the horizontal and vertical words formed by used tiles.
func (b Board) UsedTileWords() []string {
	horizontalWords := b.usedTileWordsX()
	verticalWords := b.usedTileWordsY()
	return append(horizontalWords, verticalWords...)
}

// usedTileWordsY computes all the vertical words formed by used tiles.
func (b Board) usedTileWordsY() []string {
	usedTilesXy := make(map[int]map[int]tile.Tile)
	for x, yTiles := range b.UsedTileLocs {
		xi := int(x)
		usedTilesXy[xi] = make(map[int]tile.Tile, len(yTiles))
		for y, t := range yTiles {
			usedTilesXy[xi][int(y)] = t
		}
	}
	return b.usedTileWordsZ(usedTilesXy, tilePositionY)
}

// usedTileWordsX computes all the horizontal words formed by used tiles.
func (b Board) usedTileWordsX() []string {
	usedTilesYx := make(map[int]map[int]tile.Tile)
	for x, yTiles := range b.UsedTileLocs {
		xi := int(x)
		for y, t := range yTiles {
			yi := int(y)
			if _, ok := usedTilesYx[yi]; !ok {
				usedTilesYx[yi] = make(map[int]tile.Tile)
			}
			usedTilesYx[yi][xi] = t
		}
	}
	return b.usedTileWordsZ(usedTilesYx, tilePositionX)
}

// tilePositionY returns the y position of thhe tile as an int.
func tilePositionY(tp tile.Position) int {
	return int(tp.Y)
}

// tilePositionX returns the x position of thhe tile as an int.
func tilePositionX(tp tile.Position) int {
	return int(tp.X)
}

// keyedUsedWords aggregates the words in the direction specified by ord into a map of words foreach row in the tiles
// the total number of words is also returned
func (b Board) keyedUsedWords(tiles map[int]map[int]tile.Tile, ord func(tp tile.Position) int) (map[int][]string, int) {
	keyedUsedWords := make(map[int][]string, len(tiles))
	wordCount := 0
	for z, zTiles := range tiles {
		tilePositions := make([]tile.Position, 0, len(zTiles))
		for _, t := range zTiles {
			tilePositions = append(tilePositions, b.UsedTiles[t.ID])
		}
		sort.Slice(tilePositions, func(i, j int) bool {
			return ord(tilePositions[i]) < ord(tilePositions[j])
		})
		buffer := new(bytes.Buffer)
		var zWords []string
		for i, tp := range tilePositions {
			if i > 0 && ord(tilePositions[i-1]) < ord(tp)-1 {
				if buffer.Len() > 1 {
					zWords = append(zWords, buffer.String())
				}
				buffer = new(bytes.Buffer)
			}
			buffer.WriteString(string(tp.Tile.Ch))
		}
		if buffer.Len() > 1 {
			zWords = append(zWords, buffer.String())
		}
		keyedUsedWords[z] = zWords
		wordCount += len(zWords)
	}
	return keyedUsedWords, wordCount
}

// usedTileWords computes all the words formed by used tiles in the direction specified by the ord function.
func (b Board) usedTileWordsZ(tiles map[int]map[int]tile.Tile, ord func(tp tile.Position) int) []string {
	keyedUsedWords, wordCount := b.keyedUsedWords(tiles, ord)
	//sort the keyedUsedWords by the keys (z)
	keys := make([]int, 0, len(keyedUsedWords))
	for k := range keyedUsedWords {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	usedWords := make([]string, wordCount)
	i := 0
	for _, k := range keys {
		copy(usedWords[i:], keyedUsedWords[k])
		i += len(keyedUsedWords[k])
	}
	return usedWords
}

// addSeenTileIds does a depth-first search for surrounding tiles, modifying the seenTileIds map.
func (b Board) addSeenTileIDs(x, y int, t tile.Tile, seenTileIds map[tile.ID]struct{}) {
	seenTileIds[t.ID] = struct{}{}
	for dx := -1; dx <= 1; dx++ { // check neighboring columns
		for dy := -1; dy <= 1; dy++ { // check neighboring rows
			if (dx != 0 || dy != 0) && dx*dy == 0 { // one delta is not zero, the other is
				b.addSurroundingSeenTileID(x+dx, y+dy, seenTileIds)
			}
		}
	}
}

// addSurroundingSeenTileID calls addSeenTilesIds for the tile at the location, if it exists.
func (b *Board) addSurroundingSeenTileID(x, y int, seenTileIds map[tile.ID]struct{}) {
	if yTiles, ok := b.UsedTileLocs[tile.X(x)]; ok { // x is valid
		if t2, ok := yTiles[tile.Y(y)]; ok { // y is valid
			if _, ok := seenTileIds[t2.ID]; !ok { // tile not yet seen
				b.addSeenTileIDs(x, y, t2, seenTileIds) // recursive call
			}
		}
	}
}

// hasTile returns true if the board has any unused or used tiles with the same id.
func (b Board) hasTile(t tile.Tile) bool {
	if _, ok := b.UnusedTiles[t.ID]; ok {
		return true
	}
	if _, ok := b.UsedTiles[t.ID]; ok {
		return true
	}
	return false
}

// Resize rezises the board to use the new config.  Any board size change information is returned in the message.
func (b *Board) Resize(cfg Config) (*ResizeResult, error) {
	usedTilePositions := make([]tile.Position, 0, len(b.UsedTiles))
	var movedTiles []tile.Tile
	for _, tp := range b.UsedTiles {
		switch {
		case cfg.NumCols <= int(tp.X), cfg.NumRows <= int(tp.Y):
			if err := b.RemoveTile(tp.Tile); err != nil {
				return nil, err
			}
			if err := b.AddTile(tp.Tile); err != nil {
				return nil, err
			}
			movedTiles = append(movedTiles, tp.Tile)
		default:
			usedTilePositions = append(usedTilePositions, tp)
		}
	}
	b.Config = cfg
	sort.Slice(usedTilePositions, func(i, j int) bool {
		a, b := usedTilePositions[i], usedTilePositions[j]
		// top-bottom, left-right
		switch {
		case a.Y == b.Y:
			return a.X < b.X
		default:
			return a.Y > b.Y
		}
	})
	unusedTiles := make([]tile.Tile, len(b.UnusedTiles))
	for i, id := range b.UnusedTileIDs {
		unusedTiles[i] = b.UnusedTiles[id]
	}
	rr := ResizeResult{
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
	}
	if len(movedTiles) > 0 {
		rr.Info = "moving " + strconv.Itoa(len(movedTiles)) + " tile(s) to the unused area of the narrower/shorter board"
	}
	return &rr, nil
}
