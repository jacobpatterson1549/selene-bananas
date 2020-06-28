// +build js,wasm

// Package socket contains the logic to communicate with the server for the game via websocket communication
package socket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// Socket can be used to easily push and pull messages from the server.
	Socket struct {
		webSocket       js.Value
		Lobby           *lobby.Lobby
		Game            *controller.Game
		User            User
		onOpenJsFunc    js.Func
		onCloseJsFunc   js.Func
		onErrorJsFunc   js.Func
		onMessageJsFunc js.Func
	}

	// User is the state of the current user.
	User interface {
		// JWT gets the user's Java Web Token.
		JWT() string
		// Username gets the user's username.
		Username() string
		// Logout releases the use credentials from the browser.
		Logout()
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
	f, err := dom.NewForm(event)
	if err != nil {
		return err
	}
	url := s.getWebSocketURL(*f)
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

// getWebSocketURL creates a websocket url from the form, changing it's scheme and adding an access_token.
func (s *Socket) getWebSocketURL(f dom.Form) string {
	switch f.URL.Scheme {
	case "http":
		f.URL.Scheme = "ws"
	default:
		f.URL.Scheme = "wss"
	}
	jwt := s.User.JWT()
	f.Params.Add("access_token", jwt)
	f.URL.RawQuery = f.Params.Encode()
	return f.URL.String()
}

// onMessage is called when the websocket opens.
func (s *Socket) onOpen(errC chan<- error) func() {
	return func() {
		dom.SetCheckedQuery(".has-websocket", true)
		errC <- nil
	}
}

// onMessage is called when the websocket is closing.
func (s *Socket) onClose(event js.Value) {
	if reason := event.Get("reason"); !reason.IsUndefined() && len(reason.String()) != 0 {
		log.Warning("left lobby: " + reason.String())
	}
	s.closeWebSocket()
}

func (s *Socket) closeWebSocket() {
	s.webSocket.Set("onopen", nil)
	s.webSocket.Set("onclose", nil)
	s.webSocket.Set("onerror", nil)
	s.webSocket.Set("onmessage", nil)
	s.releaseWebSocketJsFuncs()
	dom.SetCheckedQuery(".has-websocket", false)
	dom.SetCheckedQuery(".has-game", false)
	dom.SetCheckedQuery("#tab-lobby", true)
}

// onMessage is called when the websocket encounters an unexpected error.
func (s *Socket) onError(errC chan<- error) func() {
	return func() {
		s.User.Logout()
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
		s.handleGameLeave(m)
	case game.Infos:
		s.Lobby.SetGameInfos(m.GameInfos, s.User.Username())
	case game.PlayerDelete:
		s.handlePlayerDelete(m)
	case game.Join, game.StatusChange, game.TilesChange:
		s.handleInfo(m)
	case game.SocketError:
		log.Error(m.Info)
	case game.SocketWarning:
		log.Warning(m.Info)
	case game.SocketHTTPPing:
		s.httpPing()
	case game.Chat:
		log.Chat(m.Info)
	default:
		log.Error("unknown message type received")
	}
}

// Send delivers a message to the server via it's websocket, panicing if the WebSocket is not open.
func (s *Socket) Send(m game.Message) {
	if !s.isOpen() {
		log.Error("websocket not open")
		return
	}
	messageJSON, err := json.Marshal(m)
	if err != nil {
		log.Error("marshalling socket message to send: " + err.Error())
		return
	}
	s.webSocket.Call("send", js.ValueOf(string(messageJSON)))
}

// Close releases the websocket
func (s *Socket) Close() {
	if s.isOpen() {
		s.closeWebSocket() // removes onClose
		s.webSocket.Call("close")
	}
	s.Game.Leave()
}

// isOpen determines if the socket is defined and has a readyState of OPEN.
func (s *Socket) isOpen() bool {
	return !s.webSocket.IsUndefined() &&
		s.webSocket.Get("readyState").Int() == 1
}

// handleGameLeave leaves the game and logs any info text from the message.
func (s *Socket) handleGameLeave(m game.Message) {
	s.Game.Leave()
	if len(m.Info) > 0 {
		log.Info(m.Info)
	}
}

// handlePlayerDelete closes the socket and logs any info text from the message.
func (s *Socket) handlePlayerDelete(m game.Message) {
	s.Close()
	if len(m.Info) > 0 {
		log.Info(m.Info)
	}
}

// handleInfo contains the logic for handling messages with types Info and GameJoin.
func (s *Socket) handleInfo(m game.Message) {
	s.Game.UpdateInfo(m)
	if len(m.Info) > 0 {
		log.Info(m.Info)
	}
}

// httpPing submits the small ping form to keep the server's http handling active.
func (Socket) httpPing() {
	pingFormElement := dom.QuerySelector("form.ping")
	pingFormElement.Call("requestSubmit")
}
