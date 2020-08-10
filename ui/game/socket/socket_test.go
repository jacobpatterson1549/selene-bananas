// +build js,wasm

package socket

import (
	"net/url"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom/json"
)

type mockUser string

func (u mockUser) JWT() string {
	return string(u)
}

func (u mockUser) Username() string {
	return ""
}

func (u *mockUser) Logout() {
	// NOOP
}

func TestReleaseWebSocketJsFuncs(t *testing.T) {
	var s Socket
	// it should be ok to release the functions multiple times, even if they are undefined/null
	s.releaseWebSocketJsFuncs()
	s.releaseWebSocketJsFuncs()
}

func TestWebSocketURL(t *testing.T) {
	getWebSocketURLTests := []struct {
		url  string
		jwt  string
		want string
	}{
		{
			url:  "http://127.0.0.1:8000/user_join_lobby",
			jwt:  "a.jwt.token",
			want: "ws://127.0.0.1:8000/user_join_lobby?access_token=a.jwt.token",
		},
		{
			url:  "https://example.com",
			jwt:  "XYZ",
			want: "wss://example.com?access_token=XYZ",
		},
	}
	for i, test := range getWebSocketURLTests {
		u, err := url.Parse(test.url)
		if err != nil {
			t.Errorf("Test %v: %v", i, err)
			continue
		}
		f := dom.Form{
			URL:    *u,
			Params: make(url.Values, 1),
		}
		mu := mockUser(test.jwt)
		s := Socket{
			user: &mu,
		}
		got := s.webSocketURL(f)
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestParseMessageJSON(t *testing.T) {
	// TODO
}

func TestMessageToJSON(t *testing.T) {
	t.Skip() // TODO
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
	want := `{"type":1,"info":"message test","tiles":[{"id":1,"ch":"A"},{"id":2,"ch":"B"}],"tilePositions":[{"t":{"id":3,"ch":"C"},"x":4,"y":5}],"gameInfos":[{"id":9,"status":1,"players":[],"createdAt":111}],"gameID":6,"gameStatus":2,"gamePlayers":["selene","bob"],"c":7,"r":8}`
	got, err := json.Stringify(m)
	switch {
	case err != nil:
		t.Errorf("unwanted error: %v", err)
	// TODO: call w, _ := json.Parse(want); g, _ := json.Parse(got); if !reflect.DeepEqual(want, got) { ... }
	case want != got:
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}
