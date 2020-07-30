package controller

import (
	"testing"
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
