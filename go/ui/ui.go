// +build js

// Package ui contains js initialization logic.
package ui

import (
	"context"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/socket"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

// InitDom initializes the ui by registering js functions.
// TODO: simplify and move contents of this to go/cmd/ui/main.go.
func InitDom(ctx context.Context, wg *sync.WaitGroup) {
	global := js.Global()
	user.InitDom(ctx, wg) // TODO: make struct
	// canvas
	document := global.Get("document")
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
	l := lobby.Lobby{
		Game:   g,
		Socket: s,
	}
	l.InitDom(ctx, wg)
}
