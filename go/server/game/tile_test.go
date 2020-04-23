package game

import (
	"fmt"
	"testing"
)

func TestTileString(t *testing.T) {
	tile := tile('x')
	want := "x"
	got := fmt.Sprintf("%v", tile)
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}
