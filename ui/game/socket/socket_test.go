//go:build js && wasm

package socket

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui"
	"github.com/jacobpatterson1549/selene-bananas/ui/url"
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
		f := ui.Form{
			URL:    *u,
			Params: make(url.Values, 1),
		}
		s := Socket{
			user: &mockUser{
				JWTFunc: func() string {
					return test.jwt
				},
			},
		}
		got := s.webSocketURL(f)
		if test.want != got {
			t.Errorf("Test %v:\nwanted: %v\ngot:    %v", i, test.want, got)
		}
	}
}

func TestInitDom(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	var wg sync.WaitGroup
	s := Socket{
		dom: &mockDOM{
			AlertOnPanicFunc: func() {
				// NOOP
			},
		},
	}
	s.InitDom(ctx, &wg)
	cancelFunc()
	wg.Wait()
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
	t.Run("not open", func(t *testing.T) {
		errorLogged := false
		s := Socket{
			webSocket: js.Undefined(),
			log: &mockLog{
				ErrorFunc: func(text string) {
					errorLogged = true
				},
			},
		}
		m := message.Message{}
		s.Send(m)
		if !errorLogged {
			t.Error("wanted error to be logged")
		}
	})
	// t.Run("bad message json", func(t *testing.T) { }) // not tested
	t.Run("happy path", func(t *testing.T) {
		sendCalled := false
		sendJsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			want := `{"type":` + strconv.Itoa(int(message.GameChat)) + `,"info":"selene : hi","game":{"id":21}}`
			got := args[0].String()
			if want != got {
				t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
			}
			sendCalled = true
			return nil
		})
		defer sendJsFunc.Release()
		m := message.Message{
			Type: message.GameChat,
			Info: "selene : hi",
		}
		s := Socket{
			webSocket: js.ValueOf(map[string]interface{}{
				"readyState": 1,
				"send":       sendJsFunc,
			}),
			game: &mockGame{
				IDFunc: func() game.ID {
					return 21
				},
			},
		}
		s.Send(m)
		if !sendCalled {
			t.Errorf("send function not called")
		}
	})
}

func TestOnMessage_setGameInfos(t *testing.T) {
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
	user := &mockUser{
		UsernameFunc: func() string {
			return "fred"
		},
	}
	s := Socket{
		lobby: lobby,
		user:  user,
	}
	s.onMessage(event)
	if !gameInfosSet {
		t.Errorf("wanted game infos set")
	}
}
func TestOnMessage_badJSON(t *testing.T) {
	event := js.ValueOf(map[string]interface{}{
		"data": `{bad json}`,
	})
	errorLogged := false
	s := Socket{
		log: &mockLog{
			ErrorFunc: func(text string) {
				errorLogged = true
			},
		},
	}
	s.onMessage(event)
	if !errorLogged {
		t.Error("wanted error to be logged")
	}
}
func TestOnMessage_logging(t *testing.T) {
	tests := []struct {
		messageType message.Type
		want        int
	}{
		{
			messageType: -1, // bad type => error
			want:        1,
		},
		{
			messageType: message.SocketError,
			want:        1,
		},
		{
			messageType: message.SocketWarning,
			want:        2,
		},
		{
			messageType: message.GameChat,
			want:        3,
		},
	}
	for i, test := range tests {
		event := js.ValueOf(map[string]interface{}{
			"data": `{"type":` + strconv.Itoa(int(test.messageType)) + `}`,
		})
		got := 0
		s := Socket{
			log: &mockLog{
				ErrorFunc: func(text string) {
					got = 1
				},
				WarningFunc: func(text string) {
					got = 2
				},
				ChatFunc: func(text string) {
					got = 3
				},
			},
		}
		s.onMessage(event)
		if test.want != got {
			t.Errorf("Test %v: wanted log type %v to be called, got %v", i, test.want, got)
		}
	}
}
func TestOnMessage_handlers(t *testing.T) {
	tests := []struct {
		messageType     message.Type
		wantActionType  int
		wantSocketClose bool
	}{
		{
			messageType:     message.PlayerRemove,
			wantActionType:  1,
			wantSocketClose: true,
		},
		{
			messageType:    message.LeaveGame,
			wantActionType: 1,
		},
		{
			messageType:    message.JoinGame,
			wantActionType: 2,
		},
		{
			messageType:    message.ChangeGameStatus,
			wantActionType: 2,
		},
		{
			messageType:    message.ChangeGameTiles,
			wantActionType: 2,
		},
		{
			messageType:    message.RefreshGameBoard,
			wantActionType: 2,
		},
	}
	for i, test := range tests {
		event := js.ValueOf(map[string]interface{}{
			"data": `{"info":"any","type":` + strconv.Itoa(int(test.messageType)) + `}`,
		})
		infoLogged := false
		socketClosed := false
		gotAction := 0
		closeWebSocket := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			socketClosed = true
			return nil
		})
		s := Socket{
			webSocket: js.ValueOf(map[string]interface{}{
				"readyState": 1,
				"close":      closeWebSocket,
			}),
			log: &mockLog{
				InfoFunc: func(text string) {
					infoLogged = true
				},
			},
			game: &mockGame{
				LeaveFunc: func() {
					gotAction = 1
				},
				UpdateInfoFunc: func(msg message.Message) {
					gotAction = 2
				},
			},
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					if !test.wantSocketClose {
						t.Errorf("Test %v: dom should only be used when removing player", i)
					}
				},
			},
		}
		s.onMessage(event)
		closeWebSocket.Release()
		if !infoLogged {
			t.Error("wanted info to be logged")
		}
		if want, got := test.wantActionType, gotAction; want != got {
			t.Errorf("Test %v: action types not equal: wanted %v, got %v", i, want, got)
		}
		if want, got := test.wantSocketClose, socketClosed; want != got {
			t.Errorf("Test %v: socket closed calls not equal: wanted %v, got %v", i, want, got)
		}
	}
	t.Run("httpPing", func(t *testing.T) {
		event := js.ValueOf(map[string]interface{}{
			"data": `{"type":` + strconv.Itoa(int(message.SocketHTTPPing)) + `}`,
		})
		pingHandled := false
		requestSubmit := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			pingHandled = true
			return nil
		})
		s := Socket{
			dom: &mockDOM{
				QuerySelectorFunc: func(query string) js.Value {
					return js.ValueOf(map[string]interface{}{
						"requestSubmit": requestSubmit,
					})
				},
			},
		}
		s.onMessage(event)
		requestSubmit.Release()
		if !pingHandled {
			t.Error("wanted ping to be handled")
		}
	})
}

func TestNew(t *testing.T) {
	dom := &mockDOM{}
	log := &mockLog{}
	user := &mockUser{}
	game := &mockGame{}
	lobby := &mockLobby{}
	want := &Socket{
		dom:   dom,
		log:   log,
		user:  user,
		game:  game,
		lobby: lobby,
	}
	got := New(dom, log, user, game, lobby)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("sockets not equal\nwatned: %#v\ngot:    %#v", want, got)
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		webSocket js.Value
		event     js.Value
		wantOk    bool
	}{
		{ // already open
			webSocket: js.ValueOf(map[string]interface{}{
				"readyState": 1,
			}),
			wantOk: true,
		},
		{ // bad form
			webSocket: js.ValueOf(map[string]interface{}{
				"readyState": 0,
			}),
			event: js.ValueOf(map[string]interface{}{
				"target": map[string]interface{}{
					"method": "get",
					"action": "bad_url",
				},
			}),
		},
		// happy path testing may be a challenge because Connect call blocks
	}
	for i, test := range tests {
		s := Socket{
			webSocket: test.webSocket,
			dom: &mockDOM{
				QuerySelectorAllFunc: func(document js.Value, query string) (all []js.Value) {
					return
				},
			},
			user: mockUser{
				JWTFunc: func() string {
					return "user.jwt.token"
				},
			},
		}
		err := s.Connect(test.event)
		if want, got := test.wantOk, err == nil; want != got {
			t.Errorf("Test %v: connect errors not equal: wanted %v, got %v (%v)", i, want, got, err)
		}
	}
}

func TestOnOpen(t *testing.T) {
	setCheckedCalled := false
	s := Socket{
		dom: &mockDOM{
			SetCheckedFunc: func(query string, checked bool) {
				setCheckedCalled = true
			},
		},
	}
	errC := make(chan error, 1)
	f := s.onOpen(errC)
	f()
	if got := <-errC; got != nil {
		t.Errorf("wanted nil error sent on error channel when socket is opened, got %v", got)
	}
	if !setCheckedCalled {
		t.Error("wanted dom to be modified when socket opened via SetChecked")
	}
}

func TestOnClose(t *testing.T) {
	tests := []struct {
		event       js.Value
		wantWarning bool
	}{
		{
			event: js.ValueOf(map[string]interface{}{}),
		},
		{
			event: js.ValueOf(map[string]interface{}{"reason": ""}),
		},
		{
			event:       js.ValueOf(map[string]interface{}{"reason": "any"}),
			wantWarning: true,
		},
	}
	for i, test := range tests {
		warningLogged := false
		setCheckedCalled := false
		s := Socket{
			webSocket: js.ValueOf(map[string]interface{}{}),
			log: &mockLog{
				WarningFunc: func(text string) {
					warningLogged = true
				},
			},
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					setCheckedCalled = true
				},
			},
		}
		s.onClose(test.event)
		if want, got := test.wantWarning, warningLogged; want != got {
			t.Errorf("Test %v: wantWarning not equal: wanted %v, got %v", i, want, got)
		}
		if !setCheckedCalled {
			t.Error("wanted something to be checked to indicate that the socket is closed")
		}
	}
}

func TestOnError(t *testing.T) {
	userLoggedOut := false
	s := Socket{
		user: mockUser{
			LogoutFunc: func() {
				userLoggedOut = true
			},
		},
	}
	errC := make(chan error, 1)
	f := s.onError(errC)
	f()
	if got := <-errC; got == nil {
		t.Error("wanted non-nil error to be reported when socket receives an error")
	}
	if !userLoggedOut {
		t.Error("wanted user to be logged out")
	}
}

func TestCloseWebSocket(t *testing.T) {
	funcNames := []string{"onopen", "onclose", "onerror", "onmessage"}
	webSocket := js.ValueOf(map[string]interface{}{
		"onopen":    true,
		"onclose":   true,
		"onerror":   true,
		"onmessage": true,
	})
	s := Socket{
		webSocket: webSocket,
		dom: &mockDOM{
			SetCheckedFunc: func(query string, checked bool) {
				// NOOP
			},
		},
	}
	s.closeWebSocket()
	for _, funcName := range funcNames {
		if want, got := js.Null(), webSocket.Get(funcName); !want.Equal(got) {
			t.Errorf("wanted %v func on webSocket to be null, got %v", funcName, got)
		}
	}
}

// TestClose checks the closing behavior for different isOpen states
func TestClose(t *testing.T) {
	var closeCalled bool
	close := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		closeCalled = true
		return nil
	})
	tests := []struct {
		webSocket          js.Value
		wantCloseWebSocket bool
	}{
		{},
		{
			webSocket: js.ValueOf(map[string]interface{}{
				"readyState": 1,
				"close":      close,
			}),
			wantCloseWebSocket: true,
		},
	}
	for i, test := range tests {
		closeCalled = false
		leaveCalled := false
		s := Socket{
			webSocket: test.webSocket,
			game: &mockGame{
				LeaveFunc: func() {
					leaveCalled = true
				},
			},
			dom: &mockDOM{
				SetCheckedFunc: func(query string, checked bool) {
					// NOOP
				},
			},
		}
		s.Close()
		if !leaveCalled {
			t.Errorf("Test %v: wanted game to be left", i)
		}
		if want, got := test.wantCloseWebSocket, closeCalled; want != got {
			t.Errorf("Test %v: wanted web socket to be closed: %v, got %v", i, want, got)
		}
	}
	close.Release()
}

// TestHandle tests handleGameLeave, handlePlayerRemove and handleInfo
func TestHandle(t *testing.T) {
	socketTests := []struct {
		handle     func(s Socket, m message.Message)
		wantCallID int
	}{
		{
			handle:     func(s Socket, m message.Message) { s.handleGameLeave(m) },
			wantCallID: 1,
		},
		{
			handle:     func(s Socket, m message.Message) { s.handlePlayerRemove(m) },
			wantCallID: 1,
		},
		{
			handle:     func(s Socket, m message.Message) { s.handleInfo(m) },
			wantCallID: 2,
		},
	}
	for _, subTest := range socketTests {
		handleTests := []struct {
			info    string
			wantLog bool
		}{
			{},
			{
				info:    "stuff to log",
				wantLog: true,
			},
		}
		for i, test := range handleTests {
			callID := 0
			infoLogged := false
			m := message.Message{
				Info: test.info,
			}
			s := Socket{
				game: &mockGame{
					LeaveFunc: func() {
						callID = 1
					},
					UpdateInfoFunc: func(msg message.Message) {
						callID = 2
					},
				},
				log: &mockLog{
					InfoFunc: func(text string) {
						if want, got := test.info, text; want != got {
							t.Errorf("Test %v: logged info not equal: wanted %v, got %v", i, want, got)
						}
						infoLogged = true
					},
				},
			}
			subTest.handle(s, m)
			if want, got := subTest.wantCallID, callID; want != got {
				t.Errorf("Test %v: wanted call %v to be made, got %v", i, want, got)
			}
			if want, got := test.wantLog, infoLogged; want != got {
				t.Errorf("Test %v: log.info() call not equal: wanted %v, got %v", i, want, got)
			}
		}
	}
}

func TestHttpPing(t *testing.T) {
	triggered := false
	requestSubmit := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		triggered = true
		return nil
	})
	s := Socket{
		dom: &mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				if want, got := "ping", query; !strings.Contains(got, want) {
					t.Errorf("queried ping form element did not contain %q: %q", want, got)
				}
				return js.ValueOf(map[string]interface{}{
					"requestSubmit": requestSubmit,
				})
			},
		},
	}
	s.httpPing()
	requestSubmit.Release()
	if !triggered {
		t.Error("request submit not triggered")
	}
}
