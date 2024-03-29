//go:build js && wasm

// Package lobby contains code to view available games and to close the websocket.
package lobby

import (
	"context"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
)

type (
	// Lobby handles viewing, joining, and creating games on the server.
	Lobby struct {
		dom    DOM
		log    Log
		game   Game
		Socket Socket
	}

	// Log is used to store text about connection errors.
	Log interface {
		Error(text string)
	}

	// Game is held so the lobby can notify of it when the user leaves.
	Game interface {
		Leave()
	}

	// Socket is a structure that connects the server to the lobby.
	Socket interface {
		Connect(event js.Value) error
		Close()
	}

	// DOM interacts with the page.
	DOM interface {
		QuerySelector(query string) js.Value
		FormatTime(utcSeconds int64) string
		CloneElement(query string) js.Value
		RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
		NewJsFunc(fn func()) js.Func
		NewJsEventFuncAsync(fn func(event js.Value), async bool) js.Func
	}
)

// New creates a lobby for games.
func New(dom DOM, log Log, game Game) *Lobby {
	l := Lobby{
		dom:  dom,
		log:  log,
		game: game,
	}
	return &l
}

// InitDom registers lobby dom functions.
func (l *Lobby) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"connect": l.dom.NewJsEventFuncAsync(l.connect, true),
		"leave":   l.dom.NewJsFunc(l.leave),
	}
	l.dom.RegisterFuncs(ctx, wg, "lobby", jsFuncs)
}

// connect makes a BLOCKING request to connect to the lobby.
// It is expected that the server will respond with a game infos message.
func (l *Lobby) connect(event js.Value) {
	err := l.Socket.Connect(event)
	if err != nil {
		l.log.Error(err.Error())
	}
}

// leave closes the socket and leaves the game.
func (l *Lobby) leave() {
	l.Socket.Close()
	l.game.Leave()
	tbodyElement := l.dom.QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
}

// SetGameInfos updates the game-infos table with the game infos for the username.
func (l *Lobby) SetGameInfos(gameInfos []game.Info, username string) {
	tbodyElement := l.dom.QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
	if len(gameInfos) == 0 {
		emptyGameInfoElement := l.dom.CloneElement(".no-game-info-row")
		tbodyElement.Call("appendChild", emptyGameInfoElement)
		return
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := l.gameInfoElement(gameInfo, username)
		tbodyElement.Call("appendChild", gameInfoElement)
	}
}

func (l *Lobby) gameInfoElement(gameInfo game.Info, username string) js.Value {
	gameInfoElement := l.dom.CloneElement(".game-info-row")

	rowElement := gameInfoElement.Get("children").Index(0)

	createdAtTimeText := l.dom.FormatTime(gameInfo.CreatedAt)
	rowElement.Get("children").Index(0).Set("innerHTML", createdAtTimeText)

	players := strings.Join(gameInfo.Players, ", ")
	rowElement.Get("children").Index(1).Set("innerHTML", players)

	capacityRatio := gameInfo.CapacityRatio()
	rowElement.Get("children").Index(2).Set("innerHTML", capacityRatio)

	status := gameInfo.Status.String()
	rowElement.Get("children").Index(3).Set("innerHTML", status)

	joinElements := rowElement.Get("children").Index(4)
	joinElements.Get("children").Index(0).Set("value", int(gameInfo.ID)) // must cast ID to int for js.Value conversion
	if !gameInfo.CanJoin(username) {
		joinElements.Get("children").Index(1).Set("disabled", true)
	}

	return gameInfoElement
}
