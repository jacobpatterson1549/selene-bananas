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
			m: infoMessage{Type: gameCreate},
			j: `{"type":1,"body":""}`, // TODO: it would be nice to not have a body
		},
		{
			m: infoMessage{Type: gameStart, Info: "Selene started the game."},
			j: `{"type":4,"body":"Selene started the game."}`,
		},
		{
			m: tilesMessage{Type: gameSnag, Info: "Selene snagged a tile.  You got a 'X'.", Tiles: []tile{{ID: 7, Ch: 'X'}}},
			j: `{"type":5,"body":{"info":"Selene snagged a tile.  You got a 'X'.","tiles":[{"id":7,"ch":"X"}]}}`,
		},
		{
			m: tilesMessage{Type: gameSnag, Tiles: []tile{{ID: 9, Ch: 'Q'}}},
			j: `{"type":5,"body":{"tiles":[{"id":9,"ch":"Q"}]}}`,
		},
		{
			m: tilesMessage{Type: gameSwap, Info: "Selene swapped a 'Q' for ['A','B','C'].", Tiles: []tile{{ID: 3, Ch: 'A'}, {ID: 1, Ch: 'B'}, {ID: 7, Ch: 'C'}}},
			j: `{"type":6,"body":{"info":"Selene swapped a 'Q' for ['A','B','C'].","tiles":[{"id":3,"ch":"A"},{"id":1,"ch":"B"},{"id":7,"ch":"C"}]}}`,
		},
		{
			m: tilesMessage{Type: gameSwap},
			j: `{"type":6,"body":{"tiles":null}}`, // TODO: it would be nice to not have a body
		},
		{
			// TODO: the plan is to use this for tile moved messages, but a new messageType should be created for that
			m: tilePositionsMessage{TilePositions: []tilePosition{{Tile: tile{ID: 8, Ch: 'R'}, X: 3, Y: 47}}},
			j: `{"type":9,"body":{"tilePositions":[{"tile":{"id":8,"ch":"R"},"x":3,"y":47}]}}`,
		},
		{
			m: tilePositionsMessage{},
			j: `{"type":9,"body":{"tilePositions":null}}`, // TODO: it would be nice to not have a body
		},
		{
			m: gameInfosMessage([]gameInfo{{Players: []db.Username{"fred", "barney"}, CanJoin: true, CreatedAt: "long_ago"}}),
			j: `{"type":10,"body":[{"players":["fred","barney"],"canJoin":true,"createdAt":"long_ago"}]}`,
		},
		{
			m: gameInfosMessage{},
			j: `{"type":10,"body":[]}`, // TODO: it would be nice to not have a body
		},
		{
			m: infoMessage{Type: gameInfos, Username: "selene"},
			j: `{"type":10,"body":""}`, // TODO: it would be nice to not have a body
		},
		{
			m: infoMessage{Type: userRemove, Username: "selene"},
			j: `{"type":11,"body":""}`,// TODO: it would be nice to not have a body
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

func TestMessageUsername(t *testing.T) {
	// ensure the username can be added to inbound messages when processing player requests in game/lobby.
	want := db.Username("selene")
	messagers := []messager{
		infoMessage{Username: want},
		tilesMessage{Username: want},         // username and tile needed for swap
		tilePositionsMessage{Username: want}, // username and tiles needed for tile move message
	}
	for i, m := range messagers {
		message, err := m.message()
		switch {
		case err != nil:
			t.Errorf("Test %v (%T) unexpected error: %v", i, m, err)
		default:
			got := message.Username
			if want != got {
				t.Errorf("Test %v (%T) wanted %v, got %v", i, m, want, got)
			}
		}
	}
}
