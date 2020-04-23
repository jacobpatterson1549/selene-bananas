package game

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMessageJSON(t *testing.T) {
	messageJSONTests := []struct {
		m message
		j string
	}{
		{
			m: message{Type: 1},
			j: `{"type":1}`,
		},
		{
			m: message{Type: 2, Info: "Selene started the game."},
			j: `{"type":2,"message":"Selene started the game."}`,
		},
		{
			m: message{Type: 9, Info: "Selene snagged a tile.  You got a 'X'.", Tiles: []tile{'X'}},
			j: `{"type":9,"message":"Selene snagged a tile.  You got a 'X'.","tiles":["X"]}`,
		},
		{
			m: message{Type: 9, Tiles: []tile{'Q'}},
			j: `{"type":9,"tiles":["Q"]}`,
		},
		{
			m: message{Type: 9, Info: "Selene swapped a 'Q' for ['A','B','C'].", Tiles: []tile{'A', 'B', 'C'}},
			j: `{"type":9,"message":"Selene swapped a 'Q' for ['A','B','C'].","tiles":["A","B","C"]}`,
		},
	}
	for i, test := range messageJSONTests {
		j2, err := json.Marshal(test.m)
		switch {
		case err != nil:
			t.Errorf("Test %v (Marshal): unexpected error while marshalling message '%v': %v", i, test.m, err)
		case test.j != string(j2):
			t.Errorf("Test %v (Marshal): expected json to be:\n%v\nbut was:\n%v", i, test.j, string(j2))
		}
		var m2 message
		err = json.Unmarshal([]byte(test.j), &m2)
		switch {
		case err != nil:
			t.Errorf("Test %v (Unmarshal): unexpected error while unmarshalling json '%v': %v", i, test.j, err)
		case !reflect.DeepEqual(test.m, m2):
			t.Errorf("Test %v (Unmarshal): expected message to be:\n%v\nbut was:\n%v", i, test.m, m2)
		}
	}
}

func TestMessage_unmarshalBadTile(t *testing.T) {
	j := `{"type":9,"tiles":["XYZ"]}`
	var m message
	err := json.Unmarshal([]byte(j), &m)
	switch {
	case err == nil:
		t.Errorf("expected unmarshal of %v to fail because the tile is invalid, but produced %v", j, m)
	case !strings.Contains(err.Error(), "invalid tile"):
		t.Errorf("expected unmarshal error to be about an invalid tile, but got: %v", err)
	}
}

func TestMessage_unmarshalBadJson(t *testing.T) {
	j := `9`
	var m message
	err := json.Unmarshal([]byte(j), &m)
	if err == nil {
		t.Errorf("expected unmarshal of %v to fail because the tile is invalid, but produced %v", j, m)
	}
}
