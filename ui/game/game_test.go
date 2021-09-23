//go:build js && wasm

package game

import (
	"reflect"
	"strings"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/canvas"
)

func TestNewConfig(t *testing.T) {
	dom := new(mockDOM)
	log := new(mockLog)
	board := new(board.Board)
	canvas := new(canvas.Canvas)
	cfg := Config{board, canvas}
	want := &Game{
		dom:    dom,
		log:    log,
		board:  board,
		canvas: canvas,
	}
	got := cfg.New(dom, log)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestStartCreate(t *testing.T) {
	n := 0
	g := Game{
		dom: &mockDOM{
			SetCheckedFunc: func(query string, checked bool) {
				if want, got := !strings.Contains(query, "hide"), checked; want != got {
					t.Errorf("wanted setChecked(%v) to be called with %v, got %v", query, want, got)
				}
				n++
			},
		},
	}
	g.startCreate()
	if want, got := 3, n; want != got {
		t.Errorf("wanted %v calls to setChecked, got %v", want, got)
	}
}

func TestCreateWithConfig(t *testing.T) {
	t.Skip("TODO")
}

func TestJoin(t *testing.T) {
	t.Skip("TODO")
}

func TestHide(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		setCheckedCalled := false
		g := Game{
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					if want, got := want, checked; want != got {
						t.Errorf("Test %v: wanted setChecked to be called with %v, got %v", i, want, got)
					}
					setCheckedCalled = true
				},
			},
		}
		g.hide(want)
		if !setCheckedCalled {
			t.Errorf("Test %v: wanted dom element to be checked/unchecked", i)
		}
	}
}

func TestID(t *testing.T) {
	want := game.ID(1549)
	g := Game{
		id: want,
	}
	got := g.ID()
	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestSendLeave(t *testing.T) {
	messageSent := false
	g := Game{
		dom: &mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{})
			},
			SetCheckedFunc: func(query string, checked bool) {
				if want, got := true, checked; want != got {
					t.Errorf("wanted setChecked(%v) to be called with %v, got %v", query, want, got)
				}
			},
		},
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.LeaveGame, m.Type; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.sendLeave()
	if !messageSent {
		t.Error("wanted message to be sent")
	}
}

func TestLeave(t *testing.T) {
	setCheckedCallCount := 0
	g := Game{
		id: 1,
		dom: &mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{})
			},
			SetCheckedFunc: func(query string, checked bool) {
				if want, got := true, checked; want != got {
					t.Errorf("wanted setChecked(%v) to be called with %v, got %v", query, want, got)
				}
				setCheckedCallCount++
			},
		},
	}
	g.Leave()
	switch {
	case g.id != 0:
		t.Errorf("wanted game id to be set to 0, got %v", g.id)
	case setCheckedCallCount != 3:
		t.Errorf("wanted setChecked to be called 3 times, got %v", setCheckedCallCount)
	}
}

func TestDelete(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		confirmCalled := false
		messageSent := false
		g := Game{
			dom: &mockDOM{
				ConfirmFunc: func(message string) bool {
					confirmCalled = true
					return want
				},
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := message.DeleteGame, m.Type; want != got {
						t.Errorf("not equal: wanted %v, got %v", want, got)
					}
					messageSent = true
				},
			},
		}
		g.delete()
		if !confirmCalled {
			t.Errorf("Test %v: wanted confirm to be called", i)
		}
		if want, got := want, messageSent; want != got {
			t.Errorf("Test %v: wanted message to be sent: %v, got %v", i, want, got)
		}
	}
}

func TestStart(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.ChangeGameStatus, m.Type; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				if want, got := game.InProgress, m.Game.Status; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.Start()
	if !messageSent {
		t.Error("wanted message to be sent")
	}
}

func TestFinish(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.ChangeGameStatus, m.Type; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				if want, got := game.Finished, m.Game.Status; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.finish()
	if !messageSent {
		t.Error("wanted message to be sent")
	}
}

func TestSnagTile(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.SnagGameTile, m.Type; want != got {
					t.Errorf("not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.snagTile()
	if !messageSent {
		t.Error("wanted message to be sent")
	}
}

func TestStartTileSwap(t *testing.T) {
	t.Skip("TODO")
}

func TestSendChat(t *testing.T) {
	tests := []struct {
		form    js.Value
		wantErr bool
		want    message.Message
	}{
		{
			form:    js.ValueOf(map[string]interface{}{}), // bad form
			wantErr: true,
		},
		{
			form: js.ValueOf(map[string]interface{}{ // form
				"method": "get",
				"action": "https://example.com/chat_url",
			}),
			want: message.Message{
				Type: message.GameChat,
				Info: "the_message",
			},
		},
	}
	for i, test := range tests {
		messageSent := false
		errorLogged := false
		g := Game{
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			dom: &mockDOM{
				QuerySelectorAllFunc: func(document js.Value, query string) []js.Value {
					if want, got := test.form, document; !want.Equal(got) {
						t.Errorf("Test %v: forms not equal: wanted: %v, got %v", i, want, got)
					}
					return []js.Value{
						js.ValueOf(map[string]interface{}{
							"name":  "chat",
							"value": "the_message",
						}),
					}
				},
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := test.want, m; !reflect.DeepEqual(want, got) {
						t.Errorf("Test %v: sent messages not equal:\nwanted: %v\ngot:    %v", i, want, got)
					}
					messageSent = true
				},
			},
		}
		event := js.ValueOf(map[string]interface{}{
			"target": test.form,
		})
		g.sendChat(event)
		if want, got := test.wantErr, errorLogged; want != got {
			t.Errorf("Test %v: wanted error (%v), but errorLogged=%v", i, want, got)
		}
		if want, got := !test.wantErr, messageSent; want != got {
			t.Errorf("Test %v: wanted chat message to be sent (%v), but messageSent=%v", i, want, got)
		}
	}
}

func TestReplaceGameTiles(t *testing.T) {
	g := Game{
		board: &board.Board{
			// unused tiles should be cleared because unused tiles are added after used tiles
			UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
			UnusedTileIDs: []tile.ID{1},
			// used  tiles should be clear and reset from the message
			UsedTiles:    map[tile.ID]tile.Position{2: {Tile: tile.Tile{ID: 2}, X: 3, Y: 4}},
			UsedTileLocs: map[tile.X]map[tile.Y]tile.Tile{3: {4: {ID: 2}}},
		},
	}
	m := message.Message{
		Type: message.JoinGame, // avoid logged message in addUnusedTiles
		Game: &game.Info{
			Board: &board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
				UsedTiles:     map[tile.ID]tile.Position{6: {Tile: tile.Tile{ID: 6}, X: 7, Y: 8}},
				UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{7: {8: {ID: 6}}},
			},
		},
	}
	g.replaceGameTiles(m)
	got := g.board
	want := &board.Board{
		UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
		UnusedTileIDs: []tile.ID{5},
		UsedTiles:     map[tile.ID]tile.Position{6: {Tile: tile.Tile{ID: 6}, X: 7, Y: 8}},
		UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{7: {8: {ID: 6}}},
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("boards not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestAddUnusedTiles(t *testing.T) {
	tests := []struct {
		messageBoard         board.Board
		gameBoard            board.Board
		messageType          message.Type
		wantBoard            board.Board
		wantError            bool
		wantInfo             bool
		wantInfoMessagePart1 string
		wantInfoMessagePart2 string
	}{
		{ // could not add all unused tiles
			messageBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{666},
			},
			wantError: true,
		},
		{ // player already has tile with id
			messageBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
			},
			gameBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
			},
			wantBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
			},
			wantError: true,
		},
		{ // join game (no message)
			messageBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
			},
			gameBoard: board.Board{
				UnusedTiles: make(map[tile.ID]tile.Tile),
			},
			messageType: message.JoinGame,
			wantBoard: board.Board{
				UnusedTiles:   map[tile.ID]tile.Tile{5: {ID: 5}},
				UnusedTileIDs: []tile.ID{5},
			},
		},
		{ // want info
			messageBoard: board.Board{
				UnusedTiles: map[tile.ID]tile.Tile{
					5: {ID: 5, Ch: 'V'},
				},
				UnusedTileIDs: []tile.ID{5},
			},
			gameBoard: board.Board{
				UnusedTiles: make(map[tile.ID]tile.Tile),
			},
			wantInfo:             true,
			wantInfoMessagePart1: "adding unused tiles:",
			wantInfoMessagePart2: `"V"`,
			wantBoard: board.Board{
				UnusedTiles: map[tile.ID]tile.Tile{
					5: {ID: 5, Ch: 'V'},
				},
				UnusedTileIDs: []tile.ID{5},
			},
		},
		{ // want info
			messageBoard: board.Board{
				UnusedTiles: map[tile.ID]tile.Tile{
					5: {ID: 5, Ch: 'V'},
					3: {ID: 3, Ch: 'U'},
				},
				UnusedTileIDs: []tile.ID{3, 5},
			},
			gameBoard: board.Board{
				UnusedTiles: make(map[tile.ID]tile.Tile),
			},
			wantInfo:             true,
			wantInfoMessagePart1: "adding unused tile:",
			wantInfoMessagePart2: `"U", "V"`,
			wantBoard: board.Board{
				UnusedTiles: map[tile.ID]tile.Tile{
					3: {ID: 3, Ch: 'U'},
					5: {ID: 5, Ch: 'V'},
				},
				UnusedTileIDs: []tile.ID{3, 5},
			},
		},
	}
	for i, test := range tests {
		errorLogged := false
		infoLogged := false
		g := Game{
			board: &test.gameBoard,
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
				InfoFunc: func(text string) {
					for _, wantPart := range []string{test.wantInfoMessagePart1, test.wantInfoMessagePart2} {
						if want, got := wantPart, text; !strings.Contains(got, want) {
							t.Errorf("Test %v: info message did not contain %q: %v", i, want, got)
						}
					}
					infoLogged = true
				},
			},
		}
		m := message.Message{
			Type: test.messageType,
			Game: &game.Info{
				Board: &test.messageBoard,
			},
		}
		g.addUnusedTiles(m)
		if want, got := &test.wantBoard, g.board; !reflect.DeepEqual(want, got) {
			t.Errorf("Test %v: boards not equal:\nwanted %v\ngot:   %v", i, want, got)
		}
		if want, got := test.wantError, errorLogged; want != got {
			t.Errorf("Test %v: wanted error: %v, got %v", i, want, got)
		}
		if want, got := test.wantInfo, infoLogged; want != got {
			t.Errorf("Test %v: wanted info: %v, got %v", i, want, got)
		}
	}
}

func TestUpdateInfo(t *testing.T) {
	t.Skip("TODO")
}

func TestUpdateStatus(t *testing.T) {
	t.Skip("TODO")
}

func TestUpdateTilesLeft(t *testing.T) {
	tests := []struct {
		m                              message.Message
		wantSetValue                   string
		wantSetButtonDisabledCallCount int
	}{

		{
			m: message.Message{
				Game: &game.Info{
					TilesLeft: 33,
				},
			},
			wantSetValue: "33",
		},
		{
			m: message.Message{
				Game: &game.Info{
					TilesLeft: 0,
					Status:    game.NotStarted,
				},
			},
			wantSetValue:                   "0",
			wantSetButtonDisabledCallCount: 2,
		},
		{
			m: message.Message{
				Game: &game.Info{
					TilesLeft: 0,
					Status:    game.Finished,
				},
			},
			wantSetValue:                   "0",
			wantSetButtonDisabledCallCount: 2,
		},
		{
			m: message.Message{
				Game: &game.Info{
					TilesLeft: 0,
					Status:    game.InProgress,
				},
			},
			wantSetValue:                   "0",
			wantSetButtonDisabledCallCount: 3,
		},
	}
	for i, test := range tests {
		setValueCalled := false
		setButtonDisabledCallCount := 0
		g := Game{
			dom: &mockDOM{
				SetValueFunc: func(query, value string) {
					if want, got := test.wantSetValue, value; want != got {
						t.Errorf("Test %v: wanted setValue to be called with %v, got %v", i, want, got)
					}
					setValueCalled = true
				},
				SetButtonDisabledFunc: func(query string, disabled bool) {
					setButtonDisabledCallCount++
				},
			},
		}
		g.updateTilesLeft(test.m)
		if !setValueCalled {
			t.Errorf("Test %v: wanted SetValue to be called", i)
		}
		if want, got := test.wantSetButtonDisabledCallCount, setButtonDisabledCallCount; want != got {
			t.Errorf("Test %v: wanted setButtonDisabled to be called %v times, got %v", i, want, got)
		}
	}
}

func TestUpdatePlayers(t *testing.T) {
	tests := []struct {
		players      []string
		wantSetValue bool
		want         string
	}{
		{},
		{
			players:      []string{"larry", "curly", "moe"},
			wantSetValue: true,
			want:         "larry,curly,moe", // no spaces to save space
		},
	}
	for i, test := range tests {
		setValueCalled := false
		m := message.Message{
			Game: &game.Info{
				Players: test.players,
			},
		}
		g := Game{
			dom: &mockDOM{
				SetValueFunc: func(query, value string) {
					if want, got := test.want, value; want != got {
						t.Errorf("Test %v: wanted %v, got %v", i, want, got)
					}
					setValueCalled = true
				},
			},
		}
		g.updatePlayers(m)
		if want, got := test.wantSetValue, setValueCalled; want != got {
			t.Errorf("Test %v: wanted set value to be called (%v), got %v", i, want, got)
		}
	}
}

func TestResetTiles(t *testing.T) {
	b := &board.Board{
		UnusedTiles:   map[tile.ID]tile.Tile{1: {ID: 1}},
		UnusedTileIDs: []tile.ID{1},
		UsedTiles:     map[tile.ID]tile.Position{2: {Tile: tile.Tile{ID: 2}, X: 3, Y: 4}},
		UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{3: {4: {ID: 2}}},
	}
	g := Game{
		board: b,
	}
	want := &board.Board{
		UnusedTiles:   map[tile.ID]tile.Tile{},
		UnusedTileIDs: []tile.ID{},
		UsedTiles:     map[tile.ID]tile.Position{},
		UsedTileLocs:  map[tile.X]map[tile.Y]tile.Tile{},
	}
	g.resetTiles()
	if want, got := want, g.board; !reflect.DeepEqual(want, got) {
		t.Errorf("not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestRefreshTileLength(t *testing.T) {
	setValueCalled := false
	g := Game{
		dom: &mockDOM{
			ValueFunc: func(query string) string {
				return "37"
			},
			SetValueFunc: func(query, value string) {
				if want, got := "37", value; want != got {
					t.Errorf("wanted %v, got %v", want, got)
				}
				setValueCalled = true
			},
		},
	}
	g.refreshTileLength()
	if !setValueCalled {
		t.Error("wanted setValue to be called")
	}
}

func TestResizeTiles(t *testing.T) {
	t.Skip("TODO")
}

func TestSetTabActive(t *testing.T) {
	t.Skip("TODO")
}

func TestSetBoardSize(t *testing.T) {
	t.Skip("TODO")
}

func TestSetRules(t *testing.T) {
	rules := []string{"here", "are", "some", "rules"}
	var gotRules []string
	rulesList := js.ValueOf(map[string]interface{}{
		"innerHTML": "should be replaced",
	})
	appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if want, got := rulesList, this; !want.Equal(this) {
			t.Errorf("wanted to append to rules list, got %v", got)
		}
		rule := args[0].Get("innerHTML").String()
		gotRules = append(gotRules, rule)
		return nil
	})
	rulesList.Set("appendChild", appendChild)
	g := Game{
		dom: &mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return rulesList
			},
			CloneElementFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{ // clone
					"children": []interface{}{ // cloneChildren
						map[string]interface{}{}, // li
					},
				})
			},
		},
	}
	g.setRules(rules)
	if want, got := rules, gotRules; !reflect.DeepEqual(want, got) {
		t.Errorf("rulesList rules not equal:\nwanted: %v\ngot:    %v", want, got)
	}
	appendChild.Release()
}

func TestSetFinalBoards(t *testing.T) {
	t.Skip("TODO")
}

func TestNewFinalBoardDiv(t *testing.T) {
	g := Game{
		dom: &mockDOM{
			CloneElementFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{ // clone
					"children": []interface{}{ // cloneChildren
						map[string]interface{}{ // div
							"children": []interface{}{ // divChildren
								map[string]interface{}{}, // radio
								map[string]interface{}{}, // label
							},
						},
					},
				})
			},
		},
	}
	playerName := "playerX"
	got := g.newFinalBoardDiv(playerName)
	if want, got := playerName, got.Get("children").Index(1).Get("innerHTML").String(); want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestViewFinalBoard(t *testing.T) {
	t.Skip("TODO")
}
