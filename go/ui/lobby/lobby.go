// +build js,wasm

// Package lobby contains code to view available games and to close the websocket.
package lobby

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
)

type (
	Lobby struct {
		Game   controller.Game
		Socket socket.Socket
	}
)

// InitDom regesters lobby dom functions
func (l *Lobby) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	getGameInfosJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		go l.getGameInfos(event)
	})
	leaveJsFunc := dom.NewJsFunc(l.leave)
	dom.RegisterFunc("lobby", "getGameInfos", getGameInfosJsFunc)
	dom.RegisterFunc("lobby", "leave", leaveJsFunc)
	go func() {
		<-ctx.Done()
		getGameInfosJsFunc.Release()
		leaveJsFunc.Release()
		wg.Done()
	}()
}

// get game infos makes a asynchronous request to get current game infos, establishing a socket connection if necessary
func (l *Lobby) getGameInfos(event js.Value) {
	err := l.Socket.Connect(event)
	switch {
	case err != nil:
		log.Error(err.Error())
	default:
		dom.Send(game.Message{
			Type: game.Infos,
		})
	}
}

// leave closes the socket and leaves the game.
func (l *Lobby) leave() {
	l.Socket.Close()
	l.Game.Leave()
}
