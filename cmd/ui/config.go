//go:build js && wasm

// Package main initializes interactive frontend elements and runs as long as the webpage is open.
package main

import (
	"context"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/ui"
	"github.com/jacobpatterson1549/selene-bananas/ui/game"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/canvas"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/lobby"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/ui/http"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/ui/user"
)

type (
	// flags contains options for the the ui.
	flags struct {
		dom         *ui.DOM
		httpTimeout time.Duration
		tileLength  int
	}

	// domInitializer adds functions to the the dom
	domInitializer interface {
		InitDom(ctx context.Context, wg *sync.WaitGroup)
	}
)

// initDom creates, initializes, and links up dom components.
func (f *flags) initDom(ctx context.Context, wg *sync.WaitGroup) {
	domInitializers := f.createDomInitializers()
	for _, di := range domInitializers {
		di.InitDom(ctx, wg)
	}
}

// createDomInitializers creates the components that need to be initialized.
func (f *flags) createDomInitializers() []domInitializer {
	timeFunc := func() int64 {
		return time.Now().Unix()
	}
	log := log.New(f.dom, timeFunc)
	httpClient := http.Client{
		Timeout: f.httpTimeout,
	}
	canvasCfg := canvas.Config{
		TileLength: f.tileLength,
	}
	canvasCreator := canvasCreator{
		dom:       f.dom,
		log:       log,
		canvasCfg: canvasCfg,
	}
	user := user.New(f.dom, log, httpClient)
	board := new(board.Board)
	canvas := canvasCfg.New(f.dom, log, board, ".game>.canvas")
	game := game.New(f.dom, log, board, canvas, canvasCreator)
	lobby := lobby.New(f.dom, log, game)
	socket := socket.New(f.dom, log, user, game, lobby)
	user.Socket = socket   // [circular reference]
	canvas.Socket = socket // [circular reference]
	game.Socket = socket   // [circular reference]
	lobby.Socket = socket  // [circular reference]
	return []domInitializer{log, user, canvas, game, lobby, socket}
}

// canvasCreator creates canvases from the config
type canvasCreator struct {
	dom       *ui.DOM
	log       *log.Log
	canvasCfg canvas.Config
}

// Create uses the canvas config to create a new canvas
func (cc canvasCreator) Create(board *board.Board, canvasParentDivQuery string) game.Canvas {
	return cc.canvasCfg.New(cc.dom, cc.log, board, canvasParentDivQuery)
}
