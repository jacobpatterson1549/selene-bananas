//go:build js && wasm

package lobby

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/game"
)

func TestNew(t *testing.T) {
	dom := new(mockDOM)
	log := new(mockLog)
	game := new(mockGame)
	l := New(dom, log, game)
	switch {
	case !reflect.DeepEqual(dom, l.dom):
		t.Errorf("doms not equal: wanted %v, got %v", dom, l.dom)
	case !reflect.DeepEqual(log, l.log):
		t.Errorf("logs not equal: wanted %v, got %v", log, l.log)
	case !reflect.DeepEqual(game, l.game):
		t.Errorf("games not equal: wanted %v, got %v", game, l.game)
	case l.Socket != nil:
		t.Errorf("wanted nil lobby socket when new is called, got %v", l.Socket)
	}
}

func TestInitDom(t *testing.T) {
	wantJsFuncNames := []string{
		"connect",
		"leave",
	}
	functionsRegistered := false
	u := Lobby{
		dom: &mockDOM{
			RegisterFuncsFunc: func(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func) {
				if want, got := "lobby", parentName; want != got {
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
			NewJsEventFuncAsyncFunc: func(fn func(event js.Value), async bool) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
			},
		},
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	u.InitDom(ctx, &wg)
	if !functionsRegistered {
		t.Error("wanted functions to be registered when dom is initialized")
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		event      js.Value
		connectErr error
		wantLog    bool
	}{
		{
			event: js.ValueOf(1),
		},
		{
			event:      js.ValueOf(2),
			connectErr: errors.New("connect error"),
			wantLog:    true,
		},
	}
	for i, test := range tests {
		socket := mockSocket{
			connectFunc: func(event js.Value) error {
				if !test.event.Equal(event) {
					t.Errorf("Test %v: wanted connect called with %v, got %v", i, test.event, event)
				}
				return test.connectErr
			},
		}
		errorLogged := false
		log := mockLog{
			errorFunc: func(text string) {
				errorLogged = true
			},
		}
		l := Lobby{
			log:    log,
			Socket: socket,
		}
		l.connect(test.event)
		switch {
		case test.wantLog != errorLogged:
			t.Errorf("Test %v: too much or not enough error logging", i)
		}
	}
}

func TestLeave(t *testing.T) {
	socketClosed := false
	gameLeft := false
	gameInfosTbodyElement := js.ValueOf(map[string]interface{}{
		"innerHTML": "existing game infos",
	})
	socket := mockSocket{
		closeFunc: func() {
			socketClosed = true
		},
	}
	game := mockGame{
		leaveFunc: func() {
			gameLeft = true
		},
	}
	dom := mockDOM{
		QuerySelectorFunc: func(query string) js.Value {
			return gameInfosTbodyElement
		},
	}
	l := Lobby{
		dom:    &dom,
		game:   game,
		Socket: socket,
	}
	l.leave()
	switch {
	case !socketClosed:
		t.Error("close not called on socket")
	case !gameLeft:
		t.Error("leave not called on game")
	case len(gameInfosTbodyElement.Get("innerHTML").String()) != 0:
		t.Error("gameInfos table not cleared")
	}
}

func TestSetGameInfos(t *testing.T) {
	t.Run("noGameInfo", func(t *testing.T) {
		emptyGameInfoElement := js.ValueOf(1337)
		gameInfoElementAppended := false
		appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			element := args[0]
			gameInfoElementAppended = true
			if want, got := emptyGameInfoElement, element; !want.Equal(got) {
				t.Errorf("wanted %v to be appended, got %v", want, got)
			}
			return nil
		})
		gameInfosTbodyElement := js.ValueOf(map[string]interface{}{
			"innerHTML":   "existing game infos",
			"appendChild": appendChild,
		})
		dom := mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return gameInfosTbodyElement
			},
			CloneElementFunc: func(query string) js.Value {
				return emptyGameInfoElement
			},
		}
		gameInfos := make([]game.Info, 0)
		l := Lobby{
			dom: &dom,
		}
		l.SetGameInfos(gameInfos, "any-username")
		if got := gameInfosTbodyElement.Get("innerHTML").String(); len(got) != 0 {
			t.Error("wanted game infos table to be cleared")
		}
		if !gameInfoElementAppended {
			t.Errorf("wanted gameInfoElement to be appended")
		}
		appendChild.Release()
	})
	t.Run("happy path", func(t *testing.T) {
		numAppended := 0
		newGameInfoRow := func(createdAt, players, capacityRatio, status string, id int, canJoin bool) js.Value {
			return js.ValueOf(map[string]interface{}{ // gameInfoElement
				"children": []interface{}{
					map[string]interface{}{ //rowElement
						"children": []interface{}{
							map[string]interface{}{"innerHTML": createdAt},
							map[string]interface{}{"innerHTML": players},
							map[string]interface{}{"innerHTML": capacityRatio},
							map[string]interface{}{"innerHTML": status},
							map[string]interface{}{ // join column
								"children": []interface{}{
									map[string]interface{}{"value": id},
									map[string]interface{}{"disabled": !canJoin},
								},
							},
						},
					},
				},
			})
		}
		wantAppends := []js.Value{
			newGameInfoRow("A", "use, server, sort, order", "4/6", "In Progress", 3, false),
			newGameInfoRow("B", "me", "1/1", "Finished", 2, true),
		}
		jsonString := func(v js.Value) string { // hack to get around js.Value.Equal using === (refs are different)
			return js.Global().Get("JSON").Call("stringify", v).String()
		}
		appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			element := args[0]
			if want, got := jsonString(wantAppends[numAppended]), jsonString(element); want != got {
				t.Errorf("append %v not equal:\nwanted: %v\ngot:    %v", numAppended+1, want, got)
			}
			numAppended++
			return nil
		})
		gameInfosTbodyElement := js.ValueOf(map[string]interface{}{
			"innerHTML":   "existing game infos",
			"appendChild": appendChild,
		})
		dom := mockDOM{
			QuerySelectorFunc: func(query string) js.Value {
				return gameInfosTbodyElement
			},
			CloneElementFunc: func(query string) js.Value {
				return newGameInfoRow("", "", "", "", 0, true)
			},
			FormatTimeFunc: func(utcSeconds int64) string {
				return string(rune(utcSeconds))
			},
		}
		gameInfos := []game.Info{
			{
				ID:        3,
				CreatedAt: 65,
				Players:   []string{"use", "server", "sort", "order"},
				Capacity:  6,
				Status:    game.InProgress,
			},
			{
				ID:        2,
				CreatedAt: 66,
				Players:   []string{"me"},
				Capacity:  1,
				Status:    game.Finished,
			},
		}
		l := Lobby{
			dom: &dom,
		}
		l.SetGameInfos(gameInfos, "me")
		if got := gameInfosTbodyElement.Get("innerHTML").String(); len(got) != 0 {
			t.Error("wanted game infos table to be cleared")
		}
		if numAppended != 2 {
			t.Errorf("wanted 2 gameInfoElements to be appended, got %v", numAppended)
		}
		appendChild.Release()
	})
}
