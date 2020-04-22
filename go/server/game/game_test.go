package game

import "testing"

func TestCreateTiles_correctAmount(t *testing.T) {
	tiles := createTiles()
	want := 144
	got := 0
	for _, t := range tiles {
		if t != 0 {
			got++
		}
	}
	if want != got {
		t.Errorf("wanted %v tiles, but got %v", want, got)
	}
}

func TestCreateTiles_allLetters(t *testing.T) {
	tiles := createTiles()
	m := make(map[rune]bool, 26)
	for _, v := range tiles {
		if v < 'A' || v > 'Z' {
			t.Errorf("invalid tile: %v", v)
		}
		m[v] = true
	}
	want := 26
	got := len(m)
	if want != got {
		t.Errorf("wanted %v different tiles, but got %v", want, got)
	}
}
