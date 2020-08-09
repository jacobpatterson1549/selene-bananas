// +build js,wasm

package socket

import (
	"encoding/json"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestParseMessageJSON(t *testing.T) {
	// TODO
}

func TestMessageToJSON(t *testing.T) {
	tiles := []tile.Tile{
		{
			ID: 1,
			Ch: 'A',
		},
		{
			ID: 2,
			Ch: 'B',
		},
	}
	tilePositions := []tile.Position{
		{
			Tile: tile.Tile{
				ID: 3,
				Ch: 'C',
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
	want := `{"type":1,"info":"message test","tiles":[{"id":1,"ch":65},{"id":2,"ch":66}],"tilePositions":[{"t":{"id":3,"ch":67},"x":4,"y":5}],"gameInfos":[{"id":9,"status":1,"players":[],"createdAt":111}],"gameID":6,"gameStatus":2,"gamePlayers":["selene","bob"],"c":7,"r":8}`
	// got := messageToJSON(m) // TODO: use this
	gotB, err := json.Marshal(m)
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	got := string(gotB)
	if want != got {
		t.Errorf("not equal\nwanted: %v\ngot:    %v", want, got)
	}
}
