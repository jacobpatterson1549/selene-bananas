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
				t.Errorf("Test %v: unexpected error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: expected error", i)
		default:
			if test.winPoints != p.winPoints {
				t.Errorf("wanted %v winPoints, got %v", test.winPoints, p.winPoints)
			}
			b.NumCols = 22
			if p.Board.NumCols != 22 {
				t.Errorf("board reference not set correctly")
			}
		}
	}
}

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
		p := Player{
			winPoints: test.winPoints,
		}
		p.DecrementWinPoints()
		got := p.winPoints
		if test.want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}

func TestWinPoints(t *testing.T) {
	want := 37
	p := Player{
		winPoints: want,
	}
	got := p.WinPoints()
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}
