package player

import (
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
)

func TestNewPlayer(t *testing.T) {
	newPlayerTests := []struct {
		winPoints int
		wantOk    bool
	}{
		{},
		{
			winPoints: 1,
		},
		{
			winPoints: 2,
			wantOk:    true,
		},
		{
			winPoints: 10,
			wantOk:    true,
		},
	}
	for i, test := range newPlayerTests {
		var b board.Board
		cfg := Config{
			WinPoints: test.winPoints,
		}
		p, err := cfg.New(&b)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		default:
			if test.winPoints != p.WinPoints {
				t.Errorf("wanted %v winPoints, got %v", test.winPoints, p.WinPoints)
			}
			b.NumCols = 22
			if p.Board.NumCols != 22 {
				t.Errorf("board reference not set correctly")
			}
		}
	}
}
