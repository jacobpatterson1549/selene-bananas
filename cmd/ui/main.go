// +build js,wasm

package main

import (
	"context"
	"net/http"
	"sync"
	"syscall/js"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/ui/user"
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
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	u := user.New(&httpClient)
	u.InitDom(ctx, wg)
	// canvas
	canvasDiv := dom.QuerySelector(".game>.canvas")
	canvasElement := dom.QuerySelector(".game>.canvas>canvas")
	var board board.Board
	canvasCfg := canvas.Config{
		TileLength: 20,
	}
	canvas := canvasCfg.New(&board, &canvasDiv, &canvasElement)
	canvas.InitDom(ctx, wg)
	// game
	g := controller.NewGame(&board, canvas)
	g.InitDom(ctx, wg)
	// lobby
	l := lobby.Lobby{
		Game: &g,
	}
	l.InitDom(ctx, wg)
	// websocket
	s := socket.Socket{
		Lobby: &l,
		Game:  &g,
		User:  &u,
	}
	u.Socket = &s
	canvas.Socket = &s
	g.Socket = &s // [circular reference]
	l.Socket = &s // [circular reference]
	s.InitDom(ctx, wg)
	// close handling
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chrome
		cancelFunc()
		fn.Release()
		return nil
	})
	global := js.Global()
	global.Call("addEventListener", "beforeunload", fn)
	// allow interaction
	document := dom.QuerySelector("body")
	disabledSubmitButtons := dom.QuerySelectorAll(document, `input[type="submit"]:disabled`)
	for _, disabledSubmitButton := range disabledSubmitButtons {
		disabledSubmitButton.Set("disabled", false)
	}
}