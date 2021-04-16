package board

import (
	"encoding/json"

	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

// jsonBoard is used for serialization with the json/encoding package
type jsonBoard struct {
	Tiles         []tile.Tile     `json:"tiles,omitempty"`
	TilePositions []tile.Position `json:"tilePositions,omitempty"`
	Config        *Config         `json:"config,omitempty"`
}

// New creates a new board with the tiles as unusedTiles and tilePositions as used tiles.
func New(tiles []tile.Tile, tilePositions []tile.Position) *Board {
	jb := jsonBoard{
		Tiles:         tiles,
		TilePositions: tilePositions,
	}
	return jb.Board()
}

// MarshalJSON implements the encoding/json.Marshaler interface.
// Returns an object containing the array of unused tiles, map of tile positions, and the board config.
func (b Board) MarshalJSON() ([]byte, error) {
	unusedTiles := b.sortedUnusedTiles()
	usedTiles := b.sortedUsedTiles()
	jb := jsonBoard{
		Tiles:         unusedTiles,
		TilePositions: usedTiles,
	}
	if b.Config.NumRows != 0 || b.Config.NumCols != 0 { // do not marshal zero value
		jb.Config = &b.Config
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
	*b = *jb.Board()
	return nil
}

// Board creates a new Board from the JsonBoard.
func (jb jsonBoard) Board() *Board {
	var b Board
	b.UnusedTiles = make(map[tile.ID]tile.Tile, len(jb.Tiles))
	b.UnusedTileIDs = make([]tile.ID, 0, len(jb.Tiles))
	for _, t := range jb.Tiles {
		b.UnusedTiles[t.ID] = t
		b.UnusedTileIDs = append(b.UnusedTileIDs, t.ID)
	}
	b.UsedTiles = make(map[tile.ID]tile.Position, len(jb.TilePositions))
	b.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
	for _, tp := range jb.TilePositions {
		b.UsedTiles[tp.Tile.ID] = tp
		if _, ok := b.UsedTileLocs[tp.X]; !ok {
			b.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
		}
		b.UsedTileLocs[tp.X][tp.Y] = tp.Tile
	}
	if jb.Config != nil {
		b.Config = *jb.Config
	}
	return &b
}
