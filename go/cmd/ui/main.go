// +build js,wasm

package main

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

func main() {
	ctx := context.Background()
	var wg sync.WaitGroup
	initDom(ctx, &wg)
	wg.Wait()
}

func initDom(ctx context.Context, wg *sync.WaitGroup) {
	ctx, cancelFunc := context.WithCancel(ctx)
	// log
	log.InitDom(ctx, wg)
	// user
	user.InitDom(ctx, wg)
	// canvas
	document := js.Global().Get("document")
	canvasElement := document.Call("querySelector", "#game>canvas")
	contextElement := canvasElement.Call("getContext", "2d")
	canvasCtx := canvasContext{contextElement}
	var board board.Board
	canvasCfg := canvas.Config{
		Width:      canvasElement.Get("width").Int(),
		Height:     canvasElement.Get("height").Int(),
		TileLength: 20,
		FontName:   "sans-serif",
	}
	canvas := canvasCfg.New(&canvasCtx, &board)
	canvas.InitDom(ctx, wg, canvasElement)
	// game
	g := controller.NewGame(&board, &canvas)
	g.InitDom(ctx, wg)
	// websocket
	s := socket.Socket{
		Game: &g,
	}
	s.InitDom(ctx, wg)
	// lobby
	l := lobby.Lobby{
		Game:   g,
		Socket: s,
	}
	l.InitDom(ctx, wg)
	// close handling
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chrome
		cancelFunc()
		fn.Release()
		return nil
	})
	js.Global().Call("addEventListener", "beforeunload", fn)
}
