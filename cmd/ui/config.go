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

func (f *flags) createDomInitializers() []domInitializer {
	timeFunc := func() int64 {
		return time.Now().Unix()
	}
	log := f.log(timeFunc)
	user := f.user(log)
	board := new(board.Board)
	canvas := f.canvas(log, board)
	game := f.game(log, board, canvas)
	lobby := f.lobby(log, game)
	socket := f.socket(log, user, game, lobby)
	user.Socket = socket   // [circular reference]
	canvas.Socket = socket // [circular reference]
	game.Socket = socket   // [circular reference]
	lobby.Socket = socket  // [circular reference]
	return []domInitializer{log, user, canvas, game, lobby, socket}
}

// log creates and initializes the log component.
func (f flags) log(timeFunc func() int64) *log.Log {
	return log.New(f.dom, timeFunc)
}

// user creates and initializes the user/form/http component.
func (f flags) user(log *log.Log) *user.User {
	httpClient := http.Client{
		Timeout: f.httpTimeout,
	}
	return user.New(f.dom, log, httpClient)
}

// canvas creates and initializes the game drawing component with elements from the dom.
func (f flags) canvas(log *log.Log, board *board.Board) *canvas.Canvas {
	cfg := canvas.Config{
		TileLength: f.tileLength,
	}
	return cfg.New(f.dom, log, board, ".game>.canvas")
}

// game creates and initializes the game component.
func (f flags) game(log *log.Log, board *board.Board, canvas *canvas.Canvas) *game.Game {
	cfg := game.Config{
		Board:  board,
		Canvas: canvas,
	}
	return cfg.New(f.dom, log)
}

// lobby creates and initializes the game lobby component.
func (f flags) lobby(log *log.Log, game *game.Game) *lobby.Lobby {
	return lobby.New(f.dom, log, game)
}

// socket creates and initializes the player socket component for connection to the lobby.
func (f flags) socket(log *log.Log, user *user.User, game *game.Game, lobby *lobby.Lobby) *socket.Socket {
	return socket.New(f.dom, log, user, game, lobby)
}
