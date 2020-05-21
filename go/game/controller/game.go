package controller

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users
	Game struct {
		debug                  bool
		log                    *log.Logger
		id                     game.ID
		createdAt              string
		status                 game.Status
		userDao                db.UserDao
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

	// Config contiains the properties to create similar games
	Config struct {
		// Debug is a flag that causes the game to log the types messages that are read
		Debug bool
		// Log is used to log errors and other information
		Log *log.Logger
		// UserDao is used to increment the points for players when the game is finished
		UserDao db.UserDao
		// MaxPlayers is the maximum number of players that can be part of the game
		MaxPlayers int
		// NumNewTiles is the number of new tiles each player starts the game with
		NumNewTiles int
		// TileLetters is a string of all the upper case letters that can be used in the game
		// If not specified, the default 144 letters will be used.
		// If a letter should occur on multiple tiles, it sh be present multiple times.
		// For example, the TileLetters "AABCCC" will be used to initialize a game with two As, 1 B, and 3 Cs.
		TileLetters string
		// Words is the WordChecker used to validate players' words when they try to finish the game
		Words game.WordChecker
		// IdlePeroid is the amount of time that can pass between non-BoardRefresh messages before the game is idle and will delete itself
		IdlePeriod time.Duration
		// ShuffleUnusedTilesFunc is used to shuffle unused tiles when initializing the game and after tiles are swapped
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		// ShufflePlayersFunc is used to shuffle the order of players when giving tiles after a snag
		// The snagging player should always get a new tile.  Other players will get a tile, if possible.
		ShufflePlayersFunc func(playerNames []game.PlayerName)
	}
)

const (
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
)

// NewGame creates a new game and runs it
func (cfg Config) NewGame(id game.ID) (*Game, error) {
	if err := cfg.validate(id); err != nil {
		return nil, err
	}
	tileLetters := cfg.TileLetters
	if len(tileLetters) == 0 {
		tileLetters = defaultTileLetters
	}
	g := Game{
		debug:                  cfg.Debug,
		log:                    cfg.Log,
		id:                     id,
		createdAt:              time.Now().Format(time.UnixDate),
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
	}
	return nil
}

// initialize unusedTiles from tileLetters or defaultTileLetters and shuffles them
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

// Run runs the game
func (g *Game) Run(done <-chan struct{}, in <-chan game.Message, out chan<- game.Message) {
	idleTicker := time.NewTicker(g.idlePeriod)
	active := false
	messageHandlers := map[game.MessageType]func(game.Message, chan<- game.Message) error{
		game.Join:         g.handleGameJoin,
		game.Delete:       g.handleGameDelete,
		game.StatusChange: g.handleGameStatusChange,
		game.Snag:         g.handleGameSnag,
		game.Swap:         g.handleGameSwap,
		game.TilesMoved:   g.handleGameTilesMoved,
		game.BoardRefresh: g.handleBoardRefresh,
		game.Infos:        g.handleGameInfos,
		game.Chat:         g.handleGameChat,
	}
	go func() {
		for {
			select {
			case <-done:
				return
			case m := <-in:
				if g.debug {
					g.log.Printf("game reading message with type %v", m.Type)
				}
				var err error
				mh, ok := messageHandlers[m.Type]
				if !ok {
					err = fmt.Errorf("game does not know how to handle MessageType %v", m.Type)
				} else if _, ok := g.players[m.PlayerName]; !ok && m.Type != game.Join && m.Type != game.Infos {
					err = fmt.Errorf("game does not have player named '%v'", m.PlayerName)
				} else {
					if m.Type != game.BoardRefresh {
						active = true
					}
					err = mh(m, out)
				}
				if err != nil {
					g.log.Printf("game error: %v", err)
					var mt game.MessageType
					switch err.(type) {
					case gameWarning:
						mt = game.SocketWarning
					default:
						mt = game.SocketError
					}
					out <- game.Message{
						Type:       mt,
						PlayerName: m.PlayerName,
						Info:       err.Error(),
					}
				}
			case <-idleTicker.C:
				if !active {
					var m game.Message
					g.log.Printf("deleted game %v due to inactivity", g.id)
					g.handleGameDelete(m, out)
				}
				active = false
			}
		}
	}()
}

func (g *Game) handleGameJoin(m game.Message, out chan<- game.Message) error {
	if _, ok := g.players[m.PlayerName]; ok {
		return g.handleBoardRefresh(m, out)
	}
	if len(g.players) >= g.maxPlayers {
		return gameWarning("no room for another player in game")
	}
	if len(g.unusedTiles) < g.numNewTiles {
		// TODO: better handling of canJoinGame -> should there be a channel for this? should kick player...
		return gameWarning("deleting game because it can not start: there are not enough tiles")
	}
	newTiles := g.unusedTiles[:g.numNewTiles]
	g.unusedTiles = g.unusedTiles[g.numNewTiles:]
	b := board.New(newTiles)
	p := &player{
		winPoints: 10,
		Board:     b,
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
				Type:        game.SocketInfo,
				PlayerName:  n,
				Info:        fmt.Sprintf("%v joined the game", m.PlayerName),
				TilesLeft:   len(g.unusedTiles),
				GamePlayers: gamePlayers,
			}
		}
	}
	return nil
}

func (g *Game) handleGameDelete(m game.Message, out chan<- game.Message) error {
	out <- game.Message{
		Type:   game.Delete,
		GameID: g.id,
	}
	for n := range g.players {
		out <- game.Message{
			Type:       game.Delete,
			PlayerName: n,
			Info:       "game deleted",
		}
	}
	return nil
}

func (g *Game) handleGameStatusChange(m game.Message, out chan<- game.Message) error {
	switch g.status {
	case game.NotStarted:
		if m.GameStatus != game.InProgress {
			return gameWarning("game already started or is finished")
		}
		return g.start(m.PlayerName, out)
	case game.InProgress:
		if m.GameStatus != game.Finished {
			return gameWarning("can only attempt to set game that is in progress to finished")
		}
		return g.finish(m.PlayerName, out)
	}
	return fmt.Errorf("cannot change game state from %v", g.status)
}

func (g *Game) start(startingPlayerName game.PlayerName, out chan<- game.Message) error {
	g.status = game.InProgress
	info := fmt.Sprintf("%v started the game", startingPlayerName)
	for n := range g.players {
		out <- game.Message{
			Type:       game.SocketInfo,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
			TilesLeft:  len(g.unusedTiles),
		}
	}
	return nil
}

func (g *Game) finish(finishingPlayerName game.PlayerName, out chan<- game.Message) error {
	p := g.players[finishingPlayerName]
	if len(g.unusedTiles) != 0 {
		return gameWarning("snag first")
	}
	if len(p.UnusedTiles) != 0 {
		p.decrementWinPoints()
		return gameWarning("not all tiles used, possible win points decremented")
	}
	if !p.HasSingleUsedGroup() {
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
		finishingPlayerName,
		len(usedWords),
		p.winPoints,
	)
	err := g.updateUserPoints(finishingPlayerName)
	if err != nil {
		info = err.Error()
	}
	for n := range g.players {
		out <- game.Message{
			Type:       game.SocketInfo,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
			TilesLeft:  len(g.unusedTiles),
		}
	}
	return nil
}

func (g *Game) handleGameSnag(m game.Message, out chan<- game.Message) error {
	if g.status != game.InProgress {
		return gameWarning("game has not started or is finished")
	}
	if len(g.unusedTiles) == 0 {
		return gameWarning("no tiles left to snag, use what you have to finish")
	}
	snagPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	snagPlayerNames := make([]game.PlayerName, len(g.players))
	snagPlayerNames[0] = m.PlayerName
	i := 1
	for n2 := range g.players {
		if m.PlayerName != n2 {
			snagPlayerNames[i] = n2
			i++
		}
	}
	g.shufflePlayersFunc(snagPlayerNames[1:])
	for _, n2 := range snagPlayerNames {
		var m2 *game.Message
		switch {
		case n2 == m.PlayerName:
			m2 = &game.Message{
				Type:       game.SocketInfo,
				PlayerName: n2,
				Tiles:      g.unusedTiles[:1],
				Info:       "snagged a tile",
			}
			if err := g.players[n2].AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		case len(g.unusedTiles) == 0:
			m2 = &game.Message{
				Type:       game.SocketInfo,
				PlayerName: n2,
				Info:       fmt.Sprintf("%v snagged a tile", m.PlayerName),
			}
		default:
			m2 = &game.Message{
				Type:       game.SocketInfo,
				PlayerName: n2,
				Info:       fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName),
				Tiles:      g.unusedTiles[:1],
			}
			if err := g.players[n2].AddTile(g.unusedTiles[0]); err != nil {
				return err
			}
			g.unusedTiles = g.unusedTiles[1:]
		}
		snagPlayerMessages[n2] = *m2
	}
	for _, m := range snagPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		out <- m
	}
	return nil
}

func (g *Game) handleGameSwap(m game.Message, out chan<- game.Message) error {
	if g.status != game.InProgress {
		return gameWarning("game has not started or is finished")
	}
	if len(m.Tiles) != 1 {
		return gameWarning("no tile specified for swap")
	}
	if len(g.unusedTiles) == 0 {
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
		switch {
		case n == m.PlayerName:
			out <- game.Message{
				Type:       game.SocketInfo,
				PlayerName: n,
				Info:       fmt.Sprintf("swapping %v tile", t.Ch),
				Tiles:      newTiles,
				TilesLeft:  len(g.unusedTiles),
			}
		default:
			out <- game.Message{
				Type:       game.SocketInfo,
				PlayerName: n,
				Info:       fmt.Sprintf("%v swapped a tile", m.PlayerName),
				TilesLeft:  len(g.unusedTiles),
			}
		}
	}
	return nil
}

func (g *Game) handleGameTilesMoved(m game.Message, out chan<- game.Message) error {
	p := g.players[m.PlayerName]
	return p.MoveTiles(m.TilePositions)
}

func (g *Game) handleBoardRefresh(m game.Message, out chan<- game.Message) error {
	p := g.players[m.PlayerName]
	unusedTiles := make([]tile.Tile, len(p.UnusedTiles))
	for i, id := range p.UnusedTileIDs {
		unusedTiles[i] = p.UnusedTiles[id]
	}
	usedTilePositions := make([]tile.Position, len(p.UsedTiles))
	i := 0
	for _, tps := range p.UsedTiles {
		usedTilePositions[i] = tps
		i++
	}
	sort.Slice(usedTilePositions, func(i, j int) bool {
		a, b := usedTilePositions[i], usedTilePositions[j]
		// top-bottom, left-right
		switch {
		case a.Y == b.Y:
			return a.X < b.X
		default:
			return a.Y > b.Y
		}
	})
	out <- game.Message{
		Type:          m.Type,
		PlayerName:    m.PlayerName,
		Info:          m.Info,
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
		TilesLeft:     len(g.unusedTiles),
		GameStatus:    g.status,
		GamePlayers:   g.playerNames(),
		GameID:        g.id,
	}
	return nil
}

func (g *Game) handleGameInfos(m game.Message, out chan<- game.Message) error {
	var canJoin bool
	switch g.status {
	case game.NotStarted:
		canJoin = true
	case game.InProgress, game.Finished:
		_, canJoin = g.players[m.PlayerName]
	}
	m.GameInfoChan <- game.Info{
		ID:        g.id,
		Status:    g.status,
		Players:   g.playerNames(),
		CanJoin:   canJoin,
		CreatedAt: g.createdAt,
	}
	return nil
}

func (g *Game) handleGameChat(m game.Message, out chan<- game.Message) error {
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

func (g *Game) updateUserPoints(winningPlayerName game.PlayerName) error {
	users := make([]db.Username, len(g.players))
	i := 0
	for n := range g.players {
		users[i] = db.Username(n)
		i++
	}
	userPointsIncrementFunc := func(u db.Username) int {
		if string(u) == string(winningPlayerName) {
			p := g.players[winningPlayerName]
			return int(p.winPoints)
		}
		return 1
	}
	return g.userDao.UpdatePointsIncrement(users, userPointsIncrementFunc)
}

func (g Game) playerNames() []string {
	playerNames := make([]string, len(g.players))
	i := 0
	for n := range g.players {
		playerNames[i] = string(n)
		i++
	}
	sort.Strings(playerNames)
	return playerNames
}
