// +build js,wasm

// Package controller has the ui game logic.
package controller

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
)

type (
	Game struct {
		board  *board.Board
		canvas *canvas.Canvas
	}
)

// NewGame creates a new game controller with references to the board and canvas.
func NewGame(board *board.Board, canvas *canvas.Canvas) Game {
	g := Game{
		board:  board,
		canvas: canvas,
	}
	return g
}

// InitDom regesters game dom functions.
func (g *Game) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	createJsFunc := dom.NewJsFunc(g.Create)
	joinJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		joinGameButton := event.Get("srcElement")
		gameIdInput := joinGameButton.Get("previousElementSibling")
		idText := gameIdInput.Get("value").String()
		id, err := strconv.Atoi(idText)
		if err != nil {
			log.Error("could not get Id of game: " + err.Error())
			return
		}
		g.Join(id)
	})
	leaveJsFunc := dom.NewJsFunc(g.Leave)
	deleteJsFunc := dom.NewJsFunc(g.Delete)
	startJsFunc := dom.NewJsFunc(g.Start)
	finishJsFunc := dom.NewJsFunc(g.Finish)
	snagTileJsFunc := dom.NewJsFunc(g.SnagTile)
	sendChatJsFunc := dom.NewJsEventFunc(func(event js.Value) {
		f := dom.NewForm(event)
		message := f.Params.Get("chat")
		f.Reset()
		g.SendChat(message)
	})
	dom.RegisterFunc("game", "create", createJsFunc)
	dom.RegisterFunc("game", "join", joinJsFunc)
	dom.RegisterFunc("game", "leave", leaveJsFunc)
	dom.RegisterFunc("game", "delete", deleteJsFunc)
	dom.RegisterFunc("game", "start", startJsFunc)
	dom.RegisterFunc("game", "finish", finishJsFunc)
	dom.RegisterFunc("game", "snagTile", snagTileJsFunc)
	dom.RegisterFunc("game", "sendChat", sendChatJsFunc)
	go func() {
		<-ctx.Done()
		createJsFunc.Release()
		joinJsFunc.Release()
		leaveJsFunc.Release()
		deleteJsFunc.Release()
		startJsFunc.Release()
		finishJsFunc.Release()
		snagTileJsFunc.Release()
		sendChatJsFunc.Release()
		wg.Done()
	}()
}

// Create clears the tiles and asks the server for a new game to join
func (g *Game) Create() {
	g.resetTiles()
	dom.Send(game.Message{
		Type: game.Create,
	})
}

// Join asks the server to join an existing game.
func (g *Game) Join(id int) {
	g.resetTiles()
	dom.Send(game.Message{
		Type:   game.Join,
		GameID: game.ID(id),
	})
}

// Leave changes the view for game by hiding it.
func (g *Game) Leave() {
	dom.SetChecked("has-game", false)
	dom.SetChecked("tab-4", true) // lobby tab
}

// Delete removes everyone from the game and deletes it.
func (g *Game) Delete() {
	if dom.Confirm("Are you sure? Deleting the game will kick everyone out.") {
		dom.Send(game.Message{
			Type: game.Delete,
		})
	}
}

// Starts triggers the game to start for everyone.
func (g *Game) Start() {
	dom.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.InProgress,
	})
}

// Starts triggers the game to finish for everyone by checking the players tiles.
func (g *Game) Finish() {
	dom.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.Finished,
	})
}

// SnagTile asks the game to give everone a new tile.
func (g *Game) SnagTile() {
	dom.Send(game.Message{
		Type: game.Snag,
	})
}

// SendChat sends a chat message.
func (g *Game) SendChat(message string) {
	dom.Send(game.Message{
		Type: game.Chat,
		Info: message,
	})
}

// ReplacegameTiles completely replaces the games used and unused tiles
func (g *Game) ReplaceGameTiles(unusedTiles []tile.Tile, tilePositions []tile.Position, silent bool) {
	g.resetTiles()
	if len(tilePositions) > 0 {
		for _, tp := range tilePositions {
			g.board.UsedTiles[tp.Tile.ID] = tp
			if _, ok := g.board.UsedTileLocs[tp.X]; !ok {
				g.board.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
			}
			g.board.UsedTileLocs[tp.X][tp.Y] = tp.Tile
		}
	}
	g.AddUnusedTiles(unusedTiles, silent)
}

// AddUnusedTilesappends new tiles onto the game
func (g *Game) AddUnusedTiles(unusedTiles []tile.Tile, silent bool) {
	tileStrings := make([]string, len(unusedTiles))
	for i, t := range unusedTiles {
		tileStrings[i] = `"` + t.Ch.String() + `"`
		g.board.UnusedTiles[t.ID] = t
		g.board.UnusedTileIDs = append(g.board.UnusedTileIDs, t.ID) // TODO: inefficient, use copy to increase capacity
	}
	if !silent {
		message := "adding unused tile"
		if len(tileStrings) == 1 {
			message += "s"
		}
		message += ": " + strings.Join(tileStrings, ", ")
		log.Info(message)
	}
	g.canvas.Redraw()
	setTabActive()
}

// SetStatus updates the game for the specified status.
func (g *Game) SetStatus(status game.Status) {
	var statusText string
	var snagDisabled, swapDisabled, startDisabled, finishDisabled bool
	if status > 0 {
		switch status {
		case game.NotStarted:
			statusText = "Not Started"
			snagDisabled = true
			swapDisabled = true
			finishDisabled = true
		case game.InProgress:
			statusText = "In Progress"
			g.canvas.Redraw()
			startDisabled = true
			finishDisabled = true
		case game.Finished:
			statusText = "Finished"
			snagDisabled = true
			swapDisabled = true
			startDisabled = true
			finishDisabled = true
		default:
			return
		}
	}
	dom.SetValue("game-status", statusText)
	dom.SetButtonDisabled("game-snag", snagDisabled)
	dom.SetButtonDisabled("game-swap", swapDisabled)
	dom.SetButtonDisabled("game-start", startDisabled)
	dom.SetButtonDisabled("game-finish", finishDisabled)
	g.canvas.GameStatus = status
}

func (g *Game) resetTiles() {
	g.board.UnusedTiles = make(map[tile.ID]tile.Tile)
	g.board.UnusedTileIDs = make([]tile.ID, 0)
	g.board.UsedTiles = make(map[tile.ID]tile.Position)
	g.board.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
}

func setTabActive() {
	dom.SetChecked("has-game", true)
	dom.SetChecked("tab-5", true) // game tab
}