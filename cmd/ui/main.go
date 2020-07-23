// +build js,wasm

// Package main initializes interactive frontend elements and runs as long as the webpage is open.
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
	log := initLog(ctx, wg)
	user := initUser(ctx, wg, log)
	board := new(board.Board)
	canvas := initCanvas(ctx, wg, log, board)
	game := initGame(ctx, wg, log, board, canvas)
	lobby := initLobby(ctx, wg, log, game)
	socket := initSocket(ctx, wg, log, user, game, lobby)
	user.Socket = socket   // [circular reference]
	canvas.Socket = socket // [circular reference]
	game.Socket = socket   // [circular reference]
	lobby.Socket = socket  // [circular reference]
	initBeforeUnloadFn(cancelFunc, wg)
	enableInteraction()
}

func initLog(ctx context.Context, wg *sync.WaitGroup) *log.Log {
	log := new(log.Log)
	log.InitDom(ctx, wg)
	return log
}

func initUser(ctx context.Context, wg *sync.WaitGroup, log *log.Log) *user.User {
	cfg := user.Config{
		Log: log,
	}
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	user := cfg.New(&httpClient)
	user.InitDom(ctx, wg)
	return user
}

func initCanvas(ctx context.Context, wg *sync.WaitGroup, log *log.Log, board *board.Board) *canvas.Canvas {
	cfg := canvas.Config{
		Log:        log,
		TileLength: 20,
	}
	canvasDiv := dom.QuerySelector(".game>.canvas")
	canvasElement := dom.QuerySelector(".game>.canvas>canvas")
	canvas := cfg.New(board, &canvasDiv, &canvasElement)
	canvas.InitDom(ctx, wg)
	return canvas
}

func initGame(ctx context.Context, wg *sync.WaitGroup, log *log.Log, board *board.Board, canvas *canvas.Canvas) *controller.Game {
	cfg := controller.GameConfig{
		Log:    log,
		Board:  board,
		Canvas: canvas,
	}
	game := cfg.NewGame()
	game.InitDom(ctx, wg)
	return game
}

func initLobby(ctx context.Context, wg *sync.WaitGroup, log *log.Log, game *controller.Game) *lobby.Lobby {
	cfg := lobby.Config{
		Log:  log,
		Game: game,
	}
	lobby := cfg.New()
	lobby.InitDom(ctx, wg)
	return lobby
}

func initSocket(ctx context.Context, wg *sync.WaitGroup, log *log.Log, user *user.User, game *controller.Game, lobby *lobby.Lobby) *socket.Socket {
	cfg := socket.Config{
		Log:   log,
		User:  user,
		Game:  game,
		Lobby: lobby,
	}
	socket := cfg.New()
	socket.InitDom(ctx, wg)
	return socket
}

func initBeforeUnloadFn(cancelFunc context.CancelFunc, wg *sync.WaitGroup) {
	wg.Add(1)
	var fn js.Func
	fn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// args[0].Call("preventDefault") // debug in other browsers
		// args[0].Set("returnValue", "") // debug in chromium
		cancelFunc()
		fn.Release()
		wg.Done()
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
