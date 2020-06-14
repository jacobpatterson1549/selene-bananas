// +build js,wasm

// Package socket contains the logic to communicate with the server for the game via websocket communication
package socket

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

type (
	Socket struct {
		webSocket       js.Value
		Game            *controller.Game
		onOpenJsFunc    js.Func
		onCloseJsFunc   js.Func
		onErrorJsFunc   js.Func
		onMessageJsFunc js.Func
	}
)

// InitDom regesters socket dom functions.
func (s *Socket) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		<-ctx.Done()
		s.releaseWebSocketJsFuncs()
		wg.Done()
	}()
}

func (s *Socket) releaseWebSocketJsFuncs() {
	s.onOpenJsFunc.Release()
	s.onCloseJsFunc.Release()
	s.onErrorJsFunc.Release()
	s.onMessageJsFunc.Release()
}

// Connect establishes the websocket connection if it has not yet been established.
func (s *Socket) Connect(event js.Value) error {
	if s.isOpen() {
		return nil
	}
	f := dom.NewForm(event)
	url := f.URL
	if len(url) >= 4 && url[:4] == "http" {
		url = "ws" + url[4:]
	}
	jwt := user.JWT()
	url = url + "?access_token=" + jwt //TODO: write f.encodedParamasUrl() func
	s.releaseWebSocketJsFuncs()
	errC := make(chan error)
	s.onOpenJsFunc = dom.NewJsFunc(s.onOpen(errC))
	s.onCloseJsFunc = dom.NewJsEventFunc(s.onClose)
	s.onErrorJsFunc = dom.NewJsFunc(s.onError(errC))
	s.onMessageJsFunc = dom.NewJsEventFunc(s.onMessage)
	s.webSocket = dom.NewWebSocket(url)
	s.webSocket.Set("onopen", s.onOpenJsFunc)
	s.webSocket.Set("onclose", s.onCloseJsFunc)
	s.webSocket.Set("onerror", s.onErrorJsFunc)
	s.webSocket.Set("onmessage", s.onMessageJsFunc)
	return <-errC
}

// onMessage is called when the websocket opens.
func (s *Socket) onOpen(errC chan<- error) func() {
	return func() {
		dom.SetChecked("has-websocket", true)
		dom.WebSocket = s
		errC <- nil
	}
}

// onMessage is called when the websocket is closing.
func (s *Socket) onClose(event js.Value) {
	s.releaseWebSocketJsFuncs()
	if reason := event.Get("reason"); reason.IsUndefined() {
		log.Error("lobby shut down")
	} else {
		log.Warning("left lobby: " + reason.String())
	}
	dom.SetChecked("has-websocket", false)
	dom.SetChecked("has-game", false)
	dom.SetChecked("tab-4", true) // lobby tab
}

// onMessage is called when the websocket encounters an unexpected error.
func (s *Socket) onError(errC chan<- error) func() {
	return func() {
		user.Logout()
		errC <- errors.New("lobby closed")
	}
}

// onMessage is called when the websocket receives a message.
func (s *Socket) onMessage(event js.Value) {
	jsMessage := event.Get("data")
	messageJSON := jsMessage.String()
	var m game.Message
	err := json.Unmarshal([]byte(messageJSON), &m)
	if err != nil {
		log.Error("unmarshalling message: " + err.Error())
		return
	}
	switch m.Type {
	case game.Leave:
		s.Game.Leave()
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.BoardRefresh:
		s.Game.ReplaceGameTiles(m.Tiles, m.TilePositions, false)
	case game.Infos:
		dom.SetGameInfos(m.GameInfos)
	case game.PlayerDelete:
		s.Close()
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.Join, game.SocketInfo:
		if m.GameStatus != 0 {
			s.Game.SetStatus(m.GameStatus)
		}
		if m.TilesLeft != 0 {
			dom.SetValue("game-tiles-left", strconv.Itoa(m.TilesLeft))
		}
		if len(m.GamePlayers) > 0 {
			players := strings.Join(m.GamePlayers, ",")
			dom.SetValue("game-players", players)
		}
		switch {
		case len(m.TilePositions) > 0:
			silent := m.Type == game.Join
			s.Game.ReplaceGameTiles(m.Tiles, m.TilePositions, silent)
		case len(m.Tiles) > 0:
			silent := m.Type == game.Join
			s.Game.AddUnusedTiles(m.Tiles, silent)
		}
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.SocketError:
		log.Error(m.Info)
	case game.SocketWarning:
		log.Warning(m.Info)
	case game.SocketHTTPPing:
		dom.SocketHTTPPing()
	case game.Chat:
		log.Chat(m.Info)
	default:
		log.Error("unknown message type received")
	}
}

// Send delivers a message to the server via it's websocket, panicing if the WebSocket is not open.
func (s Socket) Send(m game.Message) {
	if !s.isOpen() {
		panic("websocket not open")
	}
	messageJSON, err := json.Marshal(m)
	if err != nil {
		panic("marshalling socket message to send: " + err.Error())
		return
	}
	s.webSocket.Call("send", js.ValueOf(string(messageJSON)))
}

// Close releases the websocket
func (s *Socket) Close() {
	if s.isOpen() {
		s.webSocket.Call("close")
	}
	s.webSocket = js.Null()
	s.Game.Leave()
}

// isOpen determines if the socket is defined and has a readyState of OPEN.
func (s Socket) isOpen() bool {
	return !s.webSocket.IsUndefined() &&
		!s.webSocket.IsNull() &&
		s.webSocket.Get("readyState").Int() == 1
}
