// +build js

// Package controller has the ui game logic.
package controller

import (
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
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

// Create clears the tiles and asks the server for a new game to join
func (g *Game) Create() {
	g.resetTiles()
	js.Send(game.Message{
		Type: game.Create,
	})
}

// Join asks the server to join an existing game.
func (g *Game) Join(id int) {
	g.resetTiles()
	js.Send(game.Message{
		Type:   game.Join,
		GameID: game.ID(id),
	})
}

// Leave changes the view for game by hiding it.
func (g *Game) Leave() {
	js.SetChecked("has-game", false)
	js.SetChecked("tab-4", true) // lobby tab
}

// Delete removes everyone from the game and deletes it.
func (g *Game) Delete() {
	if js.Confirm("Are you sure? Deleting the game will kick everyone out.") {
		js.Send(game.Message{
			Type: game.Delete,
		})
	}
}

// Starts triggers the game to start for everyone.
func (g *Game) Start() {
	js.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.InProgress,
	})
}

// Starts triggers the game to finish for everyone by checking the players tiles.
func (g *Game) Finish() {
	js.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.Finished,
	})
}

// SnagTile asks the game to give everone a new tile.
func (g *Game) SnagTile() {
	js.Send(game.Message{
		Type: game.Snag,
	})
}

// SendChat sends a chat message.
func (g *Game) SendChat(message string) {
	js.Send(game.Message{
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
	js.SetValue("game-status", statusText)
	js.SetButtonDisabled("game-snag", snagDisabled)
	js.SetButtonDisabled("game-swap", swapDisabled)
	js.SetButtonDisabled("game-start", startDisabled)
	js.SetButtonDisabled("game-finish", finishDisabled)
	g.canvas.GameStatus = status
}

func (g *Game) resetTiles() {
	g.board.UnusedTiles = make(map[tile.ID]tile.Tile)
	g.board.UnusedTileIDs = make([]tile.ID, 0)
	g.board.UsedTiles = make(map[tile.ID]tile.Position)
	g.board.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
}

func setTabActive() {
	js.SetChecked("has-game", true)
	js.SetChecked("tab-5", true) // game tab
}
