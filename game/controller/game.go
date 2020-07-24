// Package controller handles the logic to run the game.
package controller

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
	"github.com/jacobpatterson1549/selene-bananas/game/tile"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users.
	Game struct {
		debug                  bool
		log                    *log.Logger
		id                     game.ID
		createdAt              int64
		status                 game.Status
		userDao                *db.UserDao
		players                map[game.PlayerName]*player
		maxPlayers             int
		numNewTiles            int
		tileLetters            string
		unusedTiles            []tile.Tile
		words                  game.WordChecker
		idlePeriod             time.Duration
		shuffleUnusedTilesFunc func(tiles []tile.Tile)
		shufflePlayersFunc     func(playerNames []game.PlayerName)
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
		UserDao *db.UserDao
		// MaxPlayers is the maximum number of players that can be part of the game.
		MaxPlayers int
		// NumNewTiles is the number of new tiles each player starts the game with.
		NumNewTiles int
		// TileLetters is a string of all the upper case letters that can be used in the game.
		// If not specified, the default 144 letters will be used.
		// If a letter should occur on multiple tiles, it sh be present multiple times.
		// For example, the TileLetters "AABCCC" will be used to initialize a game with two As, 1 B, and 3 Cs.
		TileLetters string
		// Words is the WordChecker used to validate players' words when they try to finish the game.
		Words game.WordChecker
		// IdlePeroid is the amount of time that can pass between non-BoardRefresh messages before the game is idle and will delete itself.
		IdlePeriod time.Duration
		// ShuffleUnusedTilesFunc is used to shuffle unused tiles when initializing the game and after tiles are swapped.
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		// ShufflePlayersFunc is used to shuffle the order of players when giving tiles after a snag
		// The snagging player should always get a new tile.  Other players will get a tile, if possible.
		ShufflePlayersFunc func(playerNames []game.PlayerName)
	}

	// messageHandler is a function which handles game messages, returning responses to the output channel.
	messageHandler func(ctx context.Context, m game.Message, out chan<- game.Message) error
)

const (
	// gameWarningNotInProgress is a shared warning to alert users of an invalid game state.
	gameWarningNotInProgress gameWarning = "game has not started or is finished"
	// defaultTileLetters is the default if not specified
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
)

// NewGame creates a new game and runs it.
func (cfg Config) NewGame(id game.ID) (*Game, error) {
	if err := cfg.validate(id); err != nil {
		return nil, fmt.Errorf("creating game: validation: %w", err)
	}
	tileLetters := cfg.TileLetters
	if len(tileLetters) == 0 {
		tileLetters = defaultTileLetters
	}
	g := Game{
		debug:                  cfg.Debug,
		log:                    cfg.Log,
		id:                     id,
		createdAt:              cfg.TimeFunc(),
		status:                 game.NotStarted,
		userDao:                cfg.UserDao,
		maxPlayers:             cfg.MaxPlayers,
		numNewTiles:            cfg.NumNewTiles,
		tileLetters:            tileLetters,
		words:                  cfg.Words,
		idlePeriod:             cfg.IdlePeriod,
		players:                make(map[game.PlayerName]*player),
		shuffleUnusedTilesFunc: cfg.ShuffleUnusedTilesFunc,
		shufflePlayersFunc:     cfg.ShufflePlayersFunc,
	}
	if err := g.initializeUnusedTiles(); err != nil {
		return nil, err
	}
	return &g, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate(id game.ID) error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case id <= 0:
		return fmt.Errorf("positive id required")
	case cfg.UserDao == nil:
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
	g.unusedTiles = make([]tile.Tile, len(g.tileLetters))
	for i, ch := range g.tileLetters {
		id := tile.ID(i + 1)
		t, err := tile.New(id, ch)
		if err != nil {
			return fmt.Errorf("creating tile: %w", err)
		}
		g.unusedTiles[i] = *t
	}
	if g.shuffleUnusedTilesFunc != nil {
		g.shuffleUnusedTilesFunc(g.unusedTiles)
	}
	return nil
}

// Run runs the game until the context is closed.
func (g *Game) Run(ctx context.Context, removeGameFunc context.CancelFunc, in <-chan game.Message, out chan<- game.Message) {
	idleTicker := time.NewTicker(g.idlePeriod)
	active := false
	messageHandlers := map[game.MessageType]messageHandler{
		game.Join:         g.handleGameJoin,
		game.Delete:       g.handleGameDelete,
		game.StatusChange: g.handleGameStatusChange,
		game.Snag:         g.handleGameSnag,
		game.Swap:         g.handleGameSwap,
		game.TilesMoved:   g.handleGameTilesMoved,
		game.Chat:         g.handleGameChat,
		game.BoardSize:    g.handleBoardRefresh,
	}
	defer removeGameFunc()
	for { // BLOCKS
		select {
		case <-ctx.Done():
			return
		case m := <-in:
			g.handleMessage(ctx, m, out, &active, messageHandlers)
			if m.Type == game.Delete {
				return
			}
		case <-idleTicker.C:
			var m game.Message
			if !active {
				g.log.Printf("deleted game %v due to inactivity", g.id)
				g.handleGameDelete(ctx, m, out)
				return
			}
			active = false
		}
	}
}

// handleMessage handles the message with the appropriate message handler.
func (g *Game) handleMessage(ctx context.Context, m game.Message, out chan<- game.Message, active *bool, messageHandlers map[game.MessageType]messageHandler) {
	if g.debug {
		g.log.Printf("game reading message with type %v", m.Type)
	}
	var err error
	mh, ok := messageHandlers[m.Type]
	if !ok {
		err = fmt.Errorf("game does not know how to handle MessageType %v", m.Type)
	} else if _, ok := g.players[m.PlayerName]; !ok && m.Type != game.Join {
		err = fmt.Errorf("game does not have player named '%v'", m.PlayerName)
	} else {
		err = mh(ctx, m, out)
		*active = true
	}
	if err != nil {
		var mt game.MessageType
		switch err.(type) {
		case gameWarning:
			mt = game.SocketWarning
		default:
			mt = game.SocketError
			g.log.Printf("game error: %v", err)
		}
		out <- game.Message{
			Type:       mt,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
		}
	}
}

// handleGameJoin adds the player from the message to the game.
func (g *Game) handleGameJoin(ctx context.Context, m game.Message, out chan<- game.Message) (err error) {
	defer func() {
		if err != nil {
			out <- game.Message{
				Type:       game.Leave,
				PlayerName: m.PlayerName,
			}
		}
	}()
	_, ok := g.players[m.PlayerName]
	switch {
	case ok:
		return g.handleBoardRefresh(ctx, m, out)
	case g.status != game.NotStarted:
		return gameWarning("cannot join game that has been started")
	case len(g.players) >= g.maxPlayers:
		return gameWarning("no room for another player in game")
	case len(g.unusedTiles) < g.numNewTiles:
		return gameWarning("not enough tiles to join the game")
	}
	newTiles := g.unusedTiles[:g.numNewTiles]
	g.unusedTiles = g.unusedTiles[g.numNewTiles:]
	cfg := board.Config{
		NumCols: m.NumCols,
		NumRows: m.NumRows,
	}
	b, err := cfg.New(newTiles)
	if err != nil {
		return err
	}
	p := &player{
		winPoints: 10,
		Board:     *b,
	}
	g.players[m.PlayerName] = p
	gamePlayers := g.playerNames()
	out <- game.Message{
		Type:        game.Join,
		PlayerName:  m.PlayerName,
		Info:        "joining game",
		Tiles:       newTiles,
		TilesLeft:   len(g.unusedTiles),
		GamePlayers: gamePlayers,
		GameStatus:  g.status,
		GameID:      g.id,
	}
	for n := range g.players {
		if n != m.PlayerName {
			out <- game.Message{
				Type:        game.TilesChange,
				PlayerName:  n,
				Info:        fmt.Sprintf("%v joined the game", m.PlayerName),
				TilesLeft:   len(g.unusedTiles),
				GamePlayers: gamePlayers,
			}
		}
	}
	g.handleInfoChanged(out)
	return nil
}

// handleGameDelete sends game leave messages to all players in the game.
func (g *Game) handleGameDelete(ctx context.Context, m game.Message, out chan<- game.Message) error {
	for n := range g.players {
		out <- game.Message{
			Type:       game.Leave,
			PlayerName: n,
			Info:       "game deleted",
		}
	}
	return nil
}

// handleGameStatusChange changes the status of the game.
func (g *Game) handleGameStatusChange(ctx context.Context, m game.Message, out chan<- game.Message) error {
	switch g.status {
	case game.NotStarted:
		if err := g.handleGameStart(ctx, m, out); err != nil {
			return err
		}
	case game.InProgress:
		if err := g.handleGameFinish(ctx, m, out); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot change game state from %v", g.status)
	}
	g.handleInfoChanged(out)
	return nil
}

// handleGameStart starts the game.
func (g *Game) handleGameStart(ctx context.Context, m game.Message, out chan<- game.Message) error {
	if m.GameStatus != game.InProgress {
		return gameWarning("can only set game status to started")
	}
	g.status = game.InProgress
	info := fmt.Sprintf("%v started the game", m.PlayerName)
	for n := range g.players {
		out <- game.Message{
			Type:       game.StatusChange,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
			TilesLeft:  len(g.unusedTiles),
		}
	}
	return nil
}

// handleGameStart tries to finish the game for the player sending the message by checking to see if the player wins.
// If the player has won, game cleanup logic is triggered.
func (g *Game) handleGameFinish(ctx context.Context, m game.Message, out chan<- game.Message) error {
	p := g.players[m.PlayerName]
	switch {
	case m.GameStatus != game.Finished:
		return gameWarning("can only attempt to set game that is in progress to finished")
	case len(g.unusedTiles) != 0:
		return gameWarning("snag first")
	case len(p.UnusedTiles) != 0:
		p.decrementWinPoints()
		return gameWarning("not all tiles used, possible win points decremented")
	case !p.HasSingleUsedGroup():
		p.decrementWinPoints()
		return gameWarning("not all used tiles form a single group, possible win points decremented")
	}
	usedWords := p.UsedTileWords()
	var invalidWords []string
	for _, w := range usedWords {
		if !g.words.Check(w) {
			invalidWords = append(invalidWords, w)
		}
	}
	if len(invalidWords) > 0 {
		p.decrementWinPoints()
		return gameWarning(fmt.Sprintf("invalid words: %v, possible winpoints decremented", invalidWords))
	}
	g.status = game.Finished
	info := fmt.Sprintf(
		"WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point",
		m.PlayerName,
		len(usedWords),
		p.winPoints,
	)
	err := g.updateUserPoints(ctx, m.PlayerName)
	if err != nil {
		info = err.Error()
	}
	for n := range g.players {
		out <- game.Message{
			Type:       game.StatusChange,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
		}
	}
	return nil
}

// handleGameSnag adds a tile to all the players.
// The order that the players recieve their tiles is randomized, some players may not recieve tiles if there are none left.
func (g *Game) handleGameSnag(ctx context.Context, m game.Message, out chan<- game.Message) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	case len(g.unusedTiles) == 0:
		return gameWarning("no tiles left to snag, use what you have to finish")
	}
	snagPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	snagPlayerNames := make([]game.PlayerName, 1, len(g.players))
	snagPlayerNames[0] = m.PlayerName
	for n2 := range g.players {
		if m.PlayerName != n2 {
			snagPlayerNames = append(snagPlayerNames, n2)
		}
	}
	g.shufflePlayersFunc(snagPlayerNames[1:])
	for _, n2 := range snagPlayerNames {
		m2 := game.Message{
			Type:       game.TilesChange,
			PlayerName: n2,
		}
		switch {
		case n2 == m.PlayerName:
			m2.Tiles = g.unusedTiles[:1]
			m2.Info = "snagged a tile"
			if err := g.players[n2].AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		case len(g.unusedTiles) == 0:
			m2.Info = fmt.Sprintf("%v snagged a tile", m.PlayerName)
		default:
			m2.Info = fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName)
			m2.Tiles = g.unusedTiles[:1]
			if err := g.players[n2].AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		}
		snagPlayerMessages[n2] = m2
	}
	for _, m := range snagPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		out <- m
	}
	return nil
}

// handleGameSwap swaps a tile for the player for three others, if possible.
func (g *Game) handleGameSwap(ctx context.Context, m game.Message, out chan<- game.Message) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	case len(m.Tiles) != 1:
		return gameWarning("no tile specified for swap")
	case len(g.unusedTiles) == 0:
		return gameWarning("no tiles left to swap, user what you have to finish")
	}
	t := m.Tiles[0]
	p := g.players[m.PlayerName]
	err := p.RemoveTile(t)
	if err != nil {
		return err
	}
	g.unusedTiles = append(g.unusedTiles, t)
	g.shuffleUnusedTilesFunc(g.unusedTiles)
	var newTiles []tile.Tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		err := p.AddTile(g.unusedTiles[0])
		if err != nil {
			return err
		}
		g.unusedTiles = g.unusedTiles[1:]
	}
	for n := range g.players {
		m2 := game.Message{
			Type:       game.TilesChange,
			PlayerName: n,
			TilesLeft:  len(g.unusedTiles),
		}
		switch {
		case n == m.PlayerName:
			m2.Info = fmt.Sprintf("swapping %v tile", t.Ch)
			m2.Tiles = newTiles
		default:
			m2.Info = fmt.Sprintf("%v swapped a tile", m.PlayerName)
		}
		out <- m2
	}
	return nil
}

// handleGameTilesMoved updates the player's board.
func (g *Game) handleGameTilesMoved(ctx context.Context, m game.Message, out chan<- game.Message) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	}
	p := g.players[m.PlayerName]
	return p.MoveTiles(m.TilePositions)
}

// handleBoardRefresh sends the player's board back to the player.
func (g *Game) handleBoardRefresh(ctx context.Context, m game.Message, out chan<- game.Message) error {
	cfg := board.Config{
		NumCols: m.NumCols,
		NumRows: m.NumRows,
	}
	p := g.players[m.PlayerName]
	m2, err := p.refreshBoard(cfg, *g, m.PlayerName)
	if err != nil {
		return err
	}
	out <- *m2
	return nil
}

// handleGameChat sends a chat message from a player to everyone in the game.
func (g *Game) handleGameChat(ctx context.Context, m game.Message, out chan<- game.Message) error {
	info := fmt.Sprintf("%v : %v", m.PlayerName, m.Info)
	for n := range g.players {
		out <- game.Message{
			Type:       game.Chat,
			PlayerName: n,
			Info:       info,
		}
	}
	return nil
}

// updateUserPoints updates the points for users in the game after a player has won.
// The wining player gets their winPoints value, while others get a single participation point.
func (g *Game) updateUserPoints(ctx context.Context, winningPlayerName game.PlayerName) error {
	users := g.playerNames()
	userPointsIncrementFunc := func(u string) int {
		if string(u) == string(winningPlayerName) {
			p := g.players[winningPlayerName]
			return p.winPoints
		}
		return 1
	}
	return g.userDao.UpdatePointsIncrement(ctx, users, userPointsIncrementFunc)
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
func (g Game) handleInfoChanged(out chan<- game.Message) {
	i := game.Info{
		ID:        g.id,
		Status:    g.status,
		Players:   g.playerNames(),
		CreatedAt: g.createdAt,
	}
	out <- game.Message{
		Type:      game.Infos,
		GameInfos: []game.Info{i},
	}
}
