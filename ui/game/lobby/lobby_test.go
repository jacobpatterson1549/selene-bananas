//go:build js && wasm

package lobby

import (
	"errors"
	"reflect"
	"syscall/js"
	"testing"

	"github.com/jacobpatterson1549/selene-bananas/ui"
)

func TestNew(t *testing.T) {
	dom := new(ui.DOM)
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
	querySelector := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		query := args[0]
		if want, got := ".game-infos>tbody", query.String(); want != got {
			t.Errorf("wanted query to be %v, got %v", want, got)
		}
		return gameInfosTbodyElement
	})
	document := js.ValueOf(map[string]interface{}{
		"querySelector": querySelector,
	})
	js.Global().Set("document", document)
	l := Lobby{
		dom:    new(ui.DOM), // TODO: use mock
		game:   game,
		Socket: socket,
	}
	l.leave()
	querySelector.Release()
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
	t.Skip("TODO")
}
