//go:build js && wasm

// Package socket contains the logic to communicate with the server for the game via websocket communication
package socket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/ui"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// Socket can be used to easily push and pull messages from the server.
	Socket struct {
		dom       *ui.DOM
		log       *log.Log
		webSocket js.Value
		user      User
		game      Game
		lobby     Lobby
		jsFuncs   struct {
			onOpen    js.Func
			onClose   js.Func
			onError   js.Func
			onMessage js.Func
		}
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

	// Game is the game the user is currently playing.
	Game interface {
		// ID is the id for the game.
		ID() game.ID
		// Leave removes the user from his current game.
		Leave()
		// UpdateInfo updates the game for the specified message.
		UpdateInfo(m message.Message)
	}

	// Lobby is used to display available games and give users a place to join a game from.
	Lobby interface {
		SetGameInfos(gameInfos []game.Info, username string)
	}
)

// New creates a new socket.
func New(dom *ui.DOM, log *log.Log, user User, game Game, lobby Lobby) *Socket {
	s := Socket{
		dom:   dom,
		log:   log,
		user:  user,
		game:  game,
		lobby: lobby,
	}
	return &s
}

// InitDom registers socket dom functions.
func (s *Socket) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go s.releaseJsFuncsOnDone(ctx, wg)
}

// releaseJsFuncsOnDone waits for the context to be done before releasing the event listener functions.
func (s *Socket) releaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup) {
	defer s.dom.AlertOnPanic()
	<-ctx.Done() // BLOCKING
	s.releaseWebSocketJsFuncs()
	wg.Done()
}

// releaseWebSocketJsFuncs releases the event listener functions.
func (s *Socket) releaseWebSocketJsFuncs() {
	s.jsFuncs.onOpen.Release()
	s.jsFuncs.onClose.Release()
	s.jsFuncs.onError.Release()
	s.jsFuncs.onMessage.Release()
}

// Connect establishes the websocket connection if it has not yet been established.
func (s *Socket) Connect(event js.Value) error {
	if s.isOpen() {
		return nil
	}
	f, err := ui.NewForm(s.dom, event)
	if err != nil {
		return err
	}
	url := s.webSocketURL(*f)
	s.releaseWebSocketJsFuncs()
	errC := make(chan error, 1)
	s.jsFuncs.onOpen = s.dom.NewJsFunc(s.onOpen(errC))
	s.jsFuncs.onClose = s.dom.NewJsEventFunc(s.onClose)
	s.jsFuncs.onError = s.dom.NewJsFunc(s.onError(errC))
	s.jsFuncs.onMessage = s.dom.NewJsEventFunc(s.onMessage)
	s.webSocket = s.dom.NewWebSocket(url)
	s.webSocket.Set("onopen", s.jsFuncs.onOpen)
	s.webSocket.Set("onclose", s.jsFuncs.onClose)
	s.webSocket.Set("onerror", s.jsFuncs.onError)
	s.webSocket.Set("onmessage", s.jsFuncs.onMessage)
	return <-errC
}

// getWebSocketURL creates a websocket url from the form, changing it's scheme and adding an access_token.
func (s *Socket) webSocketURL(f ui.Form) string {
	switch f.URL.Scheme {
	case "http":
		f.URL.Scheme = "ws"
	default:
		f.URL.Scheme = "wss"
	}
	jwt := s.user.JWT()
	f.Params.Add("access_token", jwt)
	f.URL.RawQuery = f.Params.Encode()
	return f.URL.String()
}

// onMessage is called when the websocket opens.
func (s *Socket) onOpen(errC chan<- error) func() {
	return func() {
		s.dom.SetChecked("#has-websocket", true)
		errC <- nil
	}
}

// onMessage is called when the websocket is closing.
func (s *Socket) onClose(event js.Value) {
	if reason := event.Get("reason"); !reason.IsUndefined() && len(reason.String()) != 0 {
		s.log.Warning("left lobby: " + reason.String())
	}
	s.closeWebSocket()
}

// closeWebSocket releases the event listeners and does some dom cleanup.
func (s *Socket) closeWebSocket() {
	s.webSocket.Set("onopen", nil)
	s.webSocket.Set("onclose", nil)
	s.webSocket.Set("onerror", nil)
	s.webSocket.Set("onmessage", nil)
	s.releaseWebSocketJsFuncs()
	s.dom.SetChecked("#has-websocket", false)
	s.dom.SetChecked("#hide-game", true)
	s.dom.SetChecked("#tab-lobby", true)
}

// onMessage is called when the websocket encounters an unwanted error.
func (s *Socket) onError(errC chan<- error) func() {
	return func() {
		s.user.Logout()
		errC <- errors.New("lobby closed")
	}
}

// onMessage is called when the websocket receives a message.
func (s *Socket) onMessage(event js.Value) {
	jsMessage := event.Get("data")
	messageJSON := jsMessage.String()
	var m message.Message
	err := json.Unmarshal([]byte(messageJSON), &m)
	if err != nil {
		s.log.Error("unmarshalling message: " + err.Error())
		return
	}
	switch m.Type {
	case message.LeaveGame:
		s.handleGameLeave(m)
	case message.GameInfos:
		s.lobby.SetGameInfos(m.Games, s.user.Username())
	case message.PlayerRemove:
		s.handlePlayerRemove(m)
	case message.JoinGame, message.ChangeGameStatus, message.ChangeGameTiles, message.RefreshGameBoard:
		s.handleInfo(m)
	case message.SocketError:
		s.log.Error(m.Info)
	case message.SocketWarning:
		s.log.Warning(m.Info)
	case message.SocketHTTPPing:
		s.httpPing()
	case message.GameChat:
		s.log.Chat(m.Info)
	default:
		s.log.Error("unknown message type received")
	}
}

// Send delivers a message to the server via it's websocket.
func (s *Socket) Send(m message.Message) {
	if !s.isOpen() {
		s.log.Error("websocket not open")
		return
	}
	if m.Game == nil {
		var g game.Info
		m.Game = &g
	}
	if m.Type != message.CreateGame { // all messages except CREATE are for a specific game
		m.Game.ID = s.game.ID()
	}
	messageJSON, err := json.Marshal(m)
	if err != nil {
		s.log.Error("marshalling socket message to send: " + err.Error())
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
	s.game.Leave()
}

// isOpen determines if the socket is defined and has a readyState of OPEN.
func (s *Socket) isOpen() bool {
	return !s.webSocket.IsUndefined() &&
		s.webSocket.Get("readyState").Int() == 1
}

// handleGameLeave leaves the game and logs any info text from the message.
func (s *Socket) handleGameLeave(m message.Message) {
	s.game.Leave()
	if len(m.Info) > 0 {
		s.log.Info(m.Info)
	}
}

// handlePlayerRemove closes the socket and logs any info text from the message.
func (s *Socket) handlePlayerRemove(m message.Message) {
	s.Close()
	if len(m.Info) > 0 {
		s.log.Info(m.Info)
	}
}

// handleInfo contains the logic for handling messages with types Info and GameJoin.
func (s *Socket) handleInfo(m message.Message) {
	s.game.UpdateInfo(m)
	if len(m.Info) > 0 {
		s.log.Info(m.Info)
	}
}

// httpPing submits the small ping form to keep the server's http handling active.
func (s *Socket) httpPing() {
	pingFormElement := s.dom.QuerySelector("form.ping")
	pingFormElement.Call("requestSubmit")
}
