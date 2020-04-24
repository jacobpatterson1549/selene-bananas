package game

import (
	"sort"
	"testing"
)

func TestCreateTiles_correctAmount(t *testing.T) {
	g := game{
		shuffleTilesFunc: func(tiles []tile) {},
	}
	tiles := g.createTiles()
	want := 144
	got := 0
	for _, t := range tiles {
		if t.Ch != 0 {
			got++
		}
	}
	if want != got {
		t.Errorf("wanted %v tiles, but got %v", want, got)
	}
}

func TestCreateTiles_allLetters(t *testing.T) {
	g := game{
		shuffleTilesFunc: func(tiles []tile) {},
	}
	tiles := g.createTiles()
	m := make(map[letter]bool, 26)
	for _, v := range tiles {
		ch := v.Ch
		if ch < 'A' || ch > 'Z' {
			t.Errorf("invalid tile: %v", v)
		}
		m[ch] = true
	}
	want := 26
	got := len(m)
	if want != got {
		t.Errorf("wanted %v different letters, but got %v", want, got)
	}
}

func TestCreateTiles_shuffled(t *testing.T) {
	createTilesShuffledTests := []struct {
		want      letter
		inReverse string
	}{
		{'A', ""},
		{'Z', " IN REVERSE"},
	}
	for _, test := range createTilesShuffledTests {
		g1 := game{
			shuffleTilesFunc: func(tiles []tile) {
				sort.Slice(tiles, func(i, j int) bool {
					lessThan := tiles[i].Ch < tiles[j].Ch
					if len(test.inReverse) > 0 {
						return !lessThan
					}
					return lessThan
				})
			},
		}
		g1Tiles := g1.createTiles()
		got := g1Tiles[0].Ch
		if test.want != got {
			t.Errorf("expected first tile to be %q when sorted%v (a fake shuffle), but was %q", test.want, test.inReverse, got)
		}
	}
}

func TestCreateTiles_uniqueIds(t *testing.T) {
	g := game{
		shuffleTilesFunc: func(tiles []tile) {},
	}
	tiles := g.createTiles()
	tileIds := make(map[int]bool, len(tiles))
	for _, tile := range tiles {
		if _, ok := tileIds[tile.ID]; ok {
			t.Errorf("tile id %v repeated", tile.ID)
		}
		tileIds[tile.ID] = true
	}
}
