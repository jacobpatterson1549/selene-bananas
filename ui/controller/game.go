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
		board  *board.Board
		canvas *canvas.Canvas
		Socket Socket
	}

	// Socket sends messages to the server.
	Socket interface {
		Send(m game.Message)
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
	joinJsFunc := dom.NewJsEventFunc(g.join)
	leaveJsFunc := dom.NewJsFunc(g.Leave)
	deleteJsFunc := dom.NewJsFunc(g.Delete)
	startJsFunc := dom.NewJsFunc(g.Start)
	finishJsFunc := dom.NewJsFunc(g.Finish)
	snagTileJsFunc := dom.NewJsFunc(g.SnagTile)
	sendChatJsFunc := dom.NewJsEventFunc(g.sendChat)
	resizeTilesJsFunc := dom.NewJsFunc(g.resizeTiles)
	refreshTileLengthJsFunc := dom.NewJsFunc(g.refreshTileLength)
	dom.RegisterFunc("game", "create", createJsFunc)
	dom.RegisterFunc("game", "join", joinJsFunc)
	dom.RegisterFunc("game", "leave", leaveJsFunc)
	dom.RegisterFunc("game", "delete", deleteJsFunc)
	dom.RegisterFunc("game", "start", startJsFunc)
	dom.RegisterFunc("game", "finish", finishJsFunc)
	dom.RegisterFunc("game", "snagTile", snagTileJsFunc)
	dom.RegisterFunc("game", "sendChat", sendChatJsFunc)
	dom.RegisterFunc("game", "resizeTiles", resizeTilesJsFunc)
	dom.RegisterFunc("game", "refreshTileLength", refreshTileLengthJsFunc)
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
		resizeTilesJsFunc.Release()
		refreshTileLengthJsFunc.Release()
		wg.Done()
	}()
}

// Create clears the tiles and asks the server for a new game to join
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
		log.Error("could not get Id of game: " + err.Error())
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
	dom.SetCheckedQuery(".has-game", false)
	dom.SetCheckedQuery("#tab-lobby", true)
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
		log.Error(err.Error())
		return
	}
	message := f.Params.Get("chat")
	f.Reset()
	g.Socket.Send(game.Message{
		Type: game.Chat,
		Info: message,
	})
}

// replacegameTiles completely replaces the games used and unused tiles
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

// addUnusedTilesappends new tiles onto the game
func (g *Game) addUnusedTiles(m game.Message) {
	tileStrings := make([]string, len(m.Tiles))
	for i, t := range m.Tiles {
		tileStrings[i] = `"` + t.Ch.String() + `"`
		if err := g.board.AddTile(t); err != nil {
			log.Error("could not add unused tile(s): " + err.Error())
			return
		}
	}
	if m.Type != game.Join {
		message := "adding unused tile"
		if len(tileStrings) == 1 {
			message += "s"
		}
		message += ": " + strings.Join(tileStrings, ", ")
		log.Info(message)
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
	g.canvas.GameStatus(m.GameStatus)
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

func (g *Game) updatePlayers(m game.Message) {
	if len(m.GamePlayers) > 0 {
		players := strings.Join(m.GamePlayers, ",")
		dom.SetValue(".game>.info .players", players)
	}
}

func (g *Game) resetTiles() {
	g.board.UnusedTiles = make(map[tile.ID]tile.Tile)
	g.board.UnusedTileIDs = make([]tile.ID, 0)
	g.board.UsedTiles = make(map[tile.ID]tile.Position)
	g.board.UsedTileLocs = make(map[tile.X]map[tile.Y]tile.Tile)
}

func (g *Game) refreshTileLength() {
	tileLengthStr := dom.GetValue(`.tile-length input[name="tile-length"]`)
	dom.SetValue(`.tile-length input[name="tile-length-display"]`, tileLengthStr)
}

func (g *Game) resizeTiles() {
	tileLengthStr := dom.GetValue(`.tile-length input[name="tile-length"]`)
	tileLength, err := strconv.Atoi(tileLengthStr)
	if err != nil {
		log.Error("retrieving tile size: " + err.Error())
		return
	}
	g.canvas.TileLength(tileLength)
	m := game.Message{
		Type: game.BoardSize,
	}
	g.boardSize(m)
}

// setTabActive performs the actions need to activate the game tab and create or join a game.
func (g *Game) setTabActive(m game.Message) {
	dom.SetCheckedQuery(".has-game", true)
	dom.SetCheckedQuery("#tab-game", true)
	// the tab now has a size, so update the canvas and board
	g.canvas.UpdateSize()
	cfg := board.Config{
		NumCols: g.canvas.NumCols(),
		NumRows: g.canvas.NumRows(),
	}
	if err := cfg.Validate(); err != nil {
		log.Error("cannot open game: " + err.Error())
		g.Leave()
		return
	}
	g.boardSize(m)
}

// boardSize updates the board size due to a recent canvas update, adds the updated size to the message, and sends it.
func (g *Game) boardSize(m game.Message) {
	g.board.NumCols = g.canvas.NumCols()
	g.board.NumRows = g.canvas.NumRows()
	g.resetTiles()
	m.NumCols = g.board.NumCols
	m.NumRows = g.board.NumRows
	g.Socket.Send(m)
}
