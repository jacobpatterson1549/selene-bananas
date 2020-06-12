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
	getGameInfosFunc := dom.NewJsFuncEvent(dom.GetGameInfos)
	leaveFunc := dom.NewJsFunc(func() {
		dom.CloseWebsocket()
		dom.LeaveGame()
	})
	dom.RegisterFunc("lobby", "getGameInfos", getGameInfosFunc)
	dom.RegisterFunc("lobby", "leave", leaveFunc)
	go func() {
		<-ctx.Done()
		getGameInfosFunc.Release()
		leaveFunc.Release()
		wg.Done()
	}()
}
