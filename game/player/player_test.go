package player

import "testing"

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
