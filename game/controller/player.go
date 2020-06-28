package controller

import (
	"fmt"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

type (
	player struct {
		winPoints
		board.Board
	}

	winPoints int
)

func (p *player) decrementWinPoints() {
	if p.winPoints > 2 {
		p.winPoints--
	}
}

// refreshBoard builds a message with the state of the board.
// The board config is used to reside the board, moving tiles that would not lie in the previous board's space to the unused area.
func (p *player) refreshBoard(cfg board.Config, g *Game, n game.PlayerName) (*game.Message, error) {
	unusedTiles := make([]tile.Tile, len(p.UnusedTiles))
	for i, id := range p.UnusedTileIDs {
		unusedTiles[i] = p.UnusedTiles[id]
	}
	usedTilePositions := make([]tile.Position, 0, len(p.UsedTiles))
	var movedTiles []tile.Tile
	for _, tp := range p.UsedTiles {
		switch {
		case cfg.NumCols <= int(tp.X), cfg.NumRows <= int(tp.Y):
			if err := p.RemoveTile(tp.Tile); err != nil {
				return nil, fmt.Errorf("removing used tile to move to unused area for smaller board: %v", err)
			}
			if err := p.AddTile(tp.Tile); err != nil {
				return nil, fmt.Errorf("adding used tile to unused area on smaller board: %v", err)
			}
			movedTiles = append(movedTiles, tp.Tile)
		default:
			usedTilePositions = append(usedTilePositions, tp)
		}
	}
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
	m := game.Message{
		Type:          game.Join,
		PlayerName:    n,
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
		TilesLeft:     len(g.unusedTiles),
		GameStatus:    g.status,
		GamePlayers:   g.playerNames(),
		GameID:        g.id,
	}
	if len(movedTiles) > 0 {
		m.Info = fmt.Sprintf("moving %v tiles to unused are of narrower/shorter board", len(movedTiles))
	}
	return &m, nil
}
