//go:build js && wasm

package game

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

func TestNew(t *testing.T) {
	dom := new(mockDOM)
	log := new(mockLog)
	board := new(board.Board)
	canvas := new(mockCanvas)
	canvasCreator := new(mockCanvasCreator)
	want := &Game{
		dom:           dom,
		log:           log,
		board:         board,
		canvas:        canvas,
		canvasCreator: canvasCreator,
	}
	got := New(dom, log, board, canvas, canvasCreator)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("games not equal:\nwanted: %v\ngot:    %v", want, got)
	}
}

func TestInitDom(t *testing.T) {
	wantJsFuncNames := []string{
		"create",
		"createWithConfig",
		"join",
		"leave",
		"delete",
		"start",
		"finish",
		"snagTile",
		"swapTile",
		"sendChat",
		"resizeTiles",
		"refreshTileLength",
		"viewFinalBoard",
	}
	functionsRegistered := false
	g := Game{
		dom: &mockDOM{
			RegisterFuncsFunc: func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
				if want, got := "game", parentName; want != got {
					t.Errorf("wanted parent name to be %v, got %v", want, got)
				}
				switch len(jsFuncs) {
				case len(wantJsFuncNames):
					for _, jsFuncName := range wantJsFuncNames {
						if _, ok := jsFuncs[jsFuncName]; !ok {
							t.Errorf("wanted jsFunc named %q", jsFuncName)
						}
					}
				default:
					t.Errorf("wanted %v jsFuncs, got %v", len(wantJsFuncNames), len(jsFuncs))
				}
				functionsRegistered = true
			},
			NewJsFuncFunc: func(fn func()) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
			},
			NewJsEventFuncFunc: func(fn func(event js.Value)) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
			},
		},
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	g.InitDom(ctx, &wg)
	if !functionsRegistered {
		t.Error("wanted functions to be registered when dom is initialized")
	}
}

func TestStartCreate(t *testing.T) {
	n := 0
	g := Game{
		dom: &mockDOM{
			SetCheckedFunc: func(query string, checked bool) {
				if want, got := !strings.Contains(query, "hide"), checked; want != got {
					t.Errorf("wanted setChecked(%v) to be called with %v for start-create, got %v", query, want, got)
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
	tests := []struct {
		MinLength string
		wantErr   bool
		numRows   int
		numCols   int
	}{
		{
			MinLength: "NaN",
			wantErr:   true,
		},
		{
			MinLength: "5",
			numRows:   0,
			numCols:   0,
			wantErr:   true,
		},
	}
	for i, test := range tests {
		errorLogged := false
		g := Game{
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{})
				},
				CheckedFunc: func(query string) bool {
					return true
				},
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
				ValueFunc: func(query string) string {
					return test.MinLength
				},
			},
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			canvas: &mockCanvas{
				ParentDivOffsetWidthFunc: func() int {
					return 120
				},
				UpdateSizeFunc: func(width int) {
					if want, got := 120, width; want != got {
						t.Errorf("Test %v: create-with-config canvas widths not equal: wanted %v, got %v", i, want, got)
					}
				},
				NumColsFunc: func() int {
					return test.numCols
				},
				NumRowsFunc: func() int {
					return test.numRows
				},
			},
		}
		var event js.Value
		g.createWithConfig(event)
		if want, got := test.wantErr, errorLogged; want != got {
			t.Errorf("Test %v: error logged not equal: wanted %v, got %v", i, want, got)
		}
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		gameID          string
		wantErr         bool
		wantGameID      game.ID
		wantMessageSent bool
	}{
		{
			gameID:  "NaN",
			wantErr: true,
		},
		{
			gameID:          "7",
			wantGameID:      7,
			wantMessageSent: true,
		},
	}
	for i, test := range tests {
		errorLogged := false
		messageSent := false
		g := Game{
			board: &board.Board{},
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
			},
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			canvas: &mockCanvas{
				ParentDivOffsetWidthFunc: func() int {
					return 120
				},
				UpdateSizeFunc: func(width int) {
					if want, got := 120, width; want != got {
						t.Errorf("Test %v: join canvas widths not equal: wanted %v, got %v", i, want, got)
					}
				},
				NumColsFunc: func() int {
					return 15
				},
				NumRowsFunc: func() int {
					return 15
				},
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := message.JoinGame, m.Type; want != got {
						t.Errorf("Test %v: join message types not equal: wanted %v, got %v", i, want, got)
					}
					messageSent = true
				},
			},
		}
		event := js.ValueOf(map[string]interface{}{
			"srcElement": map[string]interface{}{ // joinGameButton
				"previousElementSibling": map[string]interface{}{ // gameIDInput
					"value": test.gameID,
				},
			},
		})
		g.join(event)
		if want, got := test.wantErr, errorLogged; want != got {
			t.Errorf("Test %v: error logged not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := test.wantMessageSent, messageSent; want != got {
			t.Errorf("Test %v: messagedSent not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := test.wantGameID, g.id; want != got {
			t.Errorf("Test %v: wanted gameID to be set to %v, got %v", i, want, got)
		}
	}
}

func TestHide(t *testing.T) {
	tests := []bool{true, false}
	for i, want := range tests {
		setCheckedCalled := false
		g := Game{
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					if want, got := want, checked; want != got {
						t.Errorf("Test %v: wanted setChecked to be called with %v for hide, got %v", i, want, got)
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
		t.Errorf("id not expected: wanted %v, got %v", want, got)
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
					t.Errorf("wanted setChecked(%v) to be called with %v for send-leave, got %v", query, want, got)
				}
			},
		},
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.LeaveGame, m.Type; want != got {
					t.Errorf("leave message types not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.sendLeave()
	if !messageSent {
		t.Error("wanted leave message to be sent")
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
					t.Errorf("wanted setChecked(%v) to be called with %v for leave, got %v", query, want, got)
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
						t.Errorf("delete message types not equal: wanted %v, got %v", want, got)
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
			t.Errorf("Test %v: wanted delete message to be sent: %v, got %v", i, want, got)
		}
	}
}

func TestStart(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.ChangeGameStatus, m.Type; want != got {
					t.Errorf("change game status message types not equal: wanted %v, got %v", want, got)
				}
				if want, got := game.InProgress, m.Game.Status; want != got {
					t.Errorf("game start statuses not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.Start()
	if !messageSent {
		t.Error("wanted start message to be sent")
	}
}

func TestFinish(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.ChangeGameStatus, m.Type; want != got {
					t.Errorf("change game status message types not equal: wanted %v, got %v", want, got)
				}
				if want, got := game.Finished, m.Game.Status; want != got {
					t.Errorf("game finish statuses not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.finish()
	if !messageSent {
		t.Error("wanted finish message to be sent")
	}
}

func TestSnagTile(t *testing.T) {
	messageSent := false
	g := Game{
		Socket: &mockSocket{
			SendFunc: func(m message.Message) {
				if want, got := message.SnagGameTile, m.Type; want != got {
					t.Errorf("snag-tile message types not equal: wanted %v, got %v", want, got)
				}
				messageSent = true
			},
		},
	}
	g.snagTile()
	if !messageSent {
		t.Error("wanted snag-tile message to be sent")
	}
}

func TestStartTileSwap(t *testing.T) {
	swapStarted := false
	g := Game{
		canvas: &mockCanvas{
			StartSwapFunc: func() {
				swapStarted = true
			},
		},
	}
	g.startTileSwap()
	if !swapStarted {
		t.Error("wanted canvas swap to be started")
	}
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
	tests := []struct {
		m          message.Message
		wantGameID game.ID
	}{
		{
			m: message.Message{
				Game: &game.Info{
					Board: &board.Board{},
				},
			},
			wantGameID: 6,
		},
		{ // replaceGameTiles
			m: message.Message{
				Game: &game.Info{
					Board: &board.Board{
						UsedTiles: map[tile.ID]tile.Position{1: {}},
					},
				},
			},
			wantGameID: 6,
		},
		{ // addUnusedTiles
			m: message.Message{
				Game: &game.Info{
					Board: &board.Board{
						UnusedTiles: map[tile.ID]tile.Tile{2: {}},
					},
				},
			},
			wantGameID: 6,
		},
		{ // join game
			m: message.Message{
				Type: message.JoinGame,
				Game: &game.Info{
					ID:     7,
					Status: game.InProgress,
					Config: &game.Config{},
				},
			},
			wantGameID: 7,
		},
	}
	for i, test := range tests {
		canvasRedrawn := false
		appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		g := Game{
			board: &board.Board{},
			id:    6,
			dom: &mockDOM{
				SetButtonDisabledFunc: func(query string, disabled bool) {
					// NOOP
				},
				SetValueFunc: func(query, value string) {
					// NOOP
				},
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{
						// setFinalBoards' playersList
						"appendChild": appendChild, // rulesList
					})
				},
				CloneElementFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{
						"children": []interface{}{ // game rules
							map[string]interface{}{},
						},
					})
				},
			},
			log: &mockLog{
				InfoFunc: func(text string) {
					// NOOP
				},
			},
			canvas: &mockCanvas{
				RedrawFunc: func() {
					canvasRedrawn = true
				},
				SetGameStatusFunc: func(s game.Status) {
					if want, got := test.m.Game.Status, s; want != got {
						t.Errorf("Test %v: game statuses not equal: wanted %v, got %v", i, want, got)
					}
				},
			},
		}
		g.UpdateInfo(test.m)
		appendChild.Release()
		if !canvasRedrawn {
			t.Errorf("Test %v: wanted canvas to be redrawn", i)
		}
		if want, got := test.wantGameID, g.id; want != got {
			t.Errorf("Test %v: game ids not equal: wanted %v, got %v", i, want, got)
		}
	}
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		s                        game.Status
		gameTilesLeft            int
		wantStatusText           string
		wantSnagButtonDisabled   bool
		wantSwapButtonDisabled   bool
		wantStartButtonDisabled  bool
		wantFinishButtonDisabled bool
	}{
		{
			s: game.Deleted, // do not set status
		},
		{
			s:                        game.NotStarted,
			wantStatusText:           "Not Started",
			wantSnagButtonDisabled:   true,
			wantSwapButtonDisabled:   true,
			wantStartButtonDisabled:  false,
			wantFinishButtonDisabled: true,
		},
		{
			s:                        game.InProgress,
			wantStatusText:           "In Progress",
			wantSnagButtonDisabled:   false,
			wantSwapButtonDisabled:   false,
			wantStartButtonDisabled:  true,
			wantFinishButtonDisabled: false,
		},
		{
			s:                        game.InProgress,
			gameTilesLeft:            1,
			wantStatusText:           "In Progress",
			wantSnagButtonDisabled:   false,
			wantSwapButtonDisabled:   false,
			wantStartButtonDisabled:  true,
			wantFinishButtonDisabled: true,
		},
		{
			s:                        game.Finished,
			wantStatusText:           "Finished",
			wantSnagButtonDisabled:   true,
			wantSwapButtonDisabled:   true,
			wantStartButtonDisabled:  true,
			wantFinishButtonDisabled: true,
		},
	}
	for i, test := range tests {
		wantStatusSet := len(test.wantStatusText) > 0
		statusSet := false
		g := Game{
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{})
				},
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
				SetValueFunc: func(query, value string) {
					if want, got := test.wantStatusText, value; want != got {
						t.Errorf("Test %v: set statuses not equal: wanted %q, got %q", i, want, got)
					}
					statusSet = true
				},
				SetButtonDisabledFunc: func(query string, disabled bool) {
					switch {
					case strings.Contains(query, "snag"):
						if want, got := test.wantSnagButtonDisabled, disabled; want != got {
							t.Errorf("Test %v: snag button not disabled correctly: wanted %v, got %v", i, want, got)
						}
					case strings.Contains(query, "swap"):
						if want, got := test.wantSwapButtonDisabled, disabled; want != got {
							t.Errorf("Test %v: swap button not disabled correctly: wanted %v, got %v", i, want, got)
						}
					case strings.Contains(query, "start"):
						if want, got := test.wantStartButtonDisabled, disabled; want != got {
							t.Errorf("Test %v: start button not disabled correctly: wanted %v, got %v", i, want, got)
						}
					case strings.Contains(query, "finish"):
						if want, got := test.wantFinishButtonDisabled, disabled; want != got {
							t.Errorf("Test %v: finish button not disabled correctly: wanted %v, got %v", i, want, got)
						}
					default:
						t.Errorf("Test %v: unwanted button disabled (%v): %v", i, disabled, query)
					}
				},
			},
			canvas: &mockCanvas{
				SetGameStatusFunc: func(s game.Status) {
					if want, got := test.s, s; want != got {
						t.Errorf("Test %v: canvas status should be same as game: wanted %v, got %v", i, want, got)
					}
				},
			},
		}
		m := message.Message{
			Game: &game.Info{
				Status:    test.s,
				TilesLeft: test.gameTilesLeft,
			},
		}
		g.updateStatus(m)
		if want, got := wantStatusSet, statusSet; want != got {
			t.Errorf("Test %v: statusSet not as desired: wanted %v, got %v", i, want, got)
		}
	}
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
						t.Errorf("Test %v: set query values not equal:  wanted %v, got %v", i, want, got)
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
					t.Errorf("tile length set value not expected: wanted %v, got %v", want, got)
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
	tests := []struct {
		tileLengthStr  string
		wantTileLength int
		wantErr        bool
	}{
		{
			tileLengthStr: "NaN",
			wantErr:       true,
		},
		{
			tileLengthStr:  "48",
			wantTileLength: 48,
		},
	}
	for i, test := range tests {
		errorLogged := false
		tileLengthSet := false
		messageSent := false
		g := Game{
			board: &board.Board{},
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			dom: &mockDOM{
				ValueFunc: func(query string) string {
					return test.tileLengthStr
				},
			},
			canvas: &mockCanvas{
				SetTileLengthFunc: func(tileLength int) {
					if want, got := test.wantTileLength, tileLength; want != got {
						t.Errorf("Test %v: tile lengths not equal: wanted %v, got %v", i, want, got)
					}
					tileLengthSet = true
				},
				NumRowsFunc: func() int { return 15 },
				NumColsFunc: func() int { return 15 },
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := message.RefreshGameBoard, m.Type; want != got {
						t.Errorf("Test %v: refresh board message types not equal: wanted %v, got %v", i, want, got)
					}
					messageSent = true
				},
			},
		}
		g.resizeTiles()
		if want, got := test.wantErr, errorLogged; want != got {
			t.Errorf("Test %v: errorLogged values not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := !test.wantErr, tileLengthSet; want != got {
			t.Errorf("Test %v: tileLengthSet values not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := !test.wantErr, messageSent; want != got {
			t.Errorf("Test %v: messageSent values not equal: wanted %v, got %v", i, want, got)
		}
	}
}

func TestSetTabActive(t *testing.T) {
	tests := []struct {
		canvasLength int
		wantErr      bool
	}{
		{
			canvasLength: -1,
			wantErr:      true,
		},
		{
			canvasLength: 133,
		},
	}
	for i, test := range tests {
		errorLogged := false
		messageSent := false
		g := Game{
			board: &board.Board{},
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{})
				},
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
			},
			canvas: &mockCanvas{
				ParentDivOffsetWidthFunc: func() int {
					return test.canvasLength
				},
				UpdateSizeFunc: func(width int) {
					if want, got := test.canvasLength, width; want != got {
						t.Errorf("Test %v: set-tab-active canvas widths not equal: wanted %v, got %v", i, want, got)
					}
				},
				NumRowsFunc: func() int { return test.canvasLength },
				NumColsFunc: func() int { return test.canvasLength },
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := message.Type(-1), m.Type; want != got {
						t.Errorf("Test %v: tab activation message types not equal: wanted %v, got %v", i, want, got)
					}
					messageSent = true
				},
			},
		}
		m := message.Message{
			Type: -1, // ensures the type is not alered, that it is passed through
		}
		g.setTabActive(m)
		if want, got := test.wantErr, errorLogged; want != got {
			t.Errorf("Test %v: errorLogged values not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := !test.wantErr, messageSent; want != got {
			t.Errorf("Test %v: messageSent values not equal: wanted %v, got %v", i, want, got)
		}
	}
}

func TestSetBoardSize(t *testing.T) {
	tests := []struct {
		m    message.Message
		want message.Message
	}{
		{ // no game or board in message
			want: message.Message{
				Game: &game.Info{
					Board: &board.Board{
						Config: board.Config{
							NumRows: 11,
							NumCols: 12,
						},
					},
				},
			},
		},
		{ // no board in message
			m: message.Message{
				Game: &game.Info{
					ID: 9,
				},
			},
			want: message.Message{
				Game: &game.Info{
					ID: 9,
					Board: &board.Board{
						Config: board.Config{
							NumRows: 11,
							NumCols: 12,
						},
					},
				},
			},
		},
		{ // happy path
			m: message.Message{
				Game: &game.Info{
					ID: 10,
					Board: &board.Board{
						UnusedTileIDs: []tile.ID{7},
					},
				},
			},
			want: message.Message{
				Game: &game.Info{
					ID: 10,
					Board: &board.Board{
						UnusedTileIDs: []tile.ID{7},
						Config: board.Config{
							NumRows: 11,
							NumCols: 12,
						},
					},
				},
			},
		},
	}
	for i, test := range tests {
		messageSent := false
		g := Game{
			board: &board.Board{
				Config:        board.Config{},
				UnusedTileIDs: []tile.ID{6},
			},
			canvas: &mockCanvas{
				NumRowsFunc: func() int {
					return 11
				},
				NumColsFunc: func() int {
					return 12
				},
			},
			Socket: &mockSocket{
				SendFunc: func(m message.Message) {
					if want, got := test.want, m; !reflect.DeepEqual(want, got) {
						t.Errorf("Test %v: sent messages not equal:\nwanted: %#v\ngot:    %#v", i, want, got)
						t.Errorf("\tboards:\n\twanted: %v\n\tgot:    %v", want.Game.Board, got.Game.Board)
					}
					messageSent = true
				},
			},
		}
		g.setBoardSize(test.m)
		if want, got := 11, g.board.Config.NumRows; want != got {
			t.Errorf("Test %v: board rows not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := 12, g.board.Config.NumCols; want != got {
			t.Errorf("Test %v: board rows not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := 0, len(g.board.UnusedTiles); want != got {
			t.Errorf("Test %v: wanted board to be reset with number of unused tiles = %v, got %v", i, want, got)
		}
		if !messageSent {
			t.Error("wanted set-board-size message to be sent")
		}
	}
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
	appendCount := 0
	appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		appendCount++
		return nil
	})
	g := Game{
		dom: &mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{
					"appendChild": appendChild,
				})
			},
			CloneElementFunc: func(query string) js.Value {
				return js.ValueOf(map[string]interface{}{
					"children": []interface{}{
						map[string]interface{}{
							"children": []interface{}{
								map[string]interface{}{}, // radio
								map[string]interface{}{}, // label
							},
						},
					},
				})
			},
			SetCheckedFunc: func(query string, checked bool) {
				// NOOP
			},
		},
	}
	finalBoards := map[string]board.Board{
		"a": {},
		"b": {},
		"c": {},
	}
	g.setFinalBoards(finalBoards)
	appendChild.Release()
	if want, got := len(finalBoards), appendCount; want != got {
		t.Errorf("final board append counts not equal: wanted %v, got %v", want, got)
	}
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
		t.Errorf(" final board inner html not expected: wanted %v, got %v", want, got)
	}
}

func TestViewFinalBoard(t *testing.T) {
	tests := []struct {
		playerName string
		wantErr    bool
	}{
		{
			playerName: "PLAYER_NOT_IN_GAME",
			wantErr:    true,
		},
		{
			playerName: "curly",
		},
	}
	for i, test := range tests {
		tileLengthSet := false
		errorLogged := false
		canvasRedrawn := false
		clearRect := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		strokestyle := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		fillText := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		strokeRect := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		g := Game{
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{
						"innerHTML": test.playerName, // player selector
					})
				},
				ColorFunc: func(element js.Value) string {
					return "black"
				},
			},
			canvas: &mockCanvas{
				TileLengthFunc: func() int {
					return 50
				},
			},
			canvasCreator: mockCanvasCreator{
				CreateFunc: func(board *board.Board, canvasParentDivQuery string) Canvas {
					return &mockCanvas{
						SetTileLengthFunc: func(tileLength int) {
							if tileLength != 50 {
								t.Errorf("Test %v: wanted tile length to be set to 50, got %v", i, tileLength)
							}
							tileLengthSet = true
						},
						DesiredWidthFunc: func() int {
							return 750
						},
						UpdateSizeFunc: func(width int) {
							// NOOP
						},
						RedrawFunc: func() {
							canvasRedrawn = true
						},
					}
				},
			},
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
			finalBoards: map[string]board.Board{
				"larry": {},
				"curly": {
					Config: board.Config{
						NumCols: 15,
					},
				},
				"moe": {},
			},
		}
		g.viewFinalBoard()
		clearRect.Release()
		strokestyle.Release()
		fillText.Release()
		strokeRect.Release()
		switch {
		case test.wantErr:
			if want, got := test.wantErr, errorLogged; want != got {
				t.Errorf("Test %v: errorLogged not equal: wanted %v, got %v", i, want, got)
			}
		case !tileLengthSet:
			t.Errorf("Test %v: tile length not set from game canvas", i)
		case !canvasRedrawn:
			t.Errorf("Test %v: canvas not redrawn", i)
		}
	}
}
