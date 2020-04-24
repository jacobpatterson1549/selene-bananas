package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

func TestMessageJSON(t *testing.T) {
	messageJSONTests := []struct {
		m message
		j string
	}{
		{
			m: message{Type: gameJoin, GameID: 6},
			j: `{"type":2,"gameID":6}`,
		},
		{
			m: message{Type: gameStart, Info: "Selene started the game."},
			j: `{"type":5,"info":"Selene started the game."}`,
		},
		{
			m: message{Type: gameSnag, Info: "Selene snagged a tile.  You got a 'X'.", Tiles: []tile{{ID: 7, Ch: 'X'}}},
			j: `{"type":7,"info":"Selene snagged a tile.  You got a 'X'.","tiles":[{"id":7,"ch":"X"}]}`,
		},
		{
			m: message{Type: gameSnag, Tiles: []tile{{ID: 9, Ch: 'Q'}}},
			j: `{"type":7,"tiles":[{"id":9,"ch":"Q"}]}`,
		},
		{
			m: message{Type: gameSwap, Info: "Selene swapped a 'Q' for ['A','B','C'].", Tiles: []tile{{ID: 3, Ch: 'A'}, {ID: 1, Ch: 'B'}, {ID: 7, Ch: 'C'}}},
			j: `{"type":8,"info":"Selene swapped a 'Q' for ['A','B','C'].","tiles":[{"id":3,"ch":"A"},{"id":1,"ch":"B"},{"id":7,"ch":"C"}]}`,
		},
		{
			m: message{Type: gameTileMoved, TilePositions: []tilePosition{{Tile: tile{ID: 8, Ch: 'R'}, X: 3, Y: 47}, {Tile: tile{ID: 8, Ch: 'R'}, X: 4, Y: 46}}},
			j: `{"type":9,"tilePositions":[{"tile":{"id":8,"ch":"R"},"x":3,"y":47},{"tile":{"id":8,"ch":"R"},"x":4,"y":46}]}`,
		},
		{
			m: message{Type: gameTilePositions, TilePositions: []tilePosition{{Tile: tile{ID: 8, Ch: 'R'}, X: 3, Y: 47}}},
			j: `{"type":10,"tilePositions":[{"tile":{"id":8,"ch":"R"},"x":3,"y":47}]}`,
		},
		{
			m: message{Type: gameInfos, GameInfos: []gameInfo{{Players: []db.Username{"fred", "barney"}, CanJoin: true, CreatedAt: "long_ago"}}},
			j: `{"type":11,"gameInfos":[{"players":["fred","barney"],"canJoin":true,"createdAt":"long_ago"}]}`,
		},
		{
			m: message{Type: gameInfos},
			j: `{"type":11}`,
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

func TestMessageMarshal_omitInternals(t *testing.T) {
	m := message{Player: &player{}, Game: &game{}, GameInfoChan: make(chan gameInfo, 0)}
	want := []byte(`{"type":0}`)
	got, err := json.Marshal(m)
	switch {
	case err != nil:
		t.Errorf("unexpected exception: %v", err)
	case !reflect.DeepEqual(want, got):
		t.Errorf("wanted %v, got %v", string(want), string(got))
	}
}
