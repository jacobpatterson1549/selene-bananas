// Package game controls the logic to run the game.
package game

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	"github.com/jacobpatterson1549/selene-bananas/game/word"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users.
	Game struct {
		id          game.ID
		createdAt   int64
		status      game.Status
		players     map[player.Name]*playerController.Player
		unusedTiles []tile.Tile
		UserDao
		Config
	}

	// Config contiains the properties to create similar games.
	Config struct {
		// Debug is a flag that causes the game to log the types messages that are read.
		Debug bool
		// Log is used to log errors and other information.
		Log *log.Logger
		// TimeFunc is a function which should supply the current time since the unix epoch.
		// Used for the created at timestamp.
		TimeFunc func() int64
		// UserDao is used to increment the points for players when the game is finished.
		// MaxPlayers is the maximum number of players that can be part of the game.
		MaxPlayers int
		// PlayerCfg is used to create new players.
		PlayerCfg playerController.Config
		// NumNewTiles is the number of new tiles each player starts the game with.
		NumNewTiles int
		// TileLetters is a string of all the upper case letters that can be used in the game.
		// If not specified, the default 144 letters will be used.
		// If a letter should occur on multiple tiles, it sh be present multiple times.
		// For example, the TileLetters "AABCCC" will be used to initialize a game with two As, 1 B, and 3 Cs.
		TileLetters string
		// WordChecker is used to validate players' words when they try to finish the game.
		WordChecker word.Checker
		// IdlePeriod is the amount of time that can pass between non-BoardRefresh messages before the game is idle and will delete itself.
		IdlePeriod time.Duration
		// ShuffleUnusedTilesFunc is used to shuffle unused tiles when initializing the game and after tiles are swapped.
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		// ShufflePlayersFunc is used to shuffle the order of players when giving tiles after a snag
		// The snagging player should always get a new tile.  Other players will get a tile, if possible.
		ShufflePlayersFunc func(playerNames []player.Name)
		// Config is the nested coniguration for the specific game
		game.Config
	}

	// messageHandler is a function which handles message.Messages, returning responses to the output channel.
	messageHandler func(ctx context.Context, m message.Message, send messageSender) error

	// UserDao makes changes to the stored state of users in the game
	UserDao interface {
		// UpdatePointsIncrement increments points for the specified usernames based on the userPointsIncrementFunc
		UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error
	}

	// messageSender is a function that sends a message somewhere.
	messageSender func(m message.Message)
)

const (
	// gameWarningNotInProgress is a shared warning to alert users of an invalid game state.
	gameWarningNotInProgress gameWarning = "game has not started or is finished"
	// defaultTileLetters is the default if not specified
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
)

// NewGame creates a new game and runs it.
func (cfg Config) NewGame(id game.ID, ud UserDao) (*Game, error) {
	if err := cfg.validate(id, ud); err != nil {
		return nil, fmt.Errorf("creating game: validation: %w", err)
	}
	if len(cfg.TileLetters) == 0 {
		cfg.TileLetters = defaultTileLetters
	}
	g := Game{
		id:        id,
		createdAt: cfg.TimeFunc(),
		status:    game.NotStarted,
		players:   make(map[player.Name]*playerController.Player),
		Config:    cfg,
	}
	if err := g.initializeUnusedTiles(); err != nil {
		return nil, err
	}
	return &g, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(id game.ID, ud UserDao) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case id <= 0:
		return fmt.Errorf("positive id required")
	case cfg.TimeFunc == nil:
		return fmt.Errorf("time func required required")
	case ud == nil:
		return fmt.Errorf("user dao required")
	case cfg.MaxPlayers <= 0:
		return fmt.Errorf("positive max player count required")
	case cfg.NumNewTiles <= 0:
		return fmt.Errorf("positive number of player starting tile count required")
	case cfg.IdlePeriod <= 0:
		return fmt.Errorf("positive idle period required")
	case cfg.ShuffleUnusedTilesFunc == nil:
		return fmt.Errorf("function to shuffle tiles required")
	case cfg.ShufflePlayersFunc == nil:
		return fmt.Errorf("function to shuffle player draw order required")
	case (len(cfg.TileLetters) != 0 && len(cfg.TileLetters) < cfg.NumNewTiles) || len(defaultTileLetters) < cfg.NumNewTiles:
		return fmt.Errorf("not enough tiles for a single player to join the game")
	}
	return nil
}

// initialize unusedTiles from tileLetters or defaultTileLetters and shuffles them.
func (g *Game) initializeUnusedTiles() error {
	g.unusedTiles = make([]tile.Tile, len(g.TileLetters))
	for i, ch := range g.TileLetters {
		id := tile.ID(i + 1)
		t, err := tile.New(id, ch)
		if err != nil {
			return fmt.Errorf("creating tile: %w", err)
		}
		g.unusedTiles[i] = *t
	}
	if g.ShuffleUnusedTilesFunc != nil {
		g.ShuffleUnusedTilesFunc(g.unusedTiles)
	}
	return nil
}

// Run runs the game asynchronously until the context is closed.
func (g *Game) Run(ctx context.Context, in <-chan message.Message, out chan<- message.Message) {
	idleTicker := time.NewTicker(g.IdlePeriod)
	active := false
	messageSender := g.sendMessage(out)
	messageHandlers := map[message.Type]messageHandler{
		message.JoinGame:         g.handleGameJoin,
		message.DeleteGame:       g.handleGameDelete,
		message.ChangeGameStatus: g.handleGameStatusChange,
		message.SnagGameTile:     g.handleGameSnag,
		message.SwapGameTile:     g.handleGameSwap,
		message.MoveGameTile:     g.handleGameTilesMoved,
		message.GameChat:         g.handleGameChat,
		message.RefreshGameBoard: g.handleBoardRefresh,
	}
	go func() {
		for { // BLOCKING
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				g.handleMessage(ctx, m, messageSender, &active, messageHandlers)
				if m.Type == message.DeleteGame {
					return
				}
			case <-idleTicker.C:
				var m message.Message
				if !active {
					g.Log.Printf("deleted game %v due to inactivity", g.id)
					g.handleGameDelete(ctx, m, messageSender)
					return
				}
				active = false
			}
		}
	}()
}

// sendMessage creates a messageSender that adds the gameId to the message before sending it on the out channel.
func (g *Game) sendMessage(out chan<- message.Message) messageSender {
	return func(m message.Message) {
		if m.Game == nil {
			var g game.Info
			m.Game = &g
		}
		m.Game.ID = g.id
		out <- m
	}
}

// handleMessage handles the message with the appropriate message handler.
func (g *Game) handleMessage(ctx context.Context, m message.Message, send messageSender, active *bool, messageHandlers map[message.Type]messageHandler) {
	if g.Debug {
		g.Log.Printf("game reading message with type %v", m.Type)
	}
	var err error
	if mh, ok := messageHandlers[m.Type]; !ok {
		err = fmt.Errorf("game does not know how to handle MessageType %v", m.Type)
	} else if _, ok := g.players[m.PlayerName]; !ok && m.Type != message.JoinGame {
		err = fmt.Errorf("game does not have player named '%v'", m.PlayerName)
	} else {
		err = mh(ctx, m, send)
		*active = true
	}
	if err != nil {
		var mt message.Type
		switch err.(type) {
		case gameWarning:
			mt = message.SocketWarning
		default:
			mt = message.SocketError
			g.Log.Printf("game error: %v", err)
		}
		m := message.Message{
			Type:       mt,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
		}
		send(m)
	}
}

// handleGameJoin adds the player from the message to the game.
func (g *Game) handleGameJoin(ctx context.Context, m message.Message, send messageSender) error {
	_, ok := g.players[m.PlayerName]
	var err error
	switch {
	case ok:
		err = g.handleBoardRefresh(ctx, m, send)
	case g.status != game.NotStarted:
		err = gameWarning("cannot join game that has been started")
	case len(g.players) >= g.MaxPlayers:
		err = gameWarning("no room for another player in game")
	case len(g.unusedTiles) < g.NumNewTiles:
		err = gameWarning("not enough tiles to join the game")
	default:
		err = g.handleAddPlayer(ctx, m, send)
	}
	if err != nil {
		// kick the player here, returning an error will not remove them from the game
		m := message.Message{
			Type:       message.LeaveGame,
			PlayerName: m.PlayerName,
		}
		send(m)
		return err
	}
	return nil
}

// handleAddPlayer adds the player to the game.
func (g *Game) handleAddPlayer(ctx context.Context, m message.Message, send messageSender) error {
	newTiles := g.unusedTiles[:g.NumNewTiles]
	g.unusedTiles = g.unusedTiles[g.NumNewTiles:]
	b, err := m.Game.Board.Config.New(newTiles)
	if err != nil {
		return err
	}
	p, err := g.PlayerCfg.New(b)
	if err != nil {
		return fmt.Errorf("creating player: %w", err)
	}
	g.players[m.PlayerName] = p
	m2, err := g.resizeBoard(m)
	if err != nil {
		return fmt.Errorf("creating board message: %w", err)
	}
	m2.Info = "joining game"
	send(*m2)
	gamePlayers := g.playerNames() // also called in g.ResizeBoard
	for n := range g.players {
		if n != m.PlayerName {
			m := message.Message{
				Type:       message.ChangeGameTiles,
				PlayerName: n,
				Info:       fmt.Sprintf("%v joined the game", m.PlayerName),
				Game: &game.Info{
					TilesLeft: len(g.unusedTiles),
					Players:   gamePlayers,
				},
			}
			send(m)
		}
	}
	g.handleInfoChanged(send)
	return nil
}

// handleGameDelete sends game leave messages to all players in the game.
func (g *Game) handleGameDelete(ctx context.Context, m message.Message, send messageSender) error {
	for n := range g.players {
		m := message.Message{
			Type:       message.LeaveGame,
			PlayerName: n,
			Info:       "game deleted",
		}
		send(m)
	}
	g.status = game.Deleted
	g.handleInfoChanged(send)
	return nil
}

// handleGameStatusChange changes the status of the game.
func (g *Game) handleGameStatusChange(ctx context.Context, m message.Message, send messageSender) error {
	switch g.status {
	case game.NotStarted:
		if err := g.handleGameStart(ctx, m, send); err != nil {
			return err
		}
	case game.InProgress:
		if err := g.handleGameFinish(ctx, m, send); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot change game state from %v", g.status)
	}
	g.handleInfoChanged(send)
	return nil
}

// handleGameStart starts the game.
func (g *Game) handleGameStart(ctx context.Context, m message.Message, send messageSender) error {
	if m.Game.Status != game.InProgress {
		return gameWarning("can only set game status to started")
	}
	g.status = game.InProgress
	info := fmt.Sprintf("%v started the game", m.PlayerName)
	for n := range g.players {
		m := message.Message{
			Type:       message.ChangeGameStatus,
			PlayerName: n,
			Info:       info,
			Game: &game.Info{
				Status:    g.status,
				TilesLeft: len(g.unusedTiles),
			},
		}
		send(m)
	}
	return nil
}

// checkPlayerBoard checks the player board to ensure all tiles are in a group, words are valid, and other tests prescribed by the config.
// The words and a possible game warning error are returned.
// The player's winPoints are decremented if an error is returned if and only if the config wants to.
func (g *Game) checkPlayerBoard(pn player.Name, checkWords bool) ([]string, error) {
	var usedWords []string
	errText := ""
	p := g.players[pn]
	switch {
	case len(p.Board.UnusedTiles) != 0:
		errText = "not all tiles used"
	case !p.Board.HasSingleUsedGroup():
		errText = "not all used tiles form a single group"
	case checkWords:
		var err error
		usedWords, err = g.checkWords(pn)
		if err != nil {
			errText = err.Error()
		}
	}
	if len(errText) != 0 {
		errText = "invalid board: " + errText
		if g.Config.Penalize && p.WinPoints > 2 {
			p.WinPoints--
			errText = errText + ", possible win points decremented"
		}
		return nil, gameWarning(errText)
	}
	return usedWords, nil
}

// checkWords returns the used words from the game and an error if the game board is not valid.
func (g Game) checkWords(pn player.Name) ([]string, error) {
	p := g.players[pn]
	usedWords := p.Board.UsedTileWords()
	var invalidWords []string
	uniqueWords := make(map[string]struct{}, len(usedWords))
	errText := ""
	for _, w := range usedWords {
		if _, ok := uniqueWords[w]; !g.Config.AllowDuplicates && ok {
			errText = "duplicate words detected"
			break
		}
		uniqueWords[w] = struct{}{}
		if len(w) < g.Config.MinLength {
			errText = fmt.Sprintf("short word detected, all must be at least %v characters", g.Config.MinLength)
			break
		}
		if !g.WordChecker.Check(w) {
			invalidWords = append(invalidWords, w)
		}
	}
	if len(invalidWords) > 0 { // len(errText) == 0
		errText = fmt.Sprintf("invalid words: %v", invalidWords)
	}
	if len(errText) != 0 {
		return usedWords, errors.New(errText)
	}
	return usedWords, nil
}

// handleGameStart tries to finish the game for the player sending the message by checking to see if the player wins.
// If the player has won, game cleanup logic is triggered.
func (g *Game) handleGameFinish(ctx context.Context, m message.Message, send messageSender) error {
	p := g.players[m.PlayerName]
	switch {
	case m.Game.Status != game.Finished:
		return gameWarning("can only attempt to set game that is in progress to finished")
	case len(g.unusedTiles) != 0:
		return gameWarning("snag first")
	}
	usedWords, boardErr := g.checkPlayerBoard(m.PlayerName, true)
	if boardErr != nil {
		return boardErr
	}
	g.status = game.Finished
	info := fmt.Sprintf(
		"WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point.  View other player's boards on the 'Final Boards' tab,",
		m.PlayerName,
		len(usedWords),
		p.WinPoints,
	)
	err := g.updateUserPoints(ctx, m.PlayerName)
	if err != nil {
		info = err.Error()
	}
	messageStatus := game.Finished
	finalBoards := g.playerFinalBoards()
	for n := range g.players {
		m := message.Message{
			Type:       message.ChangeGameStatus,
			PlayerName: n,
			Info:       info,
			Game: &game.Info{
				Status:      messageStatus,
				FinalBoards: finalBoards,
			},
		}
		send(m)
	}
	return nil
}

// handleGameSnag adds a tile to all the players.
// The order that the players receive their tiles is randomized, some players may not receive tiles if there are none left.
func (g *Game) handleGameSnag(ctx context.Context, m message.Message, send messageSender) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	case len(g.unusedTiles) == 0:
		return gameWarning("no tiles left to snag, use what you have to finish")
	}
	if _, err := g.checkPlayerBoard(m.PlayerName, g.Config.CheckOnSnag); err != nil {
		return err
	}
	snagPlayerMessages := make(map[player.Name]message.Message, len(g.players))
	snagPlayerNames := make([]player.Name, 1, len(g.players))
	snagPlayerNames[0] = m.PlayerName
	for n2 := range g.players {
		if m.PlayerName != n2 {
			snagPlayerNames = append(snagPlayerNames, n2)
		}
	}
	g.ShufflePlayersFunc(snagPlayerNames[1:])
	for _, n2 := range snagPlayerNames {
		m2 := message.Message{
			Type:       message.ChangeGameTiles,
			PlayerName: n2,
		}
		var tiles []tile.Tile
		switch {
		case n2 == m.PlayerName:
			tiles = g.unusedTiles[:1]
			m2.Info = "snagged a tile"
			if err := g.players[n2].Board.AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		case len(g.unusedTiles) == 0:
			m2.Info = fmt.Sprintf("%v snagged a tile", m.PlayerName)
		default:
			m2.Info = fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName)
			tiles = g.unusedTiles[:1]
			if err := g.players[n2].Board.AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		}
		m2.Game = &game.Info{
			Board: board.New(tiles, nil),
		}
		snagPlayerMessages[n2] = m2
	}
	for _, m := range snagPlayerMessages {
		m.Game.TilesLeft = len(g.unusedTiles)
		send(m)
	}
	return nil
}

// handleGameSwap swaps a tile for the player for three others, if possible.
func (g *Game) handleGameSwap(ctx context.Context, m message.Message, send messageSender) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	case len(m.Game.Board.UnusedTiles) != 1:
		return gameWarning("no tile specified for swap")
	case len(g.unusedTiles) == 0:
		return gameWarning("no tiles left to swap, user what you have to finish")
	}
	tid := m.Game.Board.UnusedTileIDs[0]
	t := m.Game.Board.UnusedTiles[tid]
	p := g.players[m.PlayerName]
	err := p.Board.RemoveTile(t)
	if err != nil {
		return err
	}
	g.unusedTiles = append(g.unusedTiles, t)
	g.ShuffleUnusedTilesFunc(g.unusedTiles)
	var newTiles []tile.Tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		err := p.Board.AddTile(g.unusedTiles[0])
		if err != nil {
			return err
		}
		g.unusedTiles = g.unusedTiles[1:]
	}
	for n := range g.players {
		m2 := message.Message{
			Type:       message.ChangeGameTiles,
			PlayerName: n,
			Game: &game.Info{
				TilesLeft: len(g.unusedTiles),
			},
		}
		switch {
		case n == m.PlayerName:
			m2.Info = fmt.Sprintf("swapping %v tile", t.Ch)
			m2.Game = &game.Info{
				Board: board.New(newTiles, nil),
			}
		default:
			m2.Info = fmt.Sprintf("%v swapped a tile", m.PlayerName)
		}
		send(m2)
	}
	return nil
}

// handleGameTilesMoved updates the player's board.
func (g *Game) handleGameTilesMoved(ctx context.Context, m message.Message, send messageSender) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	}
	p := g.players[m.PlayerName]
	return p.Board.MoveTiles(m.Game.Board.UsedTiles)
}

// handleBoardRefresh sends the player's board back to the player.
func (g *Game) handleBoardRefresh(ctx context.Context, m message.Message, send messageSender) error {
	m2, err := g.resizeBoard(m)
	if err != nil {
		return err
	}
	send(*m2)
	return nil
}

// handleGameChat sends a chat message from a player to everyone in the game.
func (g *Game) handleGameChat(ctx context.Context, m message.Message, send messageSender) error {
	info := fmt.Sprintf("%v : %v", m.PlayerName, m.Info)
	for n := range g.players {
		m2 := message.Message{
			Type:       message.GameChat,
			PlayerName: n,
			Info:       info,
		}
		send(m2)
	}
	return nil
}

// updateUserPoints updates the points for users in the game after a player has won.
// The winning player gets their winpoints, whould should be at least 2.  Other players in the game get a consolation point.
func (g *Game) updateUserPoints(ctx context.Context, winningPlayerName player.Name) error {
	userPoints := make(map[string]int, len(g.players))
	for pn, p := range g.players {
		points := 1
		if pn == winningPlayerName {
			points = p.WinPoints
		}
		userPoints[string(pn)] = points
	}
	return g.UserDao.UpdatePointsIncrement(ctx, userPoints)
}

// playerNames returns an array of the player name strings.
func (g Game) playerNames() []string {
	playerNames := make([]string, 0, len(g.players))
	for n := range g.players {
		playerNames = append(playerNames, string(n))
	}
	sort.Strings(playerNames)
	return playerNames
}

// handleInfoChanged sends the game's info in a message.
func (g Game) handleInfoChanged(send messageSender) {
	i := game.Info{
		ID:        g.id,
		Status:    g.status,
		Players:   g.playerNames(),
		CreatedAt: g.createdAt,
	}
	m := message.Message{
		Type: message.GameInfos,
		Game: &i,
	}
	send(m)
}

// resizeBoard refreshes the board for the specified player using the config.
func (g *Game) resizeBoard(m message.Message) (*message.Message, error) {
	p := g.players[m.PlayerName]
	b := p.Board
	rr, err := b.Resize(m.Game.Board.Config)
	if err != nil {
		return nil, err
	}
	m2 := message.Message{
		Info:       rr.Info,
		Type:       message.JoinGame,
		PlayerName: m.PlayerName,
		Game: &game.Info{
			Board:     board.New(rr.Tiles, rr.TilePositions),
			TilesLeft: len(g.unusedTiles),
			Status:    g.status,
			Players:   g.playerNames(),
			ID:        g.id,
			Config:    &g.Config.Config,
		},
		Addr: m.Addr,
	}
	if g.status == game.Finished {
		m2.Game.FinalBoards = g.playerFinalBoards()
	}
	return &m2, nil
}

// playerFinalBoards creates a map of player boards.
func (g Game) playerFinalBoards() map[string]board.Board {
	finalBoards := make(map[string]board.Board, len(g.players))
	for pn, p := range g.players {
		finalBoards[string(pn)] = *p.Board
	}
	return finalBoards
}
