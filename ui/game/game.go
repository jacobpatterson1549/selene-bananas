// +build js,wasm

// Package game has the ui game logic.
package game

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/ui/game/canvas"
	"github.com/jacobpatterson1549/selene-bananas/ui/log"
)

type (
	// Game handles managing the state of the board and drawing it on the canvas.
	Game struct {
		id          game.ID
		log         *log.Log
		board       *board.Board
		canvas      *canvas.Canvas
		Socket      Socket
		finalBoards map[string]board.Board
	}

	// Config contains the parameters to create a Game.
	Config struct {
		Board  *board.Board
		Canvas *canvas.Canvas
	}

	// Socket sends messages to the server.
	Socket interface {
		Send(m message.Message)
	}
)

// NewGame creates a new game controller with references to the board and canvas.
func (cfg Config) NewGame(log *log.Log) *Game {
	g := Game{
		log:    log,
		board:  cfg.Board,
		canvas: cfg.Canvas,
	}
	return &g
}

// InitDom registers game dom functions.
func (g *Game) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"create":            dom.NewJsFunc(g.startCreate),
		"createWithConfig":  dom.NewJsEventFunc(g.createWithConfig),
		"join":              dom.NewJsEventFunc(g.join),
		"leave":             dom.NewJsFunc(g.sendLeave),
		"delete":            dom.NewJsFunc(g.delete),
		"start":             dom.NewJsFunc(g.Start),
		"finish":            dom.NewJsFunc(g.finish),
		"snagTile":          dom.NewJsFunc(g.snagTile),
		"swapTile":          dom.NewJsFunc(g.startTileSwap),
		"sendChat":          dom.NewJsEventFunc(g.sendChat),
		"resizeTiles":       dom.NewJsFunc(g.resizeTiles),
		"refreshTileLength": dom.NewJsFunc(g.refreshTileLength),
		"viewFinalBoard":    dom.NewJsFunc(g.viewFinalBoard),
	}
	dom.RegisterFuncs(ctx, wg, "game", jsFuncs)
}

// startCreate opens the game tab in create mode.
func (g *Game) startCreate() {
	g.hide(false)
	dom.SetChecked("#hide-game-create", false)
	dom.SetChecked("#tab-game", true)
}

// createWithConfig clears the tiles and asks the server for a new game to join with the create config.
func (g *Game) createWithConfig(event js.Value) {
	checkOnSnag := dom.Checked(".checkOnSnag")
	penalize := dom.Checked(".penalize")
	minLengthStr := dom.Value(".minLength")
	minLength, err := strconv.Atoi(minLengthStr)
	if err != nil {
		g.log.Error("retrieving minimum word length: " + err.Error())
		return
	}
	allowDuplicates := dom.Checked(".allowDuplicates")
	m := message.Message{
		Type: message.CreateGame,
		Game: &game.Info{
			Config: &game.Config{
				CheckOnSnag:     checkOnSnag,
				Penalize:        penalize,
				MinLength:       minLength,
				AllowDuplicates: allowDuplicates,
			},
		},
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
	g.id = game.ID(id)
	m := message.Message{
		Type: message.JoinGame,
	}
	g.setTabActive(m)
}

// hide sets the #hide-game input.
func (g *Game) hide(hideGame bool) {
	dom.SetChecked("#hide-game", hideGame)
}

// ID gets the ID of the game.
func (g Game) ID() game.ID {
	return g.id
}

// sendLeave tells the server to stop sending messages to it and changes tabs.
func (g *Game) sendLeave() {
	g.Socket.Send(message.Message{
		Type: message.LeaveGame,
	})
	g.Leave()
}

// Leave changes the view for game by hiding it.
func (g *Game) Leave() {
	g.id = 0
	g.setFinalBoards(nil)
	g.hide(true)
	dom.SetChecked("#tab-lobby", true)
}

// delete removes everyone from the game and deletes it.
func (g *Game) delete() {
	if dom.Confirm("Are you sure? Deleting the game will kick everyone out.") {
		g.Socket.Send(message.Message{
			Type: message.DeleteGame,
		})
	}
}

// Start triggers the game to start for everyone.
func (g *Game) Start() {
	g.Socket.Send(message.Message{
		Type: message.ChangeGameStatus,
		Game: &game.Info{
			Status: game.InProgress,
		},
	})
}

// finish triggers the game to finish for everyone by checking the players tiles.
func (g *Game) finish() {
	g.Socket.Send(message.Message{
		Type: message.ChangeGameStatus,
		Game: &game.Info{
			Status: game.Finished,
		},
	})
}

// snagTile asks the game to give everone a new tile.
func (g *Game) snagTile() {
	g.Socket.Send(message.Message{
		Type: message.SnagGameTile,
	})
}

// startTileSwap start a swap move on the canvas.
func (g *Game) startTileSwap() {
	g.canvas.StartSwap()
}

// sendChat sends a chat message from the form of the event.
func (g *Game) sendChat(event js.Value) {
	f, err := dom.NewForm(event)
	if err != nil {
		g.log.Error(err.Error())
		return
	}
	info := f.Params.Get("chat")
	f.Reset()
	m := message.Message{
		Type: message.GameChat,
		Info: info,
	}
	g.Socket.Send(m)
}

// replacegameTiles completely replaces the games used and unused tiles.
func (g *Game) replaceGameTiles(m message.Message) {
	g.resetTiles()
	for _, tp := range m.Game.Board.UsedTiles {
		g.board.UsedTiles[tp.Tile.ID] = tp
		if _, ok := g.board.UsedTileLocs[tp.X]; !ok {
			g.board.UsedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
		}
		g.board.UsedTileLocs[tp.X][tp.Y] = tp.Tile
	}
	g.addUnusedTiles(m)
}

// addUnusedTilesappends new tiles onto the game.
func (g *Game) addUnusedTiles(m message.Message) {
	tileStrings := make([]string, 0, len(m.Game.Board.UnusedTiles))
	for _, tID := range m.Game.Board.UnusedTileIDs {
		t, ok := m.Game.Board.UnusedTiles[tID]
		if !ok {
			g.log.Error(("could not add all unused tiles"))
			return
		}
		tileText := `"` + string(t.Ch) + `"`
		tileStrings = append(tileStrings, tileText)
		if err := g.board.AddTile(t); err != nil {
			g.log.Error("could not add unused tile(s): " + err.Error())
			return
		}
	}
	if m.Type != message.JoinGame {
		message := "adding unused tile"
		if len(tileStrings) == 1 {
			message += "s"
		}
		message += ": " + strings.Join(tileStrings, ", ")
		g.log.Info(message)
	}
}

// UpdateInfo updates the game for the specified message.
func (g *Game) UpdateInfo(m message.Message) {
	g.updateStatus(m)
	g.updateTilesLeft(m)
	g.updatePlayers(m)
	switch {
	case m.Game.Board == nil:
		// NOOP
	case len(m.Game.Board.UsedTiles) > 0:
		g.replaceGameTiles(m)
	case len(m.Game.Board.UnusedTiles) > 0:
		g.addUnusedTiles(m)
	}
	g.canvas.Redraw()
	if m.Type == message.JoinGame {
		g.setRules(m.Game.Config.Rules())
		g.id = m.Game.ID
	}
}

// updateStatus sets the statusText and enables or disables the snag, swap, start, and finish buttons.
func (g *Game) updateStatus(m message.Message) {
	var statusText string
	var snagDisabled, swapDisabled, startDisabled, finishDisabled bool
	switch m.Game.Status {
	case game.NotStarted:
		statusText = "Not Started"
		snagDisabled = true
		swapDisabled = true
		finishDisabled = true
	case game.InProgress:
		statusText = "In Progress"
		startDisabled = true
		finishDisabled = m.Game.TilesLeft > 0
	case game.Finished:
		statusText = "Finished"
		snagDisabled = true
		swapDisabled = true
		startDisabled = true
		finishDisabled = true
	default:
		return
	}
	g.setFinalBoards(m.Game.FinalBoards)
	dom.SetValue(".game>.info .status", statusText)
	dom.SetButtonDisabled(".game .actions>.snag", snagDisabled)
	dom.SetButtonDisabled(".game .actions>.swap", swapDisabled)
	dom.SetButtonDisabled(".game .actions>.start", startDisabled)
	dom.SetButtonDisabled(".game .actions>.finish", finishDisabled)
	g.canvas.SetGameStatus(m.Game.Status)
}

// updateTilesLeft updates the TilesLeft label.  Other labels are updated if there are no tiles left.
func (g *Game) updateTilesLeft(m message.Message) {
	dom.SetValue(".game>.info .tiles-left", strconv.Itoa(m.Game.TilesLeft))
	if m.Game.TilesLeft == 0 {
		dom.SetButtonDisabled(".game .actions>.snag", true)
		dom.SetButtonDisabled(".game .actions>.swap", true)
		// enable the finish button if the game is not being started or is already finished
		switch m.Game.Status {
		case game.NotStarted, game.Finished:
			// NOOP
		default:
			dom.SetButtonDisabled(".game .actions>.finish", false)
		}
	}
}

// updatePlayers sets the players list display from the message.
func (g *Game) updatePlayers(m message.Message) {
	if len(m.Game.Players) > 0 {
		players := strings.Join(m.Game.Players, ",")
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
	m := message.Message{
		Type: message.RefreshGameBoard,
	}
	g.setBoardSize(m)
}

// setTabActive performs the actions need to activate the game tab and create or join a game.
func (g *Game) setTabActive(m message.Message) {
	g.hide(false)
	dom.SetChecked("#hide-game-create", true)
	dom.SetChecked("#tab-game", true)
	// the tab now has a size, so update the canvas and board
	parentDivOffsetWidth := g.canvas.ParentDivOffsetWidth()
	g.canvas.UpdateSize(parentDivOffsetWidth)
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
func (g *Game) setBoardSize(m message.Message) {
	g.board.Config.NumCols = g.canvas.NumCols()
	g.board.Config.NumRows = g.canvas.NumRows()
	g.resetTiles()
	if m.Game == nil {
		var i game.Info
		m.Game = &i
	}
	if m.Game.Board == nil {
		var b board.Board
		m.Game.Board = &b
	}
	m.Game.Board.Config = g.board.Config
	g.Socket.Send(m)
}

// setRules replaces the rules for the game
func (g *Game) setRules(rules []string) {
	rulesList := dom.QuerySelector(".game .rules ul")
	rulesList.Set("innerHTML", "")
	for _, r := range rules {
		clone := dom.CloneElement(".game .rules template")
		cloneChildren := clone.Get("children")
		li := cloneChildren.Index(0)
		li.Set("innerHTML", r)
		rulesList.Call("appendChild", li)
	}
}

// setFinalBoards performs the actions needed to allow final boards to be viewed.
// If no boards are specified, the tab is hidden.
// The board canvas is always cleared, requiring the user to select one, if any.
func (g *Game) setFinalBoards(finalBoards map[string]board.Board) {
	hideFinalBoards := len(finalBoards) == 0
	dom.SetChecked("#hide-final-boards", hideFinalBoards)
	playersList := dom.QuerySelector(".final-boards .player-list form")
	playersList.Set("innerHTML", "")
	g.finalBoards = finalBoards
	for playerName := range finalBoards {
		div := g.newFinalBoardDiv(playerName)
		playersList.Call("appendChild", div)
	}
	canvas := dom.QuerySelector(".final-boards .canvas canvas")
	canvas.Set("height", 0)
}

// newFinalBoardLi creates a new div to trigger drawing the board.
func (g *Game) newFinalBoardDiv(playerName string) js.Value {
	clone := dom.CloneElement(".final-boards .player-list template")
	cloneChildren := clone.Get("children")
	div := cloneChildren.Index(0)
	divChildren := div.Get("children")
	id := playerName + "-final-board"
	input := divChildren.Index(0)
	input.Set("id", id)
	label := divChildren.Index(1)
	label.Set("htmlFor", id)
	label.Set("innerHTML", playerName)
	return div
}

// viewFinalBoard draws the board for the clicked player on the .final-boards canvas.
func (g *Game) viewFinalBoard() {
	checkedLabel := dom.QuerySelector(".player-list input:checked+label")
	playerName := checkedLabel.Get("innerHTML").String()
	b, ok := g.finalBoards[playerName]
	if !ok {
		g.log.Error("could not view final board for " + playerName)
		return
	}
	tileLength := g.canvas.TileLength()
	cfg := canvas.Config{
		TileLength: tileLength,
	}
	canvas := cfg.New(g.log, &b, ".final-boards .canvas")
	width := cfg.DesiredWidth(b)
	canvas.UpdateSize(width)
	canvas.Redraw()
}
