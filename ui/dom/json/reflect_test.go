package json

import (
	"reflect"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestToMap(t *testing.T) {
	parseUserInfoJSONTests := []struct {
		src     interface{}
		wantErr bool
		want    interface{}
	}{
		{},
		{
			src:  7,
			want: 7,
		},
		{
			src:  9223372036854775807,
			want: 9223372036854775807,
		},
		{
			src:  "some text",
			want: "some text",
		},
		{
			src:  []string{"a", "b", "c"},
			want: []interface{}{"a", "b", "c"},
		},
		{
			src: struct {
				Name string
			}{
				Name: "selene",
			},
			wantErr: true, // no json tag for struct field
		},
		{
			src: struct {
				Name string `json:"id"`
			}{
				Name: "selene",
			},
			want: map[string]interface{}{
				"id": "selene",
			},
		},
	}
	for i, test := range parseUserInfoJSONTests {
		got, err := toMap(test.src)
		switch {
		case err != nil:
			if !test.wantErr {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case test.wantErr:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, got)
		}
	}
}

func TestMessageToJSON(t *testing.T) {
	tiles := []tile.Tile{
		{
			ID: 1,
			Ch: "A",
		},
		{
			ID: 2,
			Ch: "B",
		},
	}
	tilePositions := []tile.Position{
		{
			Tile: tile.Tile{
				ID: 3,
				Ch: "C",
			},
			X: 4,
			Y: 5,
		},
	}
	gameInfos := []game.Info{
		{
			ID:        9,
			Status:    game.NotStarted,
			Players:   []string{},
			CreatedAt: 111,
		},
	}
	gamePlayers := []string{
		"selene",
		"bob",
	}
	m := game.Message{
		Type:          game.Create,
		Info:          "message test",
		Tiles:         tiles,
		TilePositions: tilePositions,
		GameInfos:     gameInfos,
		GameID:        6,
		GameStatus:    game.InProgress,
		GamePlayers:   gamePlayers,
		NumCols:       7,
		NumRows:       8,
	}
	// want := `{"type":1,"info":"message test","tiles":[{"id":1,"ch":65},{"id":2,"ch":66}],"tilePositions":[{"t":{"id":3,"ch":67},"x":4,"y":5}],"gameInfos":[{"id":9,"status":1,"players":[],"createdAt":111}],"gameID":6,"gameStatus":2,"gamePlayers":["selene","bob"],"c":7,"r":8}`
	_, err := toMap(m)
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
}
