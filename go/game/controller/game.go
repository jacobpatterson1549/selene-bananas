package controller

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/db"
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/game/tile"
)

type (
	// Game contains the logic to play a tile-base word-forming game between users
	Game struct {
		log         *log.Logger
		id          game.ID
		lobby       game.MessageHandler
		createdAt   string
		status      game.Status
		players     map[game.PlayerName]*gamePlayerState
		userDao     db.UserDao
		unusedTiles []tile.Tile
		maxPlayers  int
		numNewTiles int
		tileLetters string
		words       map[string]bool
		messages    chan game.Message
		active      bool
		// the shuffle functions shuffles the slices my mutating them
		shuffleUnusedTilesFunc func(tiles []tile.Tile)
		shufflePlayersFunc     func(playerNames []game.PlayerName)
	}

	// Config contiains the properties to create similar games
	Config struct {
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

	gamePlayerState struct {
		player        game.MessageHandler
		refreshTicker *time.Ticker
		unusedTiles   map[tile.ID]tile.Tile
		unusedTileIds []tile.ID
		usedTiles     map[tile.ID]tile.Position
		usedTileLocs  map[tile.X]map[tile.Y]tile.Tile
		winPoints     int
	}
)

const (
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
	// TODO: make these  environment arguments
	gameIdlePeriod                 = 15 * time.Minute
	gameTilePositionsRefreshPeriod = 5 * time.Minute
)

// Handle adds a message to the queue
func (g *Game) Handle(m game.Message) {
	g.messages <- m
}

// New creates a new game from the config and runs it
func (cfg Config) New(id game.ID, player game.MessageHandler) Game {
	// TODO: for createdAt, have TimeFunc variable that is a function which returns a time.Time, (time.Now) => TimeFunc().Format(...), share with jwt token
	g := Game{
		log:                    cfg.Log,
		lobby:                  cfg.Lobby,
		id:                     id,
		createdAt:              time.Now().Format(time.UnixDate),
		status:                 game.NotStarted,
		words:                  cfg.Words,
		players:                make(map[game.PlayerName]*gamePlayerState, 2),
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
	idleTicker := time.NewTicker(gameIdlePeriod)
	defer idleTicker.Stop()
	defer func() {
		g.lobby.Handle(game.Message{
			Type: game.Delete,
		})
	}()
	messageHandlers := map[game.MessageType]func(game.Message){
		game.Join:          g.handleGameJoin,
		game.Leave:         g.handleGameLeave,
		game.Delete:        g.handleGameDelete,
		game.StatusChange:  g.handleGameStateChange,
		game.Snag:          g.handleGameSnag,
		game.Swap:          g.handleGameSwap,
		game.TilesMoved:    g.handleGameTilesMoved,
		game.TilePositions: g.handleGameTilePositions,
		game.Infos:         g.handleGameInfos,
		game.PlayerDelete:  g.handlePlayerDelete,
		game.ChatRecv:      g.handleGameChatRecv,
	}
	for {
		select {
		case m, ok := <-g.messages:
			if !ok {
				return
			}
			g.active = true
			mh, ok := messageHandlers[m.Type]
			if !ok {
				g.log.Printf("game does not know how to handle MessageType %v", m.Type)
				continue
			}
			// TODO: validate Player, Tiles, TilePositions, ensure certain fields not set, ...
			mh(m)
		case _, ok := <-idleTicker.C:
			if !ok {
				return
			}
			if !g.active {
				g.log.Print("closing game due to inactivity")
				return
			}
			g.active = false
		}
	}
}

func (g *Game) handleGameJoin(m game.Message) {
	if _, ok := g.players[m.PlayerName]; ok {
		g.players[m.PlayerName].player = m.Player // replace the connection
		m.Type = game.SocketInfo
		g.handleGameTilePositions(m)
		return
	}
	if len(g.players) >= g.maxPlayers {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "no room for another player in game",
		})
		return
	}
	if len(g.unusedTiles) < g.numNewTiles {
		m.Player.Handle(game.Message{
			Type: game.Delete,
			Info: "deleting game because it can not start: there are not enough tiles",
		})
		return
	}
	newTiles := g.unusedTiles[:g.numNewTiles]
	g.unusedTiles = g.unusedTiles[g.numNewTiles:]
	newTilesByID := make(map[tile.ID]tile.Tile, g.numNewTiles)
	newTileIds := make([]tile.ID, g.numNewTiles)
	for i, t := range newTiles {
		newTilesByID[t.ID] = t
		newTileIds[i] = t.ID
	}
	gameTilePositionsTicker := time.NewTicker(gameTilePositionsRefreshPeriod)
	gps := &gamePlayerState{
		player:        m.Player,
		refreshTicker: gameTilePositionsTicker,
		unusedTiles:   newTilesByID,
		unusedTileIds: newTileIds,
		usedTiles:     make(map[tile.ID]tile.Position),
		usedTileLocs:  make(map[tile.X]map[tile.Y]tile.Tile),
		winPoints:     10,
	}
	g.players[m.PlayerName] = gps
	go func() {
		for {
			_, ok := <-gameTilePositionsTicker.C
			if !ok {
				return
			}
			g.Handle(game.Message{
				Type:       game.TilePositions,
				Player:     gps.player, // TODO: should only be username, not player DO THIS EVERYWHERE
				PlayerName: m.PlayerName,
			})
		}
	}()
	gamePlayers := g.playerNames()
	m.Player.Handle(game.Message{
		Type:        game.SocketInfo,
		Info:        "joining game",
		Tiles:       newTiles,
		TilesLeft:   len(g.unusedTiles),
		GamePlayers: gamePlayers,
		GameStatus:  g.status,
	})
	for u, gps := range g.players {
		if u != m.PlayerName {
			gps.player.Handle(game.Message{
				Type:        game.SocketInfo,
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
	for _, gps := range g.players {
		gps.refreshTicker.Stop()
		gps.player.Handle(game.Message{
			Type: game.Leave,
			Info: m.Info,
		})
	}
}

func (g *Game) handleGameStateChange(m game.Message) {
	switch g.status {
	case game.NotStarted:
		if m.GameStatus != game.InProgress {
			m.Player.Handle(game.Message{
				Type: game.SocketError,
				Info: "game already started or is finished",
			})
			return
		}
		g.start(m)
	case game.InProgress:
		if m.GameStatus != game.Finished {
			m.Player.Handle(game.Message{
				Type: game.SocketError,
				Info: "can only attempt to set game that is in progress to finished",
			})
			return
		}
		g.finish(m.PlayerName)
	default:
		if m.GameStatus != game.Finished {
			m.Player.Handle(game.Message{
				Type: game.SocketError,
				Info: fmt.Sprintf("cannot change game state from %v", g.status),
			})
			return
		}
	}
}

func (g *Game) start(m game.Message) {
	g.status = game.InProgress
	info := fmt.Sprintf("%v started the game", m.PlayerName)
	for _, gps := range g.players {
		gps.player.Handle(game.Message{
			Type:       game.SocketInfo,
			Info:       info,
			GameStatus: g.status,
		})
	}
}

func (g *Game) finish(finishingPlayerName game.PlayerName) {
	gps := g.players[finishingPlayerName]
	if len(g.unusedTiles) != 0 {
		gps.player.Handle(game.Message{
			Type: game.SocketError,
			Info: "snag first",
		})
		return
	}
	if len(gps.unusedTiles) != 0 {
		gps.decrementWinPoints()
		gps.player.Handle(game.Message{
			Type: game.SocketError, Info: "not all tiles used",
		})
		return
	}
	if !gps.singleUsedGroup() {
		gps.decrementWinPoints()
		gps.player.Handle(game.Message{
			Type: game.SocketError,
			Info: "not all used tiles form a single group",
		})
		return
	}
	usedWords := gps.usedWords()
	var invalidWords []string
	for _, w := range usedWords {
		lowerW := strings.ToLower(w) // TODO: this is innefficient, words are lowercase, tiles are uppercase...
		if _, ok := g.words[lowerW]; !ok {
			invalidWords = append(invalidWords, w)
		}
	}
	if len(invalidWords) > 0 {
		gps.decrementWinPoints()
		gps.player.Handle(game.Message{
			Type: game.SocketError,
			Info: fmt.Sprintf("invalid words: %v", invalidWords),
		})
		return
	}
	g.status = game.Finished
	info := fmt.Sprintf(
		"WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point",
		finishingPlayerName,
		len(usedWords),
		gps.winPoints,
	)
	g.updateUserPoints(finishingPlayerName)
	for _, gps := range g.players {
		gps.player.Handle(game.Message{
			Type:       game.SocketInfo,
			Info:       info,
			GameStatus: g.status,
		})
	}
	g.Handle(game.Message{
		Type: game.Delete,
	})
}

func (g *Game) handleGameSnag(m game.Message) {
	if g.status != game.InProgress {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "game has not started or is finished",
		})
		return
	}
	if len(g.unusedTiles) == 0 {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "no tiles left to snag, use what you have to finish",
		})
		return
	}
	snagPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	snagPlayerMessages[m.PlayerName] = game.Message{
		Type:  game.SocketInfo,
		Info:  "snagged a tile",
		Tiles: g.unusedTiles[:1],
	}
	g.players[m.PlayerName].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
	g.players[m.PlayerName].unusedTileIds = append(g.players[m.PlayerName].unusedTileIds, g.unusedTiles[0].ID)
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
			Type:  game.SocketInfo,
			Info:  fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.PlayerName),
			Tiles: g.unusedTiles[:1],
		}
		g.players[n2].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
		g.players[n2].unusedTileIds = append(g.players[n2].unusedTileIds, g.unusedTiles[0].ID)
		g.unusedTiles = g.unusedTiles[1:]
	}
	for n, m := range snagPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.players[n].player.Handle(m)
	}
}

func (g *Game) handleGameSwap(m game.Message) {
	if g.status != game.InProgress {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "game has not started or is finished",
		})
		return
	}
	if len(m.Tiles) != 1 {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "no tile specified for swap",
		})
		return
	}
	if len(g.unusedTiles) == 0 {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "no tiles left to swap, user what you have to finish",
		})
		return
	}
	t := m.Tiles[0]
	gps := g.players[m.PlayerName]
	g.unusedTiles = append(g.unusedTiles, t)
	if _, ok := gps.unusedTiles[t.ID]; ok {
		// TODO: gps.removeTile(id) which calls gps.removeUnusedTile(), gps.removeUsedTile()
		delete(gps.unusedTiles, t.ID)
		for i := 0; i < len(gps.unusedTileIds); i++ {
			if gps.unusedTileIds[i] == t.ID {
				gps.unusedTileIds = append(gps.unusedTileIds[:i], gps.unusedTileIds[i+1:]...)
				break
			}
		}
	} else {
		tp := gps.usedTiles[t.ID]
		delete(gps.usedTiles, t.ID)
		delete(gps.usedTileLocs[tp.X], tp.Y)
	}
	g.shuffleUnusedTilesFunc(g.unusedTiles)
	var newTiles []tile.Tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		gps.unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
		gps.unusedTileIds = append(gps.unusedTileIds, g.unusedTiles[0].ID)
		g.unusedTiles = g.unusedTiles[1:]
	}
	swapPlayerMessages := make(map[game.PlayerName]game.Message, len(g.players))
	swapPlayerMessages[m.PlayerName] = game.Message{
		Type:  game.SocketInfo,
		Info:  fmt.Sprintf("swapping %v tile", t.Ch),
		Tiles: newTiles,
	}
	for n := range g.players {
		if n != m.PlayerName {
			swapPlayerMessages[n] = game.Message{
				Type: game.SocketInfo,
				Info: fmt.Sprintf("%v swapped a tile", m.PlayerName),
			}
		}
	}
	for n, m := range swapPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.players[n].player.Handle(m)
	}
}

func (g *Game) handleGameTilesMoved(m game.Message) {
	if m.Player == nil || m.TilePositions == nil {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "missing player or tilePositions",
		})
		return
	}
	gps := g.players[m.PlayerName]
	// validation
	newUsedTileLocs := make(map[tile.X]map[tile.Y]bool)
	movedTileIds := make(map[tile.ID]bool, len(m.TilePositions))
	for _, tp := range m.TilePositions {
		if _, ok := gps.unusedTiles[tp.Tile.ID]; ok {
			continue
		}
		if _, ok := gps.usedTiles[tp.Tile.ID]; !ok {
			m.Player.Handle(game.Message{
				Type: game.SocketError,
				Info: fmt.Sprintf("cannot move tile %v that the player does not own", tp.Tile),
			})
			return
		}
		movedTileIds[tp.Tile.ID] = true
		if _, ok := newUsedTileLocs[tp.X]; !ok {
			newUsedTileLocs[tp.X] = make(map[tile.Y]bool)
			newUsedTileLocs[tp.X][tp.Y] = true
			continue
		}
		if _, ok := newUsedTileLocs[tp.X][tp.Y]; ok {
			m.Player.Handle(game.Message{
				Type: game.SocketError,
				Info: fmt.Sprintf("cannot move multiple tiles to [%v,%v] ([c,r])", tp.X, tp.Y),
			})
			return
		}
		newUsedTileLocs[tp.X][tp.Y] = true
	}
	// for existing tp
	for x, yTiles := range gps.usedTileLocs {
		for y, t := range yTiles {
			if _, ok := movedTileIds[t.ID]; ok {
				continue
			}
			// duplicate-ish of above code:
			if _, ok := newUsedTileLocs[x]; !ok {
				newUsedTileLocs[x] = make(map[tile.Y]bool)
				newUsedTileLocs[x][y] = true
				continue
			}
			if _, ok := newUsedTileLocs[x][y]; ok {
				m.Player.Handle(game.Message{
					Type: game.SocketError,
					Info: "cannot move tiles to location of tile that is not being moved",
				})
				return
			}
			newUsedTileLocs[x][y] = true
		}
	}
	// actually move tiles
	for _, tp := range m.TilePositions {
		if _, ok := gps.unusedTiles[tp.Tile.ID]; ok { // unused
			gps.usedTiles[tp.Tile.ID] = tp
			delete(gps.unusedTiles, tp.Tile.ID)
			for i, id := range gps.unusedTileIds {
				if id == tp.Tile.ID {
					gps.unusedTileIds = append(gps.unusedTileIds[:i], gps.unusedTileIds[i+1:]...)
					break
				}
			}
		} else { // previously used
			tp2 := gps.usedTiles[tp.Tile.ID]
			t2 := gps.usedTileLocs[tp2.X][tp2.Y]
			if t2.ID == tp.Tile.ID { // remove from old location
				switch {
				case len(gps.usedTileLocs[tp2.X]) == 1:
					delete(gps.usedTileLocs, tp2.X)
				default:
					delete(gps.usedTileLocs[tp2.X], tp2.Y)
				}
			}
		}
		if _, ok := gps.usedTileLocs[tp.X]; !ok {
			gps.usedTileLocs[tp.X] = make(map[tile.Y]tile.Tile)
		}
		gps.usedTileLocs[tp.X][tp.Y] = tp.Tile
		gps.usedTiles[tp.Tile.ID] = tp
	}
}

func (g *Game) handleGameTilePositions(m game.Message) {
	gps := g.players[m.PlayerName]
	unusedTiles := make([]tile.Tile, len(gps.unusedTiles))
	for i, id := range gps.unusedTileIds {
		unusedTiles[i] = gps.unusedTiles[id]
	}
	usedTilePositions := make([]tile.Position, len(gps.usedTiles))
	i := 0
	for _, tps := range gps.usedTiles {
		usedTilePositions[i] = tps
		i++
	}
	// sort top-bottom, left-right
	sort.Slice(usedTilePositions, func(i, j int) bool {
		a, b := usedTilePositions[i], usedTilePositions[j]
		switch {
		case a.Y == b.Y:
			return a.X < b.X
		default:
			return a.Y > b.Y
		}
	})
	m.Player.Handle(game.Message{
		Type:          m.Type,
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
	_, ok := g.players[m.PlayerName]
	if !ok {
		m.Player.Handle(game.Message{
			Type: game.SocketError,
			Info: "cannot leave game player is not a part of",
		})
		return
	}
	g.players[m.PlayerName].refreshTicker.Stop()
	delete(g.players, m.PlayerName)
	if len(g.players) == 0 {
		g.lobby.Handle(game.Message{
			Type: game.Delete,
			Info: "deleting game because it has no players",
		})
	}
}

func (g *Game) handleGameChatRecv(m game.Message) {
	m2 := game.Message{
		Type: game.ChatSend,
		Info: fmt.Sprintf("%v : %v", m.PlayerName, m.Info),
	}
	for _, gps := range g.players {
		gps.player.Handle(m2)
	}
}

func (g *Game) updateUserPoints(winningPlayerName game.PlayerName) {
	users := make([]db.Username, len(g.players))
	i := 0
	for n := range g.players {
		users[i] = db.Username(n)
		i++
	}
	gps := g.players[winningPlayerName]
	err := g.userDao.UpdatePointsIncrement(users, func(u db.Username) int {
		if string(u) == string(winningPlayerName) {
			return gps.winPoints
		}
		return 1
	})
	if err != nil {
		gps.player.Handle(game.Message{
			Type: game.SocketError,
			Info: fmt.Sprintf("updating user points: %v", err),
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

func (gps *gamePlayerState) decrementWinPoints() {
	if gps.winPoints > 2 {
		gps.winPoints--
	}
}

func (gps gamePlayerState) usedWords() []string {
	horizontalWords := gps.usedWordsX()
	verticalWords := gps.usedWordsY()
	return append(horizontalWords, verticalWords...)
}

func (gps gamePlayerState) usedWordsY() []string {
	usedTilesXy := make(map[int]map[int]tile.Tile)
	for x, yTiles := range gps.usedTileLocs {
		xi := int(x)
		usedTilesXy[xi] = make(map[int]tile.Tile, len(yTiles))
		for y, t := range yTiles {
			usedTilesXy[xi][int(y)] = t
		}
	}
	return gps.usedWordsZ(usedTilesXy, func(tp tile.Position) int { return int(tp.Y) })
}

func (gps gamePlayerState) usedWordsX() []string {
	usedTilesYx := make(map[int]map[int]tile.Tile)
	for x, yTiles := range gps.usedTileLocs {
		xi := int(x)
		for y, t := range yTiles {
			yi := int(y)
			if _, ok := usedTilesYx[yi]; !ok {
				usedTilesYx[yi] = make(map[int]tile.Tile)
			}
			usedTilesYx[yi][xi] = t
		}
	}
	return gps.usedWordsZ(usedTilesYx, func(tp tile.Position) int { return int(tp.X) })
}

func (gps gamePlayerState) usedWordsZ(tiles map[int]map[int]tile.Tile, ord func(tp tile.Position) int) []string {
	keyedUsedWords := make(map[int][]string, len(tiles))
	wordCount := 0
	for z, zTiles := range tiles {
		tilePositions := make([]tile.Position, len(zTiles))
		i := 0
		for _, t := range zTiles {
			tilePositions[i] = gps.usedTiles[t.ID]
			i++
		}
		sort.Slice(tilePositions, func(i, j int) bool {
			return ord(tilePositions[i]) < ord(tilePositions[j])
		})
		buffer := new(bytes.Buffer)
		var zWords []string
		for i, tp := range tilePositions {
			if i > 0 && ord(tilePositions[i-1]) < ord(tp)-1 {
				if buffer.Len() > 1 {
					zWords = append(zWords, buffer.String())
				}
				buffer = new(bytes.Buffer)
			}
			buffer.WriteRune(rune(tp.Tile.Ch))
		}
		if buffer.Len() > 1 {
			zWords = append(zWords, buffer.String())
		}
		keyedUsedWords[z] = zWords
		wordCount += len(zWords)
	}
	//sort the keyedUsedWords by the keys (z)
	keys := make([]int, len(keyedUsedWords))
	i := 0
	for k := range keyedUsedWords {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	usedWords := make([]string, wordCount)
	i = 0
	for _, k := range keys {
		copy(usedWords[i:], keyedUsedWords[k])
		i += len(keyedUsedWords[k])
	}
	return usedWords
}

func (gps gamePlayerState) singleUsedGroup() bool {
	if len(gps.usedTiles) == 0 {
		return false
	}
	seenTileIds := make(map[tile.ID]bool)
	for x, yTiles := range gps.usedTileLocs {
		for y, t := range yTiles {
			gps.addSeenTileIds(int(x), int(y), t, seenTileIds)
			break // only check one tile's surrounding tilePositions
		}
		break
	}
	return len(seenTileIds) == len(gps.usedTiles)
}

// helper function which does a depth-first search for surrounding tiles, modifying the seenTileIds map
func (gps gamePlayerState) addSeenTileIds(x int, y int, t tile.Tile, seenTileIds map[tile.ID]bool) {
	seenTileIds[t.ID] = true
	for dx := -1; dx <= 1; dx++ { // check neighboring columns
		for dy := -1; dy <= 1; dy++ { // check neighboring rows
			if (dx != 0 || dy != 0) && dx*dy == 0 { // one delta is not zero, the other is
				if yTiles, ok := gps.usedTileLocs[tile.X(int(x)+dx)]; ok { // x+dx is valid
					if t2, ok := yTiles[tile.Y(int(y)+dy)]; ok { // y+dy is valid
						if _, ok := seenTileIds[t2.ID]; !ok { // tile not yet seen
							gps.addSeenTileIds(int(x)+dx, int(y)+dy, t2, seenTileIds) // recursive call
						}
					}
				}
			}
		}
	}
}
