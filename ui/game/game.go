//go:build js && wasm

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
	"github.com/jacobpatterson1549/selene-bananas/ui"
)

type (
	// Game handles managing the state of the board and drawing it on the canvas.
	Game struct {
		dom           DOM
		id            game.ID
		log           Log
		board         *board.Board
		canvas        Canvas
		canvasCreator CanvasCreator
		Socket        Socket
		finalBoards   map[string]board.Board
	}

	// Socket sends messages to the server.
	Socket interface {
		Send(m message.Message)
	}

	// DOM interacts with the page.
	DOM interface {
		QuerySelector(query string) js.Value
		QuerySelectorAll(document js.Value, query string) []js.Value
		Checked(query string) bool
		SetChecked(query string, checked bool)
		Value(query string) string
		SetValue(query, value string)
		SetButtonDisabled(query string, disabled bool)
		CloneElement(query string) js.Value
		Confirm(message string) bool
		Color(element js.Value) string
		RegisterFuncs(ctx context.Context, wg *sync.WaitGroup, parentName string, jsFuncs map[string]js.Func)
		NewJsFunc(fn func()) js.Func
		NewJsEventFunc(fn func(event js.Value)) js.Func
		ReleaseJsFuncsOnDone(ctx context.Context, wg *sync.WaitGroup, jsFuncs map[string]js.Func)
	}

	// Log is notify users about changes to the game.
	Log interface {
		Error(text string)
		Info(text string)
	}

	// Canvas is the element the game is drawn on
	Canvas interface {
		StartSwap()
		Redraw()
		SetGameStatus(s game.Status)
		TileLength() int
		SetTileLength(tileLength int)
		ParentDivOffsetWidth() int
		DesiredWidth() int
		UpdateSize(width int)
		NumRows() int
		NumCols() int
	}

	// CanvasCreator creates canvases to draw other player's final boards.
	CanvasCreator interface {
		Create(board *board.Board, canvasParentDivQuery string) Canvas
	}
)

// New creates a new game controller with references to the board and canvas.
func New(dom DOM, log Log, board *board.Board, canvas Canvas, createCanvas CanvasCreator) *Game {
	g := Game{
		dom:           dom,
		log:           log,
		board:         board,
		canvas:        canvas,
		canvasCreator: createCanvas,
	}
	return &g
}

// InitDom registers game dom functions.
func (g *Game) InitDom(ctx context.Context, wg *sync.WaitGroup) {
	jsFuncs := map[string]js.Func{
		"create":            g.dom.NewJsFunc(g.startCreate),
		"createWithConfig":  g.dom.NewJsEventFunc(g.createWithConfig),
		"join":              g.dom.NewJsEventFunc(g.join),
		"leave":             g.dom.NewJsFunc(g.sendLeave),
		"delete":            g.dom.NewJsFunc(g.delete),
		"start":             g.dom.NewJsFunc(g.Start),
		"finish":            g.dom.NewJsFunc(g.finish),
		"snagTile":          g.dom.NewJsFunc(g.snagTile),
		"swapTile":          g.dom.NewJsFunc(g.startTileSwap),
		"sendChat":          g.dom.NewJsEventFunc(g.sendChat),
		"resizeTiles":       g.dom.NewJsFunc(g.resizeTiles),
		"refreshTileLength": g.dom.NewJsFunc(g.refreshTileLength),
		"viewFinalBoard":    g.dom.NewJsFunc(g.viewFinalBoard),
	}
	g.dom.RegisterFuncs(ctx, wg, "game", jsFuncs)
}

// startCreate opens the game tab in create mode.
func (g *Game) startCreate() {
	g.hide(false)
	g.dom.SetChecked("#hide-game-create", false)
	g.dom.SetChecked("#tab-game", true)
}

// createWithConfig clears the tiles and asks the server for a new game to join with the create config.
func (g *Game) createWithConfig(event js.Value) {
	checkOnSnag := g.dom.Checked(".checkOnSnag")
	penalize := g.dom.Checked(".penalize")
	minLengthStr := g.dom.Value(".minLength")
	minLength, err := strconv.Atoi(minLengthStr)
	if err != nil {
		g.log.Error("retrieving minimum word length: " + err.Error())
		return
	}
	prohibitDuplicates := g.dom.Checked(".prohibitDuplicates")
	m := message.Message{
		Type: message.CreateGame,
		Game: &game.Info{
			Config: &game.Config{
				CheckOnSnag:        checkOnSnag,
				Penalize:           penalize,
				MinLength:          minLength,
				ProhibitDuplicates: prohibitDuplicates,
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
	g.dom.SetChecked("#hide-game", hideGame)
}

// ID gets the ID of the game.
func (g Game) ID() game.ID {
	return g.id
}

// sendLeave tells the server to stop sending messages to it and changes tabs.
func (g *Game) sendLeave() {
	m := message.Message{
		Type: message.LeaveGame,
	}
	g.Socket.Send(m)
	g.Leave()
}

// Leave changes the view for game by hiding it.
func (g *Game) Leave() {
	g.id = 0
	g.setFinalBoards(nil)
	g.hide(true)
	g.dom.SetChecked("#tab-lobby", true)
}

// delete removes everyone from the game and deletes it.
func (g *Game) delete() {
	if ok := g.dom.Confirm("Are you sure? Deleting the game will kick everyone out."); !ok {
		return
	}
	m := message.Message{
		Type: message.DeleteGame,
	}
	g.Socket.Send(m)
}

// Start triggers the game to start for everyone.
func (g *Game) Start() {
	m := message.Message{
		Type: message.ChangeGameStatus,
		Game: &game.Info{
			Status: game.InProgress,
		},
	}
	g.Socket.Send(m)
}

// finish triggers the game to finish for everyone by checking the players tiles.
func (g *Game) finish() {
	m := message.Message{
		Type: message.ChangeGameStatus,
		Game: &game.Info{
			Status: game.Finished,
		},
	}
	g.Socket.Send(m)
}

// snagTile asks the game to give everone a new tile.
func (g *Game) snagTile() {
	m := message.Message{
		Type: message.SnagGameTile,
	}
	g.Socket.Send(m)
}

// startTileSwap start a swap move on the canvas.
func (g *Game) startTileSwap() {
	g.canvas.StartSwap()
}

// sendChat sends a chat message from the form of the event.
func (g *Game) sendChat(event js.Value) {
	f, err := ui.NewForm(g.dom.QuerySelectorAll, event)
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
	var snagDisabled, swapDisabled, startDisabled, finishDisabled bool
	switch m.Game.Status {
	case game.NotStarted:
		snagDisabled = true
		swapDisabled = true
		finishDisabled = true
	case game.InProgress:
		startDisabled = true
		finishDisabled = m.Game.TilesLeft > 0
	case game.Finished:
		snagDisabled = true
		swapDisabled = true
		startDisabled = true
		finishDisabled = true
	default:
		return
	}
	statusText := m.Game.Status.String()
	g.setFinalBoards(m.Game.FinalBoards)
	g.dom.SetValue(".game>.info .status", statusText)
	g.dom.SetButtonDisabled(".game .actions>.snag", snagDisabled)
	g.dom.SetButtonDisabled(".game .actions>.swap", swapDisabled)
	g.dom.SetButtonDisabled(".game .actions>.start", startDisabled)
	g.dom.SetButtonDisabled(".game .actions>.finish", finishDisabled)
	g.canvas.SetGameStatus(m.Game.Status)
}

// updateTilesLeft updates the TilesLeft label.  Other labels are updated if there are no tiles left.
func (g *Game) updateTilesLeft(m message.Message) {
	g.dom.SetValue(".game>.info .tiles-left", strconv.Itoa(m.Game.TilesLeft))
	if m.Game.TilesLeft == 0 {
		g.dom.SetButtonDisabled(".game .actions>.snag", true)
		g.dom.SetButtonDisabled(".game .actions>.swap", true)
		// enable the finish button if the game is not being started or is already finished
		switch m.Game.Status {
		case game.NotStarted, game.Finished:
			// NOOP
		default:
			g.dom.SetButtonDisabled(".game .actions>.finish", false)
		}
	}
}

// updatePlayers sets the players list display from the message.
func (g *Game) updatePlayers(m message.Message) {
	if len(m.Game.Players) == 0 {
		return
	}
	players := strings.Join(m.Game.Players, ",")
	g.dom.SetValue(".game>.info .players", players)
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
	tileLengthStr := g.dom.Value(".tile-length-slider")
	g.dom.SetValue(".tile-length-display", tileLengthStr)
}

// resizeTiles changes the tile size of the board to be the value of the slider.
func (g *Game) resizeTiles() {
	tileLengthStr := g.dom.Value(".tile-length-slider")
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
	g.dom.SetChecked("#hide-game-create", true)
	g.dom.SetChecked("#tab-game", true)
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
	rulesList := g.dom.QuerySelector(".game .rules ul")
	rulesList.Set("innerHTML", "")
	for _, r := range rules {
		clone := g.dom.CloneElement(".game .rules template")
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
	g.dom.SetChecked("#hide-final-boards", hideFinalBoards)
	playersList := g.dom.QuerySelector(".final-boards .player-list form")
	playersList.Set("innerHTML", "")
	g.finalBoards = finalBoards
	for playerName := range finalBoards {
		div := g.newFinalBoardDiv(playerName)
		playersList.Call("appendChild", div)
	}
	canvas := g.dom.QuerySelector(".final-boards .canvas canvas")
	canvas.Set("height", 0)
}

// newFinalBoardLi creates a new div to trigger drawing the board.
func (g *Game) newFinalBoardDiv(playerName string) js.Value {
	clone := g.dom.CloneElement(".final-boards .player-list template")
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
	checkedLabel := g.dom.QuerySelector(".player-list input:checked+label")
	playerName := checkedLabel.Get("innerHTML").String()
	b, ok := g.finalBoards[playerName]
	if !ok {
		g.log.Error("could not view final board for " + playerName)
		return
	}
	canvas := g.canvasCreator.Create(&b, ".final-boards .canvas")
	tileLength := g.canvas.TileLength()
	canvas.SetTileLength(tileLength)
	width := canvas.DesiredWidth()
	canvas.UpdateSize(width)
	canvas.Redraw()
}
