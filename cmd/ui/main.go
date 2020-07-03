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
	initLog(ctx, wg)
	user := initUser(ctx, wg)
	board := new(board.Board)
	canvas := initCanvas(ctx, wg, board)
	game := initGame(ctx, wg, board, canvas)
	lobby := initLobby(ctx, wg, game)
	_ = initSocket(ctx, wg, user, canvas, game, lobby)
	initBeforeUnloadFn(cancelFunc)
	enableInteraction()
}

func initLog(ctx context.Context, wg *sync.WaitGroup) {
	log.InitDom(ctx, wg)
}

func initUser(ctx context.Context, wg *sync.WaitGroup) *user.User {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	user := user.New(&httpClient)
	user.InitDom(ctx, wg)
	return user
}

func initCanvas(ctx context.Context, wg *sync.WaitGroup, board *board.Board) *canvas.Canvas {
	canvasDiv := dom.QuerySelector(".game>.canvas")
	canvasElement := dom.QuerySelector(".game>.canvas>canvas")
	canvasCfg := canvas.Config{
		TileLength: 20,
	}
	canvas := canvasCfg.New(board, &canvasDiv, &canvasElement)
	canvas.InitDom(ctx, wg)
	return canvas
}

func initGame(ctx context.Context, wg *sync.WaitGroup, board *board.Board, canvas *canvas.Canvas) *controller.Game {
	game := controller.NewGame(board, canvas)
	game.InitDom(ctx, wg)
	return &game
}

func initLobby(ctx context.Context, wg *sync.WaitGroup, game *controller.Game) *lobby.Lobby {
	lobby := lobby.Lobby{
		Game: game,
	}
	lobby.InitDom(ctx, wg)
	return &lobby
}

func initSocket(ctx context.Context, wg *sync.WaitGroup, user *user.User, canvas *canvas.Canvas, game *controller.Game, lobby *lobby.Lobby) *socket.Socket {
	socket := socket.Socket{
		User:  user,
		Game:  game,
		Lobby: lobby,
	}
	user.Socket = &socket
	canvas.Socket = &socket
	game.Socket = &socket  // [circular reference]
	lobby.Socket = &socket // [circular reference]
	socket.InitDom(ctx, wg)
	return &socket
}

func initBeforeUnloadFn(cancelFunc context.CancelFunc) {
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chromium
		cancelFunc()
		fn.Release()
		return nil
	})
	global := js.Global()
	global.Call("addEventListener", "beforeunload", fn)
}

func enableInteraction() {
	document := dom.QuerySelector("body")
	disabledSubmitButtons := dom.QuerySelectorAll(document, `input[type="submit"]:disabled`)
	for _, disabledSubmitButton := range disabledSubmitButtons {
		disabledSubmitButton.Set("disabled", false)
	}
}
