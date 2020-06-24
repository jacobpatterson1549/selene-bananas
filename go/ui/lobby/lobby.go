// +build js,wasm

// Package lobby contains code to view available games and to close the websocket.
package lobby

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
)

type (
	Lobby struct {
		Game   *controller.Game
		Socket *socket.Socket
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
}
