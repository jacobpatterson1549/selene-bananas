// Package game controls the logic to run the game.
package game

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
	playerController "github.com/jacobpatterson1549/selene-bananas/server/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/log"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users.
	Game struct {
		log           log.Logger
		id            game.ID
		createdAt     int64
		status        game.Status
		players       map[player.Name]*playerController.Player
		unusedTiles   []tile.Tile
		WordValidator WordValidator
		userDao       UserDao
		Config
	}

	// Config contiains the properties to create similar games.
	Config struct {
		// Debug is a flag that causes the game to log the types messages that are read.
		Debug bool
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
		// IdlePeriod is the amount of time that can pass between non-BoardRefresh messages before the game is idle and will delete itself.
		IdlePeriod time.Duration
		// ShuffleUnusedTilesFunc is used to shuffle unused tiles when initializing the game and after tiles are swapped.
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		// ShufflePlayersFunc is used to shuffle the order of players when giving tiles after a snag
		// The snagging player should always get a new tile.  Other players will get a tile, if possible.
		ShufflePlayersFunc func(playerNames []player.Name)
		// Config is the nested configuration for the specific game
		game.Config
	}

	// messageHandler is a function which handles message.Messages, returning responses to the output channel.
	messageHandler func(ctx context.Context, m message.Message, send messageSender) error

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
func (cfg Config) NewGame(log log.Logger, id game.ID, WordValidator WordValidator, userDao UserDao) (*Game, error) {
	if err := cfg.validate(log, id, WordValidator, userDao); err != nil {
		return nil, fmt.Errorf("creating game: validation: %w", err)
	}
	g := Game{
		log:           log,
		id:            id,
		createdAt:     cfg.TimeFunc(),
		status:        game.NotStarted,
		players:       make(map[player.Name]*playerController.Player),
		WordValidator: WordValidator,
		userDao:       userDao,
		Config:        cfg,
	}
	if err := g.initializeUnusedTiles(); err != nil {
		return nil, err
	}
	return &g, nil
}

// validate ensures the configuration has no errors.
// the config is modified to use the default tile letters if the tile letters are empty.
func (cfg *Config) validate(log log.Logger, id game.ID, wordValidator WordValidator, userDao UserDao) error {
	if len(cfg.TileLetters) == 0 {
		cfg.TileLetters = defaultTileLetters
	}
	switch {
	case log == nil:
		return fmt.Errorf("log required")
	case id <= 0:
		return fmt.Errorf("positive id required")
	case wordValidator == nil:
		return fmt.Errorf("word validator required")
	case userDao == nil:
		return fmt.Errorf("user dao required")
	case cfg.TimeFunc == nil:
		return fmt.Errorf("time func required")
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
	case len(cfg.TileLetters) < cfg.NumNewTiles:
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
	g.ShuffleUnusedTilesFunc(g.unusedTiles)
	return nil
}

// Run runs the game asynchronously until the context is closed.
func (g *Game) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, out chan<- message.Message) {
	idleTicker := time.NewTicker(g.IdlePeriod)
	wg.Add(1)
	go g.runSync(ctx, wg, in, out, idleTicker)
}

// runSync runs the game until the context is closed or the input channel closes.
func (g *Game) runSync(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, out chan<- message.Message, idleTicker *time.Ticker) {
	defer wg.Done()
	active := false
	send := g.sendMessage(out)
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
	for { // BLOCKING
		select {
		case <-ctx.Done():
			return
		case m, ok := <-in:
			if !ok {
				return
			}
			g.handleMessage(ctx, m, send, &active, messageHandlers)
			if m.Type == message.DeleteGame {
				return
			}
		case <-idleTicker.C:
			var m message.Message
			if !active {
				g.log.Printf("deleted game %v due to inactivity", g.id)
				g.handleGameDelete(ctx, m, send)
				return
			}
			active = false
		}
	}
}

// sendMessage creates a messageSender that adds the gameId to the message before sending it on the out channel.
func (g *Game) sendMessage(out chan<- message.Message) messageSender {
	return func(m message.Message) {
		if m.Game == nil {
			var g game.Info
			m.Game = &g
		}
		m.Game.ID = g.id
		message.Send(m, out, g.Debug, g.log)
	}
}

// handleMessage handles the message with the appropriate message handler.
func (g *Game) handleMessage(ctx context.Context, m message.Message, send messageSender, active *bool, messageHandlers map[message.Type]messageHandler) {
	if g.Debug {
		g.log.Printf("game reading message with type %v", m.Type)
	}
	err := g.handleMessageHelper(ctx, m, send, active, messageHandlers)
	if err != nil {
		var mt message.Type
		switch err.(type) {
		case gameWarning:
			mt = message.SocketWarning
		default:
			mt = message.SocketError
		}
		m2 := message.Message{
			Type:       mt,
			PlayerName: m.PlayerName,
			Game:       m.Game,
			Info:       err.Error(),
		}
		send(m2)
	}
}

// handleMessageHelper clearly handles the message, after checking the a handler exists and the player for the message is in the game.
func (g *Game) handleMessageHelper(ctx context.Context, m message.Message, send messageSender, active *bool, messageHandlers map[message.Type]messageHandler) error {
	handler, handlerExists := messageHandlers[m.Type]
	if !handlerExists {
		return fmt.Errorf("game does not know how to handle MessageType %v", m.Type)
	}
	_, playerInGame := g.players[m.PlayerName]
	if !playerInGame && m.Type != message.JoinGame {
		return fmt.Errorf("game does not have player named '%v'", m.PlayerName)
	}
	*active = true
	return handler(ctx, m, send)
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
		m2 := message.Message{
			Type:       message.LeaveGame,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
			Addr:       m.Addr,
		}
		send(m2)
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
	gamePlayers := m2.Game.Players
	for n := range g.players { // send info to other players
		if n == m.PlayerName {
			continue
		}
		m3 := message.Message{
			Type:       message.ChangeGameTiles,
			PlayerName: n,
			Info:       fmt.Sprintf("%v joined the game", m.PlayerName),
			Game: &game.Info{
				TilesLeft: len(g.unusedTiles),
				Players:   gamePlayers,
			},
		}
		send(m3)
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
	switch m.Game.Status { // what to change the status to
	case game.InProgress:
		if err := g.handleGameStart(ctx, m, send); err != nil {
			return err
		}
	case game.Finished:
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
	if g.status != game.NotStarted {
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
	var err error
	p := g.players[pn]
	switch {
	case len(p.Board.UnusedTiles) != 0:
		err = fmt.Errorf("not all tiles used")
	case !p.Board.CanBeFinished():
		err = fmt.Errorf("not all used tiles form a single group")
	case checkWords:
		usedWords, err = g.checkWords(pn)
	}
	if err != nil {
		errText := "invalid board: " + err.Error()
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
	var err error
	for _, w := range usedWords {
		if _, ok := uniqueWords[w]; g.Config.ProhibitDuplicates && ok {
			err = fmt.Errorf("duplicate words are prohibited")
			break
		}
		uniqueWords[w] = struct{}{}
		if len(w) < g.Config.MinLength {
			err = fmt.Errorf("short word detected, all must be at least %v characters", g.Config.MinLength)
			break
		}
		if !g.WordValidator.Validate(w) {
			invalidWords = append(invalidWords, w)
		}
	}
	if len(invalidWords) > 0 {
		err = fmt.Errorf("invalid words: %v", invalidWords)
	}
	if err != nil {
		return nil, err
	}
	return usedWords, nil
}

// handleGameStart tries to finish the game for the player sending the message by checking to see if the player wins.
// If the player has won, game cleanup logic is triggered.
func (g *Game) handleGameFinish(ctx context.Context, m message.Message, send messageSender) error {
	p := g.players[m.PlayerName]
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
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
		g.log.Printf("updating user points: %v", err)
		info = err.Error()
	}
	finalBoards := g.playerFinalBoards()
	for n := range g.players {
		m := message.Message{
			Type:       message.ChangeGameStatus,
			PlayerName: n,
			Info:       info,
			Game: &game.Info{
				Status:      game.Finished,
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
	if err := p.Board.RemoveTile(t); err != nil {
		return err
	}
	g.unusedTiles = append(g.unusedTiles, t)
	g.ShuffleUnusedTilesFunc(g.unusedTiles)
	var newTiles []tile.Tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		if err := p.Board.AddTile(g.unusedTiles[0]); err != nil {
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
			m2.Info = fmt.Sprintf("swapping %v tile", string(t.Ch))
			m2.Game.Board = board.New(newTiles, nil)
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
// The winning player gets their winpoints, which should be at least 2.  Other players in the game get a consolation point.
func (g *Game) updateUserPoints(ctx context.Context, winningPlayerName player.Name) error {
	userPoints := make(map[string]int, len(g.players))
	for pn, p := range g.players {
		points := 1
		if pn == winningPlayerName {
			points = p.WinPoints
		}
		userPoints[string(pn)] = points
	}
	return g.userDao.UpdatePointsIncrement(ctx, userPoints)
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
		Capacity:  g.MaxPlayers,
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
		Type:       m.Type,
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
