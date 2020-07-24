package controller

import (
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestDecrementWinPoints(t *testing.T) {
	decrementWinPointsTests := []struct {
		winPoints int
		want      int
	}{
		{},
		{1, 1},
		{2, 2},
		{3, 2},
		{10, 9},
	}
	for i, test := range decrementWinPointsTests {
		p := player{
			winPoints: test.winPoints,
		}
		p.decrementWinPoints()
		got := p.winPoints
		if test.want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}

func TestRefreshBoard(t *testing.T) {
	boardCfg := board.Config{
		NumCols: 20,
		NumRows: 10,
	}
	t1, err := tile.New(1, 'A')
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	t2, err := tile.New(2, 'B')
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	unusedTiles := []tile.Tile{
		*t1,
		*t2,
	}
	board, err := boardCfg.New(unusedTiles)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	board.MoveTiles([]tile.Position{
		{
			Tile: *t1,
			X:    5,
			Y:    5,
		},
		{
			Tile: *t2,
			X:    15,
			Y:    5,
		},
	})
	boardCfg.NumCols = 10
	p := player{
		Board: *board,
	}
	playerName := "selene"
	var game Game
	m, err := p.refreshBoard(boardCfg, game, playerName)
	switch {
	case err != nil:
		t.Errorf("unexpected error: %v", err)
	case len(p.UnusedTileIDs) != 1, p.UnusedTileIDs[0] != t2.ID:
		t.Errorf("wanted tile 2 to be moved back to the unused area now that the board is more narrow")
	case len(m.Tiles) != 1, m.Tiles[0].ID != t2.ID:
		t.Errorf("wanted tile 2 to be moved back to the unused area now that the board is more narrow")
	case m.PlayerName != playerName:
		t.Errorf("wanted playerName in message to be %v, got %v", m.PlayerName, playerName)
	}
}
