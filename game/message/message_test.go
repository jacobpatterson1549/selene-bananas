package message

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestMessageJSON(t *testing.T) {
	newBoard := func(tiles []tile.Tile, tilePositions []tile.Position, config *board.Config) *board.Board {
		b := board.New(tiles, tilePositions)
		if config != nil {
			b.Config = *config
		}
		return b
	}
	MessageJSONTests := []struct {
		m Message
		j string
	}{
		{
			j: `{"type":0}`, // the MessageType should always be marshalled
		},
		{
			m: Message{Type: 2, Game: &game.Info{ID: 6}},
			j: `{"type":2,"game":{"id":6}}`,
		},
		{
			m: Message{Type: 5, Info: "Selene started the game."},
			j: `{"type":5,"info":"Selene started the game."}`,
		},
		{
			m: Message{Type: 7, Info: "Selene snagged a tile.  You got a 'X'.", Game: &game.Info{Board: newBoard([]tile.Tile{{ID: 7, Ch: 'X'}}, nil, nil)}},
			j: `{"type":7,"info":"Selene snagged a tile.  You got a 'X'.","game":{"board":{"tiles":[{"id":7,"ch":"X"}]}}}`,
		},
		{
			m: Message{Type: 7, Game: &game.Info{Board: newBoard([]tile.Tile{{ID: 9, Ch: 'Q'}}, nil, nil)}},
			j: `{"type":7,"game":{"board":{"tiles":[{"id":9,"ch":"Q"}]}}}`,
		},
		{
			m: Message{Type: 8, Info: "Selene swapped a 'Q' for ['A','B','C'].", Game: &game.Info{Board: newBoard([]tile.Tile{{ID: 3, Ch: 'A'}, {ID: 1, Ch: 'B'}, {ID: 7, Ch: 'C'}}, nil, nil)}},
			j: `{"type":8,"info":"Selene swapped a 'Q' for ['A','B','C'].","game":{"board":{"tiles":[{"id":3,"ch":"A"},{"id":1,"ch":"B"},{"id":7,"ch":"C"}]}}}`,
		},
		{
			m: Message{Type: 9, Game: &game.Info{Board: newBoard(nil, []tile.Position{{Tile: tile.Tile{ID: 8, Ch: 'R'}, X: 4, Y: 46}}, nil)}},
			j: `{"type":9,"game":{"board":{"tilePositions":[{"t":{"id":8,"ch":"R"},"x":4,"y":46}]}}}`,
		},
		{
			m: Message{Type: 3, Games: []game.Info{{ID: 7, Status: 2, Players: []string{"fred", "barney"}, CreatedAt: 1257894000}}},
			j: `{"type":3,"games":[{"id":7,"status":2,"players":["fred","barney"],"createdAt":1257894000}]}`,
		},
		{
			m: Message{Type: 11},
			j: `{"type":11}`,
		},
		{
			m: Message{Type: 1, Game: &game.Info{Board: newBoard(nil, nil, &board.Config{NumRows: 23, NumCols: 21}), Config: &game.Config{CheckOnSnag: true, Penalize: true, MinLength: 3}}},
			j: `{"type":1,"game":{"board":{"config":{"r":23,"c":21}},"config":{"checkOnSnag":true,"penalize":true,"minLength":3}}}`,
		},
	}
	for i, test := range MessageJSONTests {
		j2, err := json.Marshal(test.m)
		switch {
		case err != nil:
			t.Errorf("Test %v (Marshal): unwanted error while marshalling Message '%v': %v", i, test.m, err)
		case test.j != string(j2):
			t.Errorf("Test %v (Marshal): wanted json to be:\n%v\nbut was:\n%v", i, test.j, string(j2))
		}
		var m2 Message
		err = json.Unmarshal([]byte(test.j), &m2)
		switch {
		case err != nil:
			t.Errorf("Test %v (Unmarshal): unwanted error while unmarshalling json '%v': %v", i, test.j, err)
		case !reflect.DeepEqual(test.m, m2):
			t.Errorf("Test %v (Unmarshal): wanted Message to be:\n%v\nbut was:\n%v", i, test.m, m2)
		}
	}
}

func TestMessageMarshalOmitsInternals(t *testing.T) {
	m := Message{PlayerName: "selene"}
	want := []byte(`{"type":0}`)
	got, err := json.Marshal(m)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("wanted %v, got %v", string(want), string(got))
	}
}
