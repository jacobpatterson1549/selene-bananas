package controller

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users
	Game struct {
		debug       bool
		log         *log.Logger
		id          game.ID
		lobby       game.MessageHandler
		createdAt   string
		status      game.Status
		players     map[game.PlayerName]*player
		userDao     db.UserDao
		unusedTiles []tile.Tile
		maxPlayers  int
		numNewTiles int
		tileLetters string
		words       map[string]bool
		messages    chan game.Message
		// the shuffle functions shuffles the slices my mutating them
		shuffleUnusedTilesFunc func(tiles []tile.Tile)
		shufflePlayersFunc     func(playerNames []game.PlayerName)
	}

	// Config contiains the properties to create similar games
	Config struct {
		Debug       bool
		Log         *log.Logger
		Lobby       game.MessageHandler
		UserDao     db.UserDao
		MaxPlayers  int
		NumNewTiles int
		TileLetters string
		Words       map[string]bool
		// the shuffle functions shuffles the slices my mutating them
		ShuffleUnusedTilesFunc func(tiles []tile.Tile)
		ShufflePlayersFunc     func(playerNames []game.PlayerName)
	}
)

const (
	// TODO: make these  environment arguments
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
	gameIdlePeriod     = 10 * time.Minute
	boardRefreshPeriod = 5 * time.Minute
)

var _ game.MessageHandler = &Game{}

// Handle adds a message to the queue
func (g *Game) Handle(m game.Message) {
	g.messages <- m
}

// NewGame creates a new game and runs it
func (cfg Config) NewGame(id game.ID) Game {
	// TODO: for createdAt, have TimeFunc variable that is a function which returns a time.Time, (time.Now) => TimeFunc().Format(...), share with jwt token
	g := Game{
		debug:                  cfg.Debug,
		log:                    cfg.Log,
		lobby:                  cfg.Lobby,
		id:                     id,
		createdAt:              time.Now().Format(time.UnixDate),
		status:                 game.NotStarted,
		words:                  cfg.Words,
		players:                make(map[game.PlayerName]*player, 2),
		userDao:                cfg.UserDao,
		maxPlayers:             cfg.MaxPlayers,
		numNewTiles:            cfg.NumNewTiles,
		tileLetters:            cfg.TileLetters,
		messages:               make(chan game.Message, 64),
		shuffleUnusedTilesFunc: cfg.ShuffleUnusedTilesFunc,
		shufflePlayersFunc:     cfg.ShufflePlayersFunc,
	}
	g.initializeUnusedTiles()
	go g.run()
	return g
}

// initialize unusedTiles from tileLetters or defaultTileLetters and shuffles them
func (g *Game) initializeUnusedTiles() {
	if len(g.tileLetters) == 0 {
		g.tileLetters = defaultTileLetters
	}
	g.unusedTiles = make([]tile.Tile, len(g.tileLetters))
	for i, ch := range g.tileLetters {
		id := tile.ID(i + 1)
		t, err := tile.New(id, ch)
		if err != nil {
			g.log.Printf("creating tile: %v", err)
			continue
		}
		g.unusedTiles[i] = t
	}
	if g.shuffleUnusedTilesFunc != nil {
		g.shuffleUnusedTilesFunc(g.unusedTiles)
	}
}

func (g *Game) run() {
	messageHandlers := map[game.MessageType]func(game.Message){
		game.Join:         g.handleGameJoin,
		game.Leave:        g.handleGameLeave,
		game.Delete:       g.handleGameDelete,
		game.StatusChange: g.handleGameStatusChange,
		game.Snag:         g.handleGameSnag,
		game.Swap:         g.handleGameSwap,
		game.TilesMoved:   g.handleGameTilesMoved,
		game.BoardRefresh: g.handleBoardRefresh,
		game.Infos:        g.handleGameInfos,
		game.PlayerDelete: g.handlePlayerDelete,
		game.ChatRecv:     g.handleGameChatRecv,
	}
	for {
		select {
		case m := <-g.messages:
			if g.debug {
				g.log.Printf("game handling message with type %v", m.Type)
			}
			mh, ok := messageHandlers[m.Type]
			if !ok {
				g.lobby.Handle(game.Message{
					Type:       game.SocketError,
					PlayerName: m.PlayerName,
					Info:       fmt.Sprintf("game does not know how to handle MessageType %v", m.Type),
				})
				continue
			}
			if _, ok := g.players[m.PlayerName]; !ok && m.Type != game.Join {
				g.lobby.Handle(game.Message{
					Type:       game.SocketError,
					PlayerName: m.PlayerName,
					Info:       fmt.Sprintf("game does not have player named '%v'", m.PlayerName),
				})
				continue
			}
			mh(m)
		}
	}
}

func (g *Game) handleGameJoin(m game.Message) {
	if _, ok := g.players[m.PlayerName]; ok {
		m.Type = game.SocketInfo
		g.Handle(game.Message{
			Type:       game.BoardRefresh,
			PlayerName: m.PlayerName,
		})
		return
	}
	if len(g.players) >= g.maxPlayers {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "no room for another player in game",
		})
		return
	}
	if len(g.unusedTiles) < g.numNewTiles {
		g.lobby.Handle(game.Message{
			Type: game.SocketInfo,
			Info: "deleting game because it can not start: there are not enough tiles",
		})
		return
	}
	newTiles := g.unusedTiles[:g.numNewTiles]
	g.unusedTiles = g.unusedTiles[g.numNewTiles:]
	boardRefreshTicker := time.NewTicker(boardRefreshPeriod)
	b := board.New(newTiles)
	p := &player{
		Ticker:    boardRefreshTicker,
		winPoints: 10,
		Board:     b,
	}
	g.players[m.PlayerName] = p
	go func() {
		for {
			<-boardRefreshTicker.C
			g.Handle(game.Message{
				Type:       game.BoardRefresh,
				PlayerName: m.PlayerName,
			})
		}
	}()
	gamePlayers := g.playerNames()
	g.lobby.Handle(game.Message{
		Type:        game.SocketInfo,
		PlayerName:  m.PlayerName,
		Info:        "joining game",
		Tiles:       newTiles,
		TilesLeft:   len(g.unusedTiles),
		GamePlayers: gamePlayers,
		GameStatus:  g.status,
	})
	for n := range g.players {
		if n != m.PlayerName {
			g.lobby.Handle(game.Message{
				Type:        game.SocketInfo,
				PlayerName:  n,
				Info:        fmt.Sprintf("%v joined the game", m.PlayerName),
				TilesLeft:   len(g.unusedTiles),
				GamePlayers: gamePlayers,
			})
		}
	}
}

func (g *Game) handleGameLeave(m game.Message) {
	g.log.Printf("%v left a game", m.PlayerName)
}

func (g *Game) handleGameDelete(m game.Message) {
	for n, p := range g.players {
		p.stopBoardRefresh()
		g.lobby.Handle(game.Message{
			Type:       game.Leave,
			PlayerName: n,
			Info:       m.Info,
		})
	}
}

func (g *Game) handleGameStatusChange(m game.Message) {
	switch g.status {
	case game.NotStarted:
		if m.GameStatus != game.InProgress {
			g.lobby.Handle(game.Message{
				Type:       game.SocketError,
				PlayerName: m.PlayerName,
				Info:       "game already started or is finished",
			})
			return
		}
		g.start(m.PlayerName)
	case game.InProgress:
		if m.GameStatus != game.Finished {
			g.lobby.Handle(game.Message{
				Type:       game.SocketError,
				PlayerName: m.PlayerName,
				Info:       "can only attempt to set game that is in progress to finished",
			})
			return
		}
		g.finish(m.PlayerName)
	default:
		if m.GameStatus != game.Finished {
			g.lobby.Handle(game.Message{
				Type:       game.SocketError,
				PlayerName: m.PlayerName,
				Info:       fmt.Sprintf("cannot change game state from %v", g.status),
			})
			return
		}
	}
}

func (g *Game) start(startingPlayerName game.PlayerName) {
	g.status = game.InProgress
	info := fmt.Sprintf("%v started the game", startingPlayerName)
	for n := range g.players {
		g.lobby.Handle(game.Message{
			Type:       game.SocketInfo,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
		})
	}
}

func (g *Game) finish(finishingPlayerName game.PlayerName) {
	p := g.players[finishingPlayerName]
	if len(p.UnusedTiles) != 0 {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: finishingPlayerName,
			Info:       "snag first",
		})
		return
	}
	if len(p.UnusedTiles) != 0 {
		p.decrementWinPoints()
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: finishingPlayerName,
			Info:       "not all tiles used",
		})
		return
	}
	if !p.HasSingleUsedGroup() {
		p.decrementWinPoints()
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: finishingPlayerName,
			Info:       "not all used tiles form a single group",
		})
		return
	}
	usedWords := p.UsedTileWords()
	var invalidWords []string
	for _, w := range usedWords {
		lowerW := strings.ToLower(w) // TODO: this is innefficient, words are lowercase, tiles are uppercase...
		if _, ok := g.words[lowerW]; !ok {
			invalidWords = append(invalidWords, w)
		}
	}
	if len(invalidWords) > 0 {
		p.decrementWinPoints()
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: finishingPlayerName,
			Info:       fmt.Sprintf("invalid words: %v", invalidWords),
		})
		return
	}
	g.status = game.Finished
	info := fmt.Sprintf(
		"WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point",
		finishingPlayerName,
		len(usedWords),
		p.winPoints,
	)
	g.updateUserPoints(finishingPlayerName)
	for n := range g.players {
		g.lobby.Handle(game.Message{
			Type:       game.SocketInfo,
			PlayerName: n,
			Info:       info,
			GameStatus: g.status,
		})
	}
}

func (g *Game) handleGameSnag(m game.Message) {
	if g.status != game.InProgress {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "game has not started or is finished",
		})
		return
	}
	if len(g.unusedTiles) == 0 {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "no tiles left to snag, use what you have to finish",
		})
		return
	}
	snagPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	snagPlayerMessages[m.PlayerName] = game.Message{
		Type:       game.SocketInfo,
		PlayerName: m.PlayerName,
		Info:       "snagged a tile",
		Tiles:      g.unusedTiles[:1],
	}
	g.players[m.PlayerName].AddTile(g.unusedTiles[0])
	g.unusedTiles = g.unusedTiles[1:]
	otherPlayers := make([]game.PlayerName, len(g.players)-1)
	i := 0
	for n2 := range g.players {
		if m.PlayerName != n2 {
			otherPlayers[i] = n2
			i++
		}
	}
	g.shufflePlayersFunc(otherPlayers)
	for _, n2 := range otherPlayers {
		if len(g.unusedTiles) == 0 {
			break
		}
		snagPlayerMessages[n2] = game.Message{
			Type:       game.SocketInfo,
			PlayerName: n2,
			Info:       fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName),
			Tiles:      g.unusedTiles[:1],
		}
		g.players[n2].AddTile(g.unusedTiles[0])
		g.unusedTiles = g.unusedTiles[1:]
	}
	for _, m := range snagPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.lobby.Handle(m)
	}
}

func (g *Game) handleGameSwap(m game.Message) {
	if g.status != game.InProgress {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "game has not started or is finished",
		})
		return
	}
	if len(m.Tiles) != 1 {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "no tile specified for swap",
		})
		return
	}
	if len(g.unusedTiles) == 0 {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       "no tiles left to swap, user what you have to finish",
		})
		return
	}
	t := m.Tiles[0]
	p := g.players[m.PlayerName]
	err := p.RemoveTile(t)
	if err != nil {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
		})
	}
	g.unusedTiles = append(g.unusedTiles, t)
	g.shuffleUnusedTilesFunc(g.unusedTiles)
	var newTiles []tile.Tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		p.AddTile(g.unusedTiles[0])
		g.unusedTiles = g.unusedTiles[1:]
	}
	swapPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	swapPlayerMessages[m.PlayerName] = game.Message{
		Type:       game.SocketInfo,
		PlayerName: m.PlayerName,
		Info:       fmt.Sprintf("swapping %v tile", t.Ch),
		Tiles:      newTiles,
	}
	for n := range g.players {
		if n != m.PlayerName {
			swapPlayerMessages[n] = game.Message{
				Type:       game.SocketInfo,
				PlayerName: n,
				Info:       fmt.Sprintf("%v swapped a tile", m.PlayerName),
			}
		}
	}
	for _, m := range swapPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.lobby.Handle(m)
	}
}

func (g *Game) handleGameTilesMoved(m game.Message) {
	err := g.players[m.PlayerName].MoveTiles(m.TilePositions)
	if err != nil {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: m.PlayerName,
			Info:       err.Error(),
		})
	}
}

func (g *Game) handleBoardRefresh(m game.Message) {
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
	g.lobby.Handle(game.Message{
		Type:          m.Type,
		PlayerName:    m.PlayerName,
		Info:          m.Info,
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
		TilesLeft:     len(g.unusedTiles),
		GameStatus:    g.status,
		GamePlayers:   g.playerNames(),
	})
}

func (g *Game) handleGameInfos(m game.Message) {
	var canJoin bool
	switch g.status {
	case game.NotStarted:
		canJoin = true
	case game.InProgress:
		_, canJoin = g.players[m.PlayerName]
	}
	m.GameInfoChan <- game.Info{
		ID:        g.id,
		Status:    g.status,
		Players:   g.playerNames(),
		CanJoin:   canJoin,
		CreatedAt: g.createdAt,
	}
}

func (g *Game) handlePlayerDelete(m game.Message) {
	g.players[m.PlayerName].stopBoardRefresh()
	delete(g.players, m.PlayerName)
	if len(g.players) == 0 {
		g.lobby.Handle(game.Message{
			Type: game.Delete,
			Info: "deleting game because it has no players",
		})
	}
}

func (g *Game) handleGameChatRecv(m game.Message) {
	info := fmt.Sprintf("%v : %v", m.PlayerName, m.Info)
	for n := range g.players {
		g.lobby.Handle(game.Message{
			Type:       game.ChatSend,
			PlayerName: n,
			Info:       info,
		})
	}
}

func (g *Game) updateUserPoints(winningPlayerName game.PlayerName) {
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
	err := g.userDao.UpdatePointsIncrement(users, userPointsIncrementFunc)
	if err != nil {
		g.lobby.Handle(game.Message{
			Type:       game.SocketError,
			PlayerName: winningPlayerName,
			Info:       fmt.Sprintf("updating user points: %v", err),
		})
	}
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
