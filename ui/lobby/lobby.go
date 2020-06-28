// +build js,wasm

// Package lobby contains code to view available games and to close the websocket.
package lobby

import (
	"context"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	Lobby struct {
		Game   *controller.Game
		Socket Socket
	}

	// Socket is a structure that connects the server to the lobby.
	Socket interface {
		Connect(event js.Value) error
		Close()
	}
)

// InitDom regesters lobby dom functions
func (l *Lobby) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	connectJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		go l.connect(event)
	})
	leaveJsFunc := dom.NewJsFunc(l.leave)
	dom.RegisterFunc("lobby", "connect", connectJsFunc)
	dom.RegisterFunc("lobby", "leave", leaveJsFunc)
	go func() {
		<-ctx.Done()
		connectJsFunc.Release()
		leaveJsFunc.Release()
		wg.Done()
	}()
}

// connect makes a synchronous request to connect to the lobby.
// It is expected that the server will respond with a game infos message
func (l *Lobby) connect(event js.Value) {
	err := l.Socket.Connect(event)
	if err != nil {
		log.Error(err.Error())
	}
}

// leave closes the socket and leaves the game.
func (l *Lobby) leave() {
	l.Socket.Close()
	l.Game.Leave()
	tbodyElement := dom.QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
}

// setGameInfos updates the game-infos table with the game infos for the username.
func (l *Lobby) SetGameInfos(gameInfos []game.Info, username string) {
	tbodyElement := dom.QuerySelector(".game-infos>tbody")
	tbodyElement.Set("innerHTML", "")
	if len(gameInfos) == 0 {
		emptyGameInfoElement := dom.CloneElement(".no-game-info-row")
		tbodyElement.Call("appendChild", emptyGameInfoElement)
		return
	}
	for _, gameInfo := range gameInfos {
		gameInfoElement := dom.CloneElement(".game-info-row")
		rowElement := gameInfoElement.Get("children").Index(0)
		createdAtTimeText := dom.FormatTime(gameInfo.CreatedAt)
		rowElement.Get("children").Index(0).Set("innerHTML", createdAtTimeText)
		players := strings.Join(gameInfo.Players, ", ")
		rowElement.Get("children").Index(1).Set("innerHTML", players)
		status := gameInfo.Status.String()
		rowElement.Get("children").Index(2).Set("innerHTML", status)
		if gameInfo.CanJoin(username) {
			joinGameButtonElement := dom.CloneElement(".join-game-button")
			joinGameButtonElement.Get("children").Index(0).Set("value", int(gameInfo.ID))
			rowElement.Get("children").Index(2).Call("appendChild", joinGameButtonElement)
		}
		tbodyElement.Call("appendChild", gameInfoElement)
	}
}
