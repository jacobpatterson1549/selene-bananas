// +build js,wasm

// Package main initializes interactive frontend elements and runs as long as the webpage is open.
package main

import (
	"context"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/ui/game"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/canvas"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/lobby"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/socket"
	"github.com/jacobpatterson1549/selene-bananas/ui/http/xhr"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/ui/user"
)

// mainFlags contains options for the the ui.
type mainFlags struct {
	httpTimeout time.Duration
	tileLength  int
}

// initDom creates, initializes, and links up dom components.
func (m mainFlags) initDom(ctx context.Context, wg *sync.WaitGroup) {
	log := m.log(ctx, wg)
	user := m.user(ctx, wg, log)
	board := new(board.Board)
	canvas := m.canvas(ctx, wg, log, board)
	game := m.game(ctx, wg, log, board, canvas)
	lobby := m.lobby(ctx, wg, log, game)
	socket := m.socket(ctx, wg, log, user, game, lobby)
	user.Socket = socket   // [circular reference]
	canvas.Socket = socket // [circular reference]
	game.Socket = socket   // [circular reference]
	lobby.Socket = socket  // [circular reference]
}

// log creates and initializes the log component.
func (mainFlags) log(ctx context.Context, wg *sync.WaitGroup) *log.Log {
	log := new(log.Log)
	log.InitDom(ctx, wg)
	return log
}

// user creates and initializes the user/form/http component.
func (m mainFlags) user(ctx context.Context, wg *sync.WaitGroup, log *log.Log) *user.User {
	cfg := user.Config{
		Log: log,
	}
	httpClient := xhr.HTTPClient{
		Timeout: m.httpTimeout,
	}
	user := cfg.New(httpClient)
	user.InitDom(ctx, wg)
	return user
}

// canvas creates and initializes the game drawing component with elements from the dom.
func (m mainFlags) canvas(ctx context.Context, wg *sync.WaitGroup, log *log.Log, board *board.Board) *canvas.Canvas {
	cfg := canvas.Config{
		Log:        log,
		TileLength: m.tileLength,
	}
	canvas := cfg.New(board)
	canvas.InitDom(ctx, wg)
	return canvas
}

// game creates and initializes the game component.
func (mainFlags) game(ctx context.Context, wg *sync.WaitGroup, log *log.Log, board *board.Board, canvas *canvas.Canvas) *game.Game {
	cfg := game.Config{
		Log:    log,
		Board:  board,
		Canvas: canvas,
	}
	game := cfg.NewGame()
	game.InitDom(ctx, wg)
	return game
}

// lobby creates and initializes the game lobby component.
func (mainFlags) lobby(ctx context.Context, wg *sync.WaitGroup, log *log.Log, game *game.Game) *lobby.Lobby {
	cfg := lobby.Config{
		Log:  log,
		Game: game,
	}
	lobby := cfg.New()
	lobby.InitDom(ctx, wg)
	return lobby
}

// socket creates and initializes the player socket component for connection to the lobby.
func (mainFlags) socket(ctx context.Context, wg *sync.WaitGroup, log *log.Log, user *user.User, game *game.Game, lobby *lobby.Lobby) *socket.Socket {
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
