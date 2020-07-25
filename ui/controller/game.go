// +build js,wasm

// Package controller has the ui game logic.
package controller

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/canvas"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// Game handles managing the state of the board and drawing it on the canvas.
	Game struct {
		log    *log.Log
		board  *board.Board
		canvas *canvas.Canvas
		Socket Socket
	}

	// GameConfig contains the parameters to create a Game.
	GameConfig struct {
		Log    *log.Log
		Board  *board.Board
		Canvas *canvas.Canvas
	}

	// Socket sends messages to the server.
	Socket interface {
		Send(m game.Message)
	}
)

// NewGame creates a new game controller with references to the board and canvas.
func (cfg GameConfig) NewGame() *Game {
	g := Game{
		log:    cfg.Log,
		board:  cfg.Board,
		canvas: cfg.Canvas,
	}
	return &g
}

// InitDom regesters game dom functions.
func (g *Game) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"create":            dom.NewJsFunc(g.Create),
		"join":              dom.NewJsEventFunc(g.join),
		"leave":             dom.NewJsFunc(g.Leave),
		"delete":            dom.NewJsFunc(g.Delete),
		"start":             dom.NewJsFunc(g.Start),
		"finish":            dom.NewJsFunc(g.Finish),
		"snagTile":          dom.NewJsFunc(g.SnagTile),
		"sendChat":          dom.NewJsEventFunc(g.sendChat),
		"resizeTiles":       dom.NewJsFunc(g.resizeTiles),
		"refreshTileLength": dom.NewJsFunc(g.refreshTileLength),
	}
	dom.RegisterFuncs(ctx, wg, "game", jsFuncs)
}

// Create clears the tiles and asks the server for a new game to join.
func (g *Game) Create() {
	m := game.Message{
		Type: game.Create,
	}
	g.setTabActive(m)
}

// join asks the server to join an existing game.
func (g *Game) join(event js.Value) {
	joinGameButton := event.Get("srcElement")
	gameIDInput := joinGameButton.Get("previousElementSibling")
	idText := gameIDInput.Get("value").String()
	id, err := strconv.Atoi(idText)
	if err != nil {
		g.log.Error("could not get Id of game: " + err.Error())
		return
	}
	m := game.Message{
		Type:   game.Join,
		GameID: game.ID(id),
	}
	g.setTabActive(m)
}

// Leave changes the view for game by hiding it.
func (g *Game) Leave() {
	dom.SetChecked(".has-game", false)
	dom.SetChecked("#tab-lobby", true)
}

// Delete removes everyone from the game and deletes it.
func (g *Game) Delete() {
	if dom.Confirm("Are you sure? Deleting the game will kick everyone out.") {
		g.Socket.Send(game.Message{
			Type: game.Delete,
		})
	}
}

// Start triggers the game to start for everyone.
func (g *Game) Start() {
	g.Socket.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.InProgress,
	})
}

// Finish triggers the game to finish for everyone by checking the players tiles.
func (g *Game) Finish() {
	g.Socket.Send(game.Message{
		Type:       game.StatusChange,
		GameStatus: game.Finished,
	})
}

// SnagTile asks the game to give everone a new tile.
func (g *Game) SnagTile() {
	g.Socket.Send(game.Message{
		Type: game.Snag,
	})
}

// sendChat sends a chat message from the form of the event.
func (g *Game) sendChat(event js.Value) {
	f, err := dom.NewForm(event)
	if err != nil {
		g.log.Error(err.Error())
		return
	}
	message := f.Params.Get("chat")
	f.Reset()
	g.Socket.Send(game.Message{
		Type: game.Chat,
		Info: message,
	})
}

// replacegameTiles completely replaces the games used and unused tiles.
func (g *Game) replaceGameTiles(m game.Message) {
	g.resetTiles()
	for _, tp := range m.TilePositions {
		g.board.UsedTiles[tp.Tile.ID] = tp
		if _, ok := g.board.UsedTileLocs[tp.X]; !ok {
			g.board.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
		}
		g.board.UsedTileLocs[tp.X][tp.Y] = tp.Tile
	}
	g.addUnusedTiles(m)
}

// addUnusedTilesappends new tiles onto the game.
func (g *Game) addUnusedTiles(m game.Message) {
	tileStrings := make([]string, len(m.Tiles))
	for i, t := range m.Tiles {
		tileStrings[i] = `"` + t.Ch.String() + `"`
		if err := g.board.AddTile(t); err != nil {
			g.log.Error("could not add unused tile(s): " + err.Error())
			return
		}
	}
	if m.Type != game.Join {
		message := "adding unused tile"
		if len(tileStrings) == 1 {
			message += "s"
		}
		message += ": " + strings.Join(tileStrings, ", ")
		g.log.Info(message)
	}
	g.canvas.Redraw()
}

// UpdateInfo updates the game for the specified message.
func (g *Game) UpdateInfo(m game.Message) {
	g.updateStatus(m)
	g.updateTilesLeft(m)
	g.updatePlayers(m)
	switch {
	case len(m.TilePositions) > 0:
		g.replaceGameTiles(m)
	case len(m.Tiles) > 0:
		g.addUnusedTiles(m)
	}
}

// updateStatus sets the statusText and enables or disables the snag, swap, start, and finish buttons.
func (g *Game) updateStatus(m game.Message) {
	var statusText string
	var snagDisabled, swapDisabled, startDisabled, finishDisabled bool
	switch m.GameStatus {
	case game.NotStarted:
		statusText = "Not Started"
		snagDisabled = true
		swapDisabled = true
		finishDisabled = true
	case game.InProgress:
		statusText = "In Progress"
		g.canvas.Redraw()
		startDisabled = true
		finishDisabled = m.TilesLeft > 0
	case game.Finished:
		statusText = "Finished"
		snagDisabled = true
		swapDisabled = true
		startDisabled = true
		finishDisabled = true
	default:
		return
	}
	dom.SetValue(".game>.info .status", statusText)
	dom.SetButtonDisabled(".game>.actions>.snag", snagDisabled)
	dom.SetButtonDisabled(".game>.actions>.swap", swapDisabled)
	dom.SetButtonDisabled(".game>.actions>.start", startDisabled)
	dom.SetButtonDisabled(".game>.actions>.finish", finishDisabled)
	g.canvas.SetGameStatus(m.GameStatus)
}

// updateTilesLeft updates the TilesLeft label.  Other labels are updated if there are no tiles left.
func (g *Game) updateTilesLeft(m game.Message) {
	dom.SetValue(".game>.info .tiles-left", strconv.Itoa(m.TilesLeft))
	if m.TilesLeft == 0 {
		dom.SetButtonDisabled(".game>.actions>.snag", true)
		dom.SetButtonDisabled(".game>.actions>.swap", true)
		// enable the finish button if the game is not being started or is already finished
		switch m.GameStatus {
		case game.NotStarted, game.Finished:
		default:
			dom.SetButtonDisabled(".game>.actions>.finish", false)
		}
	}
}

// updatePlayers sets the players list display from the message.
func (g *Game) updatePlayers(m game.Message) {
	if len(m.GamePlayers) > 0 {
		players := strings.Join(m.GamePlayers, ",")
		dom.SetValue(".game>.info .players", players)
	}
}

// resetTiles clears the tiles on the board.
func (g *Game) resetTiles() {
	g.board.UnusedTiles = make(map[tile.ID]tile.Tile)
	g.board.UnusedTileIDs = make([]tile.ID, 0)
	g.board.UsedTiles = make(map[tile.ID]tile.Position)
	g.board.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
}

// refreshTileLength updates the displayed tile length number.
func (g *Game) refreshTileLength() {
	tileLengthStr := dom.Value(".tile-length-slider")
	dom.SetValue(".tile-length-display", tileLengthStr)
}

// resizeTiles changes the tile size of the board to be the value of the slider.
func (g *Game) resizeTiles() {
	tileLengthStr := dom.Value(".tile-length-slider")
	tileLength, err := strconv.Atoi(tileLengthStr)
	if err != nil {
		g.log.Error("retrieving tile size: " + err.Error())
		return
	}
	g.canvas.SetTileLength(tileLength)
	m := game.Message{
		Type: game.BoardSize,
	}
	g.setBoardSize(m)
}

// setTabActive performs the actions need to activate the game tab and create or join a game.
func (g *Game) setTabActive(m game.Message) {
	dom.SetChecked(".has-game", true)
	dom.SetChecked("#tab-game", true)
	// the tab now has a size, so update the canvas and board
	g.canvas.UpdateSize()
	cfg := board.Config{
		NumCols: g.canvas.NumCols(),
		NumRows: g.canvas.NumRows(),
	}
	if err := cfg.Validate(); err != nil {
		g.log.Error("cannot open game: " + err.Error())
		g.Leave()
		return
	}
	g.setBoardSize(m)
}

// setBoardSize updates the board size due to a recent canvas update, adds the updated size to the message, and sends it.
func (g *Game) setBoardSize(m game.Message) {
	g.board.NumCols = g.canvas.NumCols()
	g.board.NumRows = g.canvas.NumRows()
	g.resetTiles()
	m.NumCols = g.board.NumCols
	m.NumRows = g.board.NumRows
	g.Socket.Send(m)
}
