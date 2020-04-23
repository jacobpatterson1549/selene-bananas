package game

import (
	"encoding/json"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

func TestMessageJSON(t *testing.T) {
	messageJSONTests := []struct {
		m messager
		j string
	}{
		{
			m: messageType(1),
			j: `{"type":1}`,
		},
		{
			m: infoMessage{Type: 4, Info: "Selene started the game."},
			j: `{"type":4,"body":"Selene started the game."}`,
		},
		{
			m: tilesMessage{Type: 5, Info: "Selene snagged a tile.  You got a 'X'.", Tiles: []tile{{ID: 7, Ch: 'X'}}},
			j: `{"type":5,"body":{"info":"Selene snagged a tile.  You got a 'X'.","tiles":[{"id":7,"ch":"X"}]}}`,
		},
		{
			m: tilesMessage{Type: 6, Tiles: []tile{{ID: 9, Ch: 'Q'}}},
			j: `{"type":6,"body":{"tiles":[{"id":9,"ch":"Q"}]}}`,
		},
		{
			m: tilesMessage{Type: 6, Info: "Selene swapped a 'Q' for ['A','B','C'].", Tiles: []tile{{ID: 3, Ch: 'A'}, {ID: 1, Ch: 'B'}, {ID: 7, Ch: 'C'}}},
			j: `{"type":6,"body":{"info":"Selene swapped a 'Q' for ['A','B','C'].","tiles":[{"id":3,"ch":"A"},{"id":1,"ch":"B"},{"id":7,"ch":"C"}]}}`,
		},
		{
			m: tilePositionsMessage([]tilePosition{{Tile: tile{ID: 8, Ch: 'R'}, X: 3, Y: 47}}),
			j: `{"type":9,"body":[{"tile":{"id":8,"ch":"R"},"x":3,"y":47}]}`,
		},
		{
			m: gameInfosMessage([]gameInfo{{Players: []db.Username{"fred", "barney"}, CanJoin: true, CreatedAt: "long_ago"}}),
			j: `{"type":10,"body":[{"players":["fred","barney"],"canJoin":true,"createdAt":"long_ago"}]}`,
		},
		{
			m: userRemoveMessage("selene"),
			j: `{"type":11,"body":"selene"}`,
		},
	}
	for i, test := range messageJSONTests {
		m1, err := test.m.message()
		if err != nil {
			t.Errorf("Test %v: could not create message: %v", i, err)
			continue
		}
		j2, err := json.Marshal(m1)
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
		case m1.Type != m2.Type:
			t.Errorf("Test %v (Unmarshal): expected messageType to be:\n%v\nbut was:\n%v", i, m1.Type, m2.Type)
		case string(m1.Content) != string(m2.Content):
			t.Errorf("Test %v (Unmarshal): expected message to be:\n%v\nbut was:\n%v", i, string(m1.Content), string(m2.Content))
		}
	}
}
