// +build js

// Package lobby contains code to view available games and to close the websocket.
// TODO: investigate if this is still needed. It is currently just a callthrough to the js package.
package lobby

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	util "github.com/jacobpatterson1549/selene-bananas/go/ui/js"
)

// Init regesters lobby functions
func Init(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	getGameInfosFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		util.GetGameInfos(event)
		return nil
	})
	leaveFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		util.CloseWebsocket()
		util.LeaveGame()
		return nil
	})
	util.RegisterFunc("lobby", "getGameInfos", getGameInfosFunc)
	util.RegisterFunc("lobby", "leave", leaveFunc)
	go func() {
		<-ctx.Done()
		getGameInfosFunc.Release()
		leaveFunc.Release()
		wg.Done()
	}()
}

// SetGameInfos updates the game-infos table with the specified game infos.
func SetGameInfos(gameInfos []game.Info) {
	util.SetGameInfos(gameInfos)
}
