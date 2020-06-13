// +build js,wasm

// Package lobby contains code to view available games and to close the websocket.
// TODO: investigate if this is still needed. It is currently just a callthrough to the js package.
package lobby

import (
	"context"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
)

// InitDom regesters lobby dom functions
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	getGameInfosJsFunc := dom.NewJsEventFunc(dom.GetGameInfos)
	leaveJsFunc := dom.NewJsFunc(func() {
		dom.CloseWebsocket()
		dom.LeaveGame()
	})
	dom.RegisterFunc("lobby", "getGameInfos", getGameInfosJsFunc)
	dom.RegisterFunc("lobby", "leave", leaveJsFunc)
	go func() {
		<-ctx.Done()
		getGameInfosJsFunc.Release()
		leaveJsFunc.Release()
		wg.Done()
	}()
}
