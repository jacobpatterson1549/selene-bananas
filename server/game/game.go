// Package game controls the logic to run the game.
package game

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/board"
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
		UserDao UserDao
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
		// WordsConfig controlls how checking words works
		WordsConfig WordsConfig
		// IdlePeroid is the amount of time that can pass between non-BoardRefresh messages before the game is idle and will delete itself.
		IdlePeriod time.Duration
		// ShuffleUnusedTilesFunc is used to shuffle unused tiles when initializing the game and after tiles are swapped.
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		// ShufflePlayersFunc is used to shuffle the order of players when giving tiles after a snag
		// The snagging player should always get a new tile.  Other players will get a tile, if possible.
		ShufflePlayersFunc func(playerNames []player.Name)
	}

	// WordsConfig is used when checking player words on a snag or game finish request.
	WordsConfig struct {
		// CheckOnSnag is a flag to check the board when a player wants to snag to ensure their board has on group of valid words.
		CheckOnSnag bool `json:"checkOnSnag,omitempty"`
		// Penalize is a flag to decrement a player's points if they try to snag a tile when their board is invalid.
		Penalize bool `json:"penalize,omitempty"`
		// MinLength is the minimum allowed word length for each word on the board.
		MinLength int `json:"minLength,omitempty"`
		// AllowDuplicates is a flag for whether or not to allow duplicate words when checking the board
		AllowDuplicates bool `json:"allowDuplicates,omitempty"`
		// FinishedAllowMove is a flag on whether or not the ui should allow tiles to be moved after the game is finished.
		FinishedAllowMove bool `json:"finishedAllowMove,omitempty"`
	}

	// messageHandler is a function which handles game messages, returning responses to the output channel.
	messageHandler func(ctx context.Context, m game.Message, out chan<- game.Message) error

	// UserDao makes changes to the stored state of users in the game
	UserDao interface {
		// UpdatePointsIncrement increments points for the specified usernames based on the userPointsIncrementFunc
		UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error
	}
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

// Run runs the game until the context is closed.
func (g *Game) Run(ctx context.Context, removeGameFunc context.CancelFunc, in <-chan game.Message, out chan<- game.Message) {
	idleTicker := time.NewTicker(g.IdlePeriod)
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
	for { // BLOCKING
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
				g.Log.Printf("deleted game %v due to inactivity", g.id)
				g.handleGameDelete(ctx, m, out)
				return
			}
			active = false
		}
	}
}

// handleMessage handles the message with the appropriate message handler.
func (g *Game) handleMessage(ctx context.Context, m game.Message, out chan<- game.Message, active *bool, messageHandlers map[game.MessageType]messageHandler) {
	if g.Debug {
		g.Log.Printf("game reading message with type %v", m.Type)
	}
	var err error
	if mh, ok := messageHandlers[m.Type]; !ok {
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
			g.Log.Printf("game error: %v", err)
		}
		out <- game.Message{
			Type:       mt,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
		}
	}
}

// handleGameJoin adds the player from the message to the game.
func (g *Game) handleGameJoin(ctx context.Context, m game.Message, out chan<- game.Message) error {
	_, ok := g.players[m.PlayerName]
	var err error
	switch {
	case ok:
		err = g.handleBoardRefresh(ctx, m, out)
	case g.status != game.NotStarted:
		err = gameWarning("cannot join game that has been started")
	case len(g.players) >= g.MaxPlayers:
		err = gameWarning("no room for another player in game")
	case len(g.unusedTiles) < g.NumNewTiles:
		err = gameWarning("not enough tiles to join the game")
	default:
		err = g.handleAddPlayer(ctx, m, out)
	}
	if err != nil {
		// kick the player here, returning an error will not remove them from the game
		out <- game.Message{
			Type:       game.Leave,
			PlayerName: m.PlayerName,
		}
		return err
	}
	return nil
}

// handleAddPlayer adds the player to the game.
func (g *Game) handleAddPlayer(ctx context.Context, m game.Message, out chan<- game.Message) error {
	newTiles := g.unusedTiles[:g.NumNewTiles]
	g.unusedTiles = g.unusedTiles[g.NumNewTiles:]
	boardCfg := board.Config{
		NumCols: m.NumCols,
		NumRows: m.NumRows,
	}
	b, err := boardCfg.New(newTiles)
	if err != nil {
		return err
	}
	p, err := g.PlayerCfg.New(b)
	if err != nil {
		return fmt.Errorf("creating player: %w", err)
	}
	g.players[m.PlayerName] = p
	m2, err := b.Resize(boardCfg)
	if err != nil {
		return fmt.Errorf("creating board message: %w", err)
	}
	gamePlayers := g.playerNames()
	m2.Type = game.Join
	m2.PlayerName = m.PlayerName
	m2.Info = "joining game"
	m2.TilesLeft = len(g.unusedTiles)
	m2.GamePlayers = gamePlayers
	m2.GameStatus = g.status
	m2.GameID = g.id
	m2.GameRules = g.Rules()
	out <- *m2
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
		uniqueWords := make(map[string]struct{}, len(usedWords))
		usedWords = p.Board.UsedTileWords()
		var invalidWords []string
		for _, w := range usedWords {
			if _, ok := uniqueWords[w]; !g.WordsConfig.AllowDuplicates && ok {
				errText = "duplicate words detected"
				break
			}
			uniqueWords[w] = struct{}{}
			if len(w) < g.WordsConfig.MinLength {
				errText = fmt.Sprintf("short word detected, all must be at least %v characters", g.WordsConfig.MinLength)
				break
			}
			if !g.WordChecker.Check(w) {
				invalidWords = append(invalidWords, w)
			}
		}
		if len(invalidWords) > 0 {
			errText = fmt.Sprintf("invalid words: %v", invalidWords)
		}
	}
	if len(errText) != 0 {
		errText = "invalid board: " + errText
		if g.WordsConfig.Penalize {
			p.WinPoints--
			errText = errText + ", possible win points decremented"
		}
		return nil, gameWarning(errText)
	}
	return usedWords, nil
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
	}
	usedWords, boardErr := g.checkPlayerBoard(m.PlayerName, true)
	if boardErr != nil {
		return boardErr
	}
	g.status = game.Finished
	info := fmt.Sprintf(
		"WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point",
		m.PlayerName,
		len(usedWords),
		p.WinPoints,
	)
	err := g.updateUserPoints(ctx, m.PlayerName)
	if err != nil {
		info = err.Error()
	}
	messageStatus := game.Finished
	if g.WordsConfig.FinishedAllowMove {
		messageStatus = game.FinishedAllowMove
	}
	for n := range g.players {
		out <- game.Message{
			Type:       game.StatusChange,
			PlayerName: n,
			Info:       info,
			GameStatus: messageStatus,
		}
	}
	return nil
}

// handleGameSnag adds a tile to all the players.
// The order that the players receive their tiles is randomized, some players may not receive tiles if there are none left.
func (g *Game) handleGameSnag(ctx context.Context, m game.Message, out chan<- game.Message) error {
	switch {
	case g.status != game.InProgress:
		return gameWarningNotInProgress
	case len(g.unusedTiles) == 0:
		return gameWarning("no tiles left to snag, use what you have to finish")
	}
	if _, err := g.checkPlayerBoard(m.PlayerName, g.WordsConfig.CheckOnSnag); err != nil {
		return err
	}
	snagPlayerMessages := make(map[player.Name]game.Message, len(g.players))
	snagPlayerNames := make([]player.Name, 1, len(g.players))
	snagPlayerNames[0] = m.PlayerName
	for n2 := range g.players {
		if m.PlayerName != n2 {
			snagPlayerNames = append(snagPlayerNames, n2)
		}
	}
	g.ShufflePlayersFunc(snagPlayerNames[1:])
	for _, n2 := range snagPlayerNames {
		m2 := game.Message{
			Type:       game.TilesChange,
			PlayerName: n2,
		}
		switch {
		case n2 == m.PlayerName:
			m2.Tiles = g.unusedTiles[:1]
			m2.Info = "snagged a tile"
			if err := g.players[n2].Board.AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		case len(g.unusedTiles) == 0:
			m2.Info = fmt.Sprintf("%v snagged a tile", m.PlayerName)
		default:
			m2.Info = fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName)
			m2.Tiles = g.unusedTiles[:1]
			if err := g.players[n2].Board.AddTile(g.unusedTiles[0]); err != nil {
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
	return p.Board.MoveTiles(m.TilePositions)
}

// handleBoardRefresh sends the player's board back to the player.
func (g *Game) handleBoardRefresh(ctx context.Context, m game.Message, out chan<- game.Message) error {
	cfg := board.Config{
		NumCols: m.NumCols,
		NumRows: m.NumRows,
	}
	p := g.players[m.PlayerName]
	m2, err := p.Board.Resize(cfg)
	if err != nil {
		return err
	}
	m2.Type = game.Join
	m2.PlayerName = m.PlayerName
	m2.TilesLeft = len(g.unusedTiles)
	m2.GameStatus = g.status
	m2.GamePlayers = g.playerNames()
	m2.GameID = g.id
	m2.GameRules = g.Rules()
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
// The winning player gets their winpoints or at least 2 points.  Other players in the game get a consolation point.
func (g *Game) updateUserPoints(ctx context.Context, winningPlayerName player.Name) error {
	userPoints := make(map[string]int, len(g.players))
	for pn, p := range g.players {
		points := 1
		if pn == winningPlayerName {
			switch {
			case p.WinPoints > 1:
				points = p.WinPoints
			default:
				points = 2
			}
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

// Rules creates a list of the shared rules for any game.
func Rules() []string {
	return []string{
		"Create or join a game from the Lobby after refreshing the games list.",
		"Any player can join a game that is not started, but active games can only be joined by players who started in them.",
		"After all players have joined the game, click the Start button to start the game.",
		"Arrange unused tiles in the game area form vertical and horizontal English words.",
		"Click the Snag button to get a new tile if all tiles are used in words. This also gives other players a new tile.",
		"Click the Swap button and then a tile to exchange it for three others.",
		"Click the Finish button to run the scoring function when there are no tiles left to use.  The scoring function determines if all of the player's tiles are used and form a continuous block of English words.  If successful, the player wins. Otherwise, the player's potential winning score is decremented and play continues.",
	}
}

// Rules creates a list of rules for the specific game.
func (g Game) Rules() []string {
	rules := Rules()
	if g.WordsConfig.CheckOnSnag {
		rules = append(rules, "Words are checked to be valid when a player tries to snag a new letter.")
	}
	if g.WordsConfig.Penalize {
		rules = append(rules, "If a player tries to snag unsuccessfully, the amount potential of win points is decremented")
	}
	if g.WordsConfig.MinLength > 2 {
		rules = append(rules, fmt.Sprintf("All words must be at least %d letters long", g.WordsConfig.MinLength))
	}
	if !g.WordsConfig.AllowDuplicates {
		rules = append(rules, "Duplicate words are not allowed.")
	}
	if g.WordsConfig.FinishedAllowMove {
		rules = append(rules, "Tiles can be moved when game is finished.")
	}
	return rules
}
