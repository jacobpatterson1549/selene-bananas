// +build js,wasm

package socket

import (
	"encoding/json"
	"reflect"
	"strconv"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom/url"
)

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

func TestMessageJSON(t *testing.T) {
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
			CreatedAt: 111,
			Capacity:  11,
		},
	}
	gamePlayers := []string{
		"selene",
		"bob",
	}
	boardCfg := board.Config{
		NumCols: 7,
		NumRows: 8,
	}
	gameCfg := game.Config{
		CheckOnSnag:        true,
		Penalize:           true,
		MinLength:          9,
		ProhibitDuplicates: true,
	}
	b := board.New(tiles, tilePositions)
	b.Config = boardCfg
	m := message.Message{
		Type: message.CreateGame,
		Info: "message test",
		Game: &game.Info{
			ID:       6,
			Board:    b,
			Status:   game.InProgress,
			Players:  gamePlayers,
			Config:   &gameCfg,
			Capacity: 7,
		},
		Games: gameInfos,
	}
	wantS := `{"type":1,"info":"message test","game":{"id":6,"status":2,"board":{"tiles":[{"id":1,"ch":"A"},{"id":2,"ch":"B"}],"tilePositions":[{"t":{"id":3,"ch":"C"},"x":4,"y":5}],"config":{"c":7,"r":8}},"players":["selene","bob"],"config":{"checkOnSnag":true,"penalize":true,"minLength":9,"prohibitDuplicates":true},"capacity":7},"games":[{"id":9,"status":1,"createdAt":111,"capacity":11}]}`
	gotS, errS := json.Marshal(m)
	if errS != nil {
		t.Fatalf("stringify: %v", errS)
	}
	var wantM, gotM message.Message
	errW := json.Unmarshal([]byte(wantS), &wantM)
	if errW != nil {
		t.Fatalf("parsing wanted json: %v", errW)
	}
	errG := json.Unmarshal([]byte(gotS), &gotM)
	switch {
	case errG != nil:
		t.Errorf("parsing json: %v", errG)
	case reflect.ValueOf(gotM).IsZero():
		t.Errorf("wanted non-empty message when json is %v", gotS)
	case !reflect.DeepEqual(wantM, gotM):
		t.Errorf("not equal\nwanted %#v\ngot    %#v", wantM, gotM)
	}
}

func TestSend(t *testing.T) {
	webSocket := js.ValueOf(make(map[string]interface{}))
	webSocket.Set("readyState", 1)
	sendCalled := false
	sendFunc := func(this js.Value, args []js.Value) interface{} {
		want := `{"type":15,"info":"selene : hi","game":{"id":21}}`
		got := args[0].String()
		if want != got {
			t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
		}
		sendCalled = true
		return nil
	}
	sendJsFunc := js.FuncOf(sendFunc)
	defer sendJsFunc.Release()
	webSocket.Set("send", sendJsFunc)
	m := message.Message{Type: 15, Info: "selene : hi"}
	g := mockGame(21)
	s := Socket{
		webSocket: webSocket,
		game:      g,
	}
	s.Send(m)
	if !sendCalled {
		t.Errorf("send function not called")
	}
}

func TestOnMessage(t *testing.T) {
	t.Run("setGameInfos", func(t *testing.T) {
		mt := strconv.Itoa(int(message.GameInfos))
		eventM := map[string]interface{}{
			"data": `{"type":` + mt + `,"games":[{"id":8}]}`,
		}
		event := js.ValueOf(eventM)
		gameInfosSet := false
		lobby := mockLobby{
			SetGameInfosFunc: func(gameInfos []game.Info, username string) {
				switch {
				case len(gameInfos) != 1, gameInfos[0].ID != 8, username != "fred":
					t.Errorf("wanted infos for game 8 for fred, got: %v, %v", gameInfos, username)
				}
				gameInfosSet = true
			},
		}
		user := mockUser("fred")
		s := Socket{
			lobby: lobby,
			user:  user,
		}
		s.onMessage(event)
		if !gameInfosSet {
			t.Errorf("wanted game infos set")
		}
	})
}
