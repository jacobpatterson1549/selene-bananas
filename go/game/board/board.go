package board

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Board represents the positioning of a player's tiles from a game.
	// (Each player has their own board)
	Board struct {
		UnusedTiles   map[tile.ID]tile.Tile
		UnusedTileIDs []tile.ID
		UsedTiles     map[tile.ID]tile.Position
		UsedTileLocs  map[tile.X]map[tile.Y]tile.Tile
	}
)

// New creates a new board with the unused tiles.
func New(unusedTiles []tile.Tile) Board {
	unusedTilesByID := make(map[tile.ID]tile.Tile, len(unusedTiles))
	unusedTileIDs := make([]tile.ID, len(unusedTiles))
	for i, t := range unusedTiles {
		unusedTilesByID[t.ID] = t
		unusedTileIDs[i] = t.ID
	}
	return Board{
		UnusedTiles:   unusedTilesByID,
		UnusedTileIDs: unusedTileIDs,
		UsedTiles:     make(map[tile.ID]tile.Position),
		UsedTileLocs:  make(map[tile.X]map[tile.Y]tile.Tile),
	}
}

// AddTile adds a tile to the board's unused tiles.
// An error is returned and the tile is not added if the player already has it.
func (b *Board) AddTile(t tile.Tile) error {
	if b.hasTile(t) {
		return fmt.Errorf("player already has tile id=%v", t.ID)
	}
	b.UnusedTiles[t.ID] = t
	b.UnusedTileIDs = append(b.UnusedTileIDs, t.ID)
	return nil
}

// RemoveTile removes a single tile from the board's tiles.
// An error is returned if the board does not have the tile.
func (b *Board) RemoveTile(t tile.Tile) error {
	if !b.hasTile(t) {
		return fmt.Errorf("player does not have tile id=%v", t.ID)
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
		return fmt.Errorf("cannot move tiles that the player does not have or cannot move tiles to the same spot as others")
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
func (b *Board) CanMoveTiles(tilePositions []tile.Position) bool {
	desiredUsedTileLocs := make(map[tile.X]map[tile.Y]struct{}, len(b.UsedTileLocs))
	var e struct{}
	movedTileIds := make(map[tile.ID]struct{}, len(tilePositions))
	for _, tp := range tilePositions {
		if !b.hasTile(tp.Tile) {
			return false
		}
		if _, ok := movedTileIds[tp.Tile.ID]; ok {
			return false
		}
		movedTileIds[tp.Tile.ID] = e
		if _, ok := desiredUsedTileLocs[tp.X]; !ok {
			desiredUsedTileLocs[tp.X] = make(map[tile.Y]struct{}, 1)
		}
		if _, ok := desiredUsedTileLocs[tp.X][tp.Y]; ok {
			return false
		}
		desiredUsedTileLocs[tp.X][tp.Y] = e
	}
	for t2ID, tp2 := range b.UsedTiles {
		if _, ok := movedTileIds[t2ID]; !ok {
			if desiredUsedTileLocsY, ok := desiredUsedTileLocs[tp2.X]; ok {
				if _, ok := desiredUsedTileLocsY[tp2.Y]; ok {
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
			b.addSeenTileIds(int(x), int(y), t, seenTileIds)
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
	return b.usedTileWordsZ(usedTilesXy, func(tp tile.Position) int { return int(tp.Y) })
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
	return b.usedTileWordsZ(usedTilesYx, func(tp tile.Position) int { return int(tp.X) })
}

// usedTileWords computes all the words formed by used tiles in the direction specified by the ord function.
func (b Board) usedTileWordsZ(tiles map[int]map[int]tile.Tile, ord func(tp tile.Position) int) []string {
	keyedUsedWords := make(map[int][]string, len(tiles))
	wordCount := 0
	for z, zTiles := range tiles {
		tilePositions := make([]tile.Position, len(zTiles))
		i := 0
		for _, t := range zTiles {
			tilePositions[i] = b.UsedTiles[t.ID]
			i++
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
			buffer.WriteRune(rune(tp.Tile.Ch))
		}
		if buffer.Len() > 1 {
			zWords = append(zWords, buffer.String())
		}
		keyedUsedWords[z] = zWords
		wordCount += len(zWords)
	}
	//sort the keyedUsedWords by the keys (z)
	keys := make([]int, len(keyedUsedWords))
	i := 0
	for k := range keyedUsedWords {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	usedWords := make([]string, wordCount)
	i = 0
	for _, k := range keys {
		copy(usedWords[i:], keyedUsedWords[k])
		i += len(keyedUsedWords[k])
	}
	return usedWords
}

// addSeenTileIds  does a depth-first search for surrounding tiles, modifying the seenTileIds map.
func (b *Board) addSeenTileIds(x, y int, t tile.Tile, seenTileIds map[tile.ID]struct{}) {
	seenTileIds[t.ID] = struct{}{}
	for dx := -1; dx <= 1; dx++ { // check neighboring columns
		for dy := -1; dy <= 1; dy++ { // check neighboring rows
			if (dx != 0 || dy != 0) && dx*dy == 0 { // one delta is not zero, the other is
				if yTiles, ok := b.UsedTileLocs[tile.X(int(x)+dx)]; ok { // x+dx is valid
					if t2, ok := yTiles[tile.Y(int(y)+dy)]; ok { // y+dy is valid
						if _, ok := seenTileIds[t2.ID]; !ok { // tile not yet seen
							b.addSeenTileIds(int(x)+dx, int(y)+dy, t2, seenTileIds) // recursive call
						}
					}
				}
			}
		}
	}
}

// hasTile returns true if the board has any unused or used tiles with the same id.
func (b *Board) hasTile(t tile.Tile) bool {
	if _, ok := b.UnusedTiles[t.ID]; ok {
		return true
	}
	if _, ok := b.UsedTiles[t.ID]; ok {
		return true
	}
	return false
}
