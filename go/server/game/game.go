package game

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
	// TODO: refactor: replace db.Username with string to avoid imports in this package
	// possibly remove imports on fmt, log
)

type (
	game struct {
		log         *log.Logger
		id          int
		lobby       *lobby
		createdAt   string
		state       gameState
		words       map[string]bool
		players     map[db.Username]*gamePlayerState
		userDao     db.UserDao
		unusedTiles []tile
		maxPlayers  int
		numNewTiles int
		tileLetters string
		messages    chan message
		active      bool
		// the shuffle functions shuffles the slices my mutating them
		shuffleUnusedTilesFunc func(tiles []tile)
		shufflePlayersFunc     func(players []*player)
	}

	gameState int

	gameInfo struct {
		ID        int       `json:"id"`
		State     gameState `json:"state"`
		Players   []string  `json:"players"`
		CanJoin   bool      `json:"canJoin"`
		CreatedAt string    `json:"createdAt"`
	}

	gamePlayerState struct {
		log           *log.Logger
		player        *player
		refreshTicker *time.Ticker
		unusedTiles   map[int]tile
		unusedTileIds []int
		usedTiles     map[int]tilePosition
		usedTileLocs  map[int]map[int]tile // X -> Y -> tile
		winPoints     int
	}
)

const (
	defaultTileLetters = "AAAAAAAAAAAAABBBCCCDDDDDDEEEEEEEEEEEEEEEEEEFFFGGGGHHHIIIIIIIIIIIIJJKKLLLLLMMMNNNNNNNNOOOOOOOOOOOPPPQQRRRRRRRRRSSSSSSTTTTTTTTTUUUUUUVVVWWWXXYYYZZ"
	// not using iota because gameStates hardcoded in ui javascript
	gameInProgress gameState = 1
	gameFinished   gameState = 2
	gameNotStarted gameState = 3
	// TODO: make this an environment argument
	gameIdlePeriod = 15 * time.Minute
)

// initialize unusedTiles from tileLetters or defaultTileLetters and shuffles them
func (g *game) initializeUnusedTiles() {
	if len(g.tileLetters) == 0 {
		g.tileLetters = defaultTileLetters
	}
	g.unusedTiles = make([]tile, len(g.tileLetters))
	for i, ch := range g.tileLetters {
		g.unusedTiles[i] = tile{
			ID: i + 1,
			Ch: letter(ch),
		}
	}
	if g.shuffleUnusedTilesFunc != nil {
		g.shuffleUnusedTilesFunc(g.unusedTiles)
	}
}

func (g *game) run() {
	idleTicker := time.NewTicker(gameIdlePeriod)
	defer idleTicker.Stop()
	defer func() {
		g.lobby.messages <- message{Type: gameDelete}
	}()
	messageHandlers := map[messageType]func(message){
		gameJoin:          g.handleGameJoin,
		gameLeave:         g.handleGameLeave,
		gameDelete:        g.handleGameDelete,
		gameStateChange:   g.handleGameStateChange,
		gameSnag:          g.handleGameSnag,
		gameSwap:          g.handleGameSwap,
		gameTileMoved:     g.handleGameTileMoved,
		gameTilePositions: g.handleGameTilePositions,
		gameInfos:         g.handleGameInfos,
		playerDelete:      g.handlePlayerDelete,
		gameChatRecv:      g.handleGameChatRecv,
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
				g.log.Printf("game does not know how to handle messageType %v", m.Type)
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

func (g *game) handleGameJoin(m message) {
	if _, ok := g.players[m.Player.username]; ok {
		g.players[m.Player.username].player = m.Player // replace the connection
		g.handleGameTilePositions(m)
		return
	}
	if len(g.players) >= g.maxPlayers {
		m.Player.messages <- message{Type: socketError, Info: "no room for another player in game"}
		return
	}
	if len(g.unusedTiles) < g.numNewTiles {
		m.Player.messages <- message{
			Type: gameDelete,
			Info: "deleting game because it can not start: there are not enough tiles",
		}
		return
	}
	newTiles := g.unusedTiles[:g.numNewTiles]
	g.unusedTiles = g.unusedTiles[g.numNewTiles:]
	newTilesByID := make(map[int]tile, g.numNewTiles)
	newTileIds := make([]int, g.numNewTiles)
	for i, t := range newTiles {
		newTilesByID[t.ID] = t
		newTileIds[i] = t.ID
	}
	gameTilePositionsTicker := time.NewTicker(1 * time.Minute) // TODO: make env var
	gps := &gamePlayerState{
		log:           g.log,
		player:        m.Player,
		refreshTicker: gameTilePositionsTicker,
		unusedTiles:   newTilesByID,
		unusedTileIds: newTileIds,
		usedTiles:     make(map[int]tilePosition),
		usedTileLocs:  make(map[int]map[int]tile),
		winPoints:     10,
	}
	g.players[m.Player.username] = gps
	go func() {
		for {
			_, ok := <-gameTilePositionsTicker.C
			if !ok {
				return
			}
			g.messages <- message{Type: gameTilePositions, Player: gps.player} // TODO: should only be username, not player
		}
	}()
	gamePlayers := g.playerUsernames()
	m.Player.messages <- message{
		Type:        socketInfo,
		Info:        "joining game",
		Tiles:       newTiles,
		TilesLeft:   len(g.unusedTiles),
		GamePlayers: gamePlayers,
		GameState:   g.state,
	}
	for u, gps := range g.players {
		if u != m.Player.username {
			gps.player.messages <- message{
				Type:        socketInfo,
				Info:        fmt.Sprintf("%v joined the game", m.Player.username),
				TilesLeft:   len(g.unusedTiles),
				GamePlayers: gamePlayers,
			}
		}
	}
}

func (g *game) handleGameLeave(m message) {
	g.log.Printf("%v left a game", m.Player.username)
}

func (g *game) handleGameDelete(m message) {
	for _, gps := range g.players {
		gps.refreshTicker.Stop()
		gps.player.messages <- message{
			Type: gameLeave,
			Info: m.Info,
		}
	}
}

func (g *game) handleGameStateChange(m message) {
	switch g.state {
	case gameNotStarted:
		if m.GameState != gameInProgress {
			m.Player.messages <- message{Type: socketError, Info: "game already started or is finished"}
			return
		}
		g.start()
	case gameInProgress:
		if m.GameState != gameFinished {
			m.Player.messages <- message{Type: socketError, Info: "can only attempt to set game that is in progress to finished"}
			return
		}
		g.finish(m.Player)
	default:
		if m.GameState != gameFinished {
			m.Player.messages <- message{Type: socketError, Info: fmt.Sprintf("cannot change game state from %v", g.state)}
			return
		}
	}
}

func (g *game) start() {
	g.state = gameInProgress
	for _, gps := range g.players {
		gps.player.messages <- message{
			Type:      socketInfo,
			Info:      "game started",
			GameState: g.state,
		}
	}
}

func (g *game) finish(finishingPlayer *player) {
	if len(g.unusedTiles) != 0 {
		finishingPlayer.messages <- message{Type: socketError, Info: "snag first"}
		return
	}
	gps := g.players[finishingPlayer.username]
	if len(gps.unusedTiles) != 0 {
		gps.decrementWinPoints()
		finishingPlayer.messages <- message{Type: socketError, Info: "not all tiles used"}
		return
	}
	if !gps.singleUsedGroup() {
		gps.decrementWinPoints()
		finishingPlayer.messages <- message{Type: socketError, Info: "not all used tiles form a single group"}
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
		finishingPlayer.messages <- message{
			Type: socketError,
			Info: fmt.Sprintf("invalid words: %v", invalidWords),
		}
		return
	}
	g.state = gameFinished
	info := fmt.Sprintf("WINNER! - %v won, creating %v words, getting %v points.  Other players each get 1 point", finishingPlayer.username, len(usedWords), gps.winPoints)
	g.updateUserPoints(finishingPlayer.username)
	for _, gps := range g.players {
		gps.player.messages <- message{
			Type:      socketInfo,
			Info:      info,
			GameState: g.state,
		}
	}
	g.messages <- message{Type: gameDelete}
}

func (g *game) handleGameSnag(m message) {
	if g.state != gameInProgress {
		m.Player.messages <- message{Type: socketError, Info: "game has not started or is finished"}
		return
	}
	if len(g.unusedTiles) == 0 {
		m.Player.messages <- message{Type: socketError, Info: "no tiles left to snag, use what you have to finish"}
		return
	}
	snagPlayerMessages := make(map[db.Username]message, len(g.players))
	snagPlayerMessages[m.Player.username] = message{
		Type:  socketInfo,
		Info:  "snagged a tile",
		Tiles: g.unusedTiles[:1],
	}
	g.players[m.Player.username].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
	g.players[m.Player.username].unusedTileIds = append(g.players[m.Player.username].unusedTileIds, g.unusedTiles[0].ID)
	g.unusedTiles = g.unusedTiles[1:]
	otherPlayers := make([]*player, len(g.players)-1)
	i := 0
	for u2, p2 := range g.players {
		if m.Player.username != u2 {
			otherPlayers[i] = p2.player
			i++
		}
	}
	g.shufflePlayersFunc(otherPlayers)
	for i := 0; i < len(otherPlayers) && len(g.unusedTiles) > 0; i++ {
		snagPlayerMessages[otherPlayers[i].username] = message{
			Type:  socketInfo,
			Info:  fmt.Sprintf("%v snagged a tile, adding a tile to your pile", m.Player.username),
			Tiles: g.unusedTiles[:1],
		}
		g.players[otherPlayers[i].username].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
		g.players[otherPlayers[i].username].unusedTileIds = append(g.players[otherPlayers[i].username].unusedTileIds, g.unusedTiles[0].ID)
		g.unusedTiles = g.unusedTiles[1:]
	}
	for u, m := range snagPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.players[u].player.messages <- m
	}
}

func (g *game) handleGameSwap(m message) {
	if g.state != gameInProgress {
		m.Player.messages <- message{Type: socketError, Info: "game has not started or is finished"}
		return
	}
	if len(m.Tiles) != 1 {
		m.Player.messages <- message{Type: socketError, Info: "no tile specified for swap"}
		return
	}
	if len(g.unusedTiles) == 0 {
		m.Player.messages <- message{Type: socketError, Info: "no tiles left to swap, user what you have to finish"}
		return
	}
	t := m.Tiles[0]
	gps := g.players[m.Player.username]
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
	var newTiles []tile
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		gps.unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
		gps.unusedTileIds = append(gps.unusedTileIds, g.unusedTiles[0].ID)
		g.unusedTiles = g.unusedTiles[1:]
	}
	swapPlayerMessages := make(map[db.Username]message, len(g.players))
	swapPlayerMessages[m.Player.username] = message{
		Type:  socketInfo,
		Info:  fmt.Sprintf("swapping %v tile", t.Ch),
		Tiles: newTiles,
	}
	for u, p2 := range g.players {
		if u != m.Player.username {
			swapPlayerMessages[p2.player.username] = message{
				Type: socketInfo,
				Info: fmt.Sprintf("%v swapped a tile", m.Player.username),
			}
		}
	}
	for u, m := range swapPlayerMessages {
		m.TilesLeft = len(g.unusedTiles)
		g.players[u].player.messages <- m
	}
}

func (g *game) handleGameTileMoved(m message) {
	gps := g.players[m.Player.username]
	tp := m.TilePositions[0]
	switch len(m.TilePositions) {
	case 1:
		if _, ok := gps.unusedTiles[tp.Tile.ID]; !ok {
			m.Player.messages <- message{
				Type: socketError,
				Info: fmt.Sprintf("trying to add tile %v to the words area, but it is not in the unused pile", tp.Tile),
			}
			return
		}
		delete(gps.unusedTiles, tp.Tile.ID)
		for i := 0; i < len(gps.unusedTileIds); i++ {
			if gps.unusedTileIds[i] == tp.Tile.ID {
				gps.unusedTileIds = append(gps.unusedTileIds[:i], gps.unusedTileIds[i+1:]...)
				break
			}
		}
	case 2:
		srcTp := tp
		tp = m.TilePositions[1]
		if xTiles, ok := gps.usedTileLocs[tp.X]; ok {
			if destTp, ok := xTiles[tp.Y]; ok && destTp.ID != tp.Tile.ID {
				m.Player.messages <- message{Type: socketError, Info: "trying move tile to location of other tile"}
				return
			}
		}
		delete(gps.usedTileLocs[srcTp.X], srcTp.Y)
	}
	if _, ok := gps.usedTileLocs[tp.X]; !ok {
		gps.usedTileLocs[tp.X] = make(map[int]tile)
	}
	if _, ok := gps.usedTileLocs[tp.X][tp.Y]; ok {
		m.Player.messages <- message{Type: socketError, Info: "trying move tile to used location"}
		return
	}
	gps.usedTiles[tp.Tile.ID] = tp
	gps.usedTileLocs[tp.X][tp.Y] = tp.Tile
}

func (g *game) handleGameTilePositions(m message) {
	gps := g.players[m.Player.username]
	unusedTiles := make([]tile, len(gps.unusedTiles))
	for i, id := range gps.unusedTileIds {
		unusedTiles[i] = gps.unusedTiles[id]
	}
	usedTilePositions := make([]tilePosition, len(gps.usedTiles))
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
	m.Player.messages <- message{
		Type:          socketInfo,
		Info:          m.Info,
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
		TilesLeft:     len(g.unusedTiles),
		GameState:     g.state,
		GamePlayers:   g.playerUsernames(),
	}
}

func (g *game) handleGameInfos(m message) {
	var canJoin bool
	switch g.state {
	case gameNotStarted:
		canJoin = true
	case gameInProgress:
		_, canJoin = g.players[m.Player.username]
	}
	gi := gameInfo{
		ID:        g.id,
		State:     g.state,
		Players:   g.playerUsernames(),
		CanJoin:   canJoin,
		CreatedAt: g.createdAt,
	}
	m.GameInfoChan <- gi
}

func (g *game) handlePlayerDelete(m message) {
	_, ok := g.players[m.Player.username]
	if !ok {
		m.Player.messages <- message{Type: socketError, Info: "cannot leave game player is not a part of"}
		return
	}
	g.players[m.Player.username].refreshTicker.Stop()
	delete(g.players, m.Player.username)
	if len(g.players) == 0 {
		g.lobby.messages <- message{Type: gameDelete, Info: "deleting game because it has no players"}
	}
}

func (g *game) handleGameChatRecv(m message) {
	m2 := message{
		Type: gameChatSend,
		Info: fmt.Sprintf("%v : %v", m.Player.username, m.Info),
	}
	for _, gps := range g.players {
		gps.player.messages <- m2
	}
}

func (g *game) updateUserPoints(winningUsername db.Username) {
	users := make([]db.Username, len(g.players))
	i := 0
	for u := range g.players {
		users[i] = u
		i++
	}
	gps := g.players[winningUsername]
	err := g.userDao.UpdatePointsIncrement(users, func(u db.Username) int {
		if u == winningUsername {
			return gps.winPoints
		}
		return 1
	})
	if err != nil {
		gps.player.messages <- message{
			Type: socketError,
			Info: fmt.Sprintf("updating user points: %v", err),
		}
	}
}

func (g game) playerUsernames() []string {
	usernames := make([]string, len(g.players))
	i := 0
	for u := range g.players {
		usernames[i] = string(u)
		i++
	}
	sort.Strings(usernames)
	return usernames
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
	return gps.usedWordsZ(gps.usedTileLocs, func(tp tilePosition) int { return tp.Y })
}

func (gps gamePlayerState) usedWordsX() []string {
	usedTilesYx := make(map[int]map[int]tile)
	for x, yTiles := range gps.usedTileLocs {
		for y, t := range yTiles {
			if _, ok := usedTilesYx[y]; !ok {
				usedTilesYx[y] = make(map[int]tile)
			}
			usedTilesYx[y][x] = t
		}
	}
	return gps.usedWordsZ(usedTilesYx, func(tp tilePosition) int { return tp.X })
}

func (gps gamePlayerState) usedWordsZ(tiles map[int]map[int]tile, ord func(tp tilePosition) int) []string {
	keyedUsedWords := make(map[int][]string, len(tiles))
	wordCount := 0
	for z, zTiles := range tiles {
		tilePositions := make([]tilePosition, len(zTiles))
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
			if i > 1 && ord(tilePositions[i-1]) < ord(tp)-1 && buffer.Len() > 1 {
				zWords = append(zWords, buffer.String())
				buffer = new(bytes.Buffer)
			}
			buffer.WriteString(tp.Tile.Ch.String())
			if i+1 == len(tilePositions) && buffer.Len() > 1 {
				zWords = append(zWords, buffer.String())
			}
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
	if gps.player != nil { // TODO: remove these debug logs
		gps.log.Printf("tiles for %v:", gps.player.username)
		for _, r := range gps.getUsedTilesRows() {
			gps.log.Print(r)
		}
	}
	seenTileIds := make(map[int]bool)
	for x, yTiles := range gps.usedTileLocs {
		for y, t := range yTiles {
			gps.addSeenTileIds(x, y, t, seenTileIds)
			break // only check one tile's surrounding tilePositions
		}
		break
	}
	return len(seenTileIds) == len(gps.usedTiles)
}

// helper function which does a depth-first search for surrounding tiles, modifying the seenTileIds map
func (gps gamePlayerState) addSeenTileIds(x, y int, t tile, seenTileIds map[int]bool) {
	seenTileIds[t.ID] = true
	for dx := -1; dx <= 1; dx++ { // check neighboring columns
		for dy := -1; dy <= 1; dy++ { // check neighboring rows
			if (dx != 0 || dy != 0) && dx*dy == 0 { // one delta is not zero, the other is
				if yTiles, ok := gps.usedTileLocs[x+dx]; ok { // x+dx is valid
					if t2, ok := yTiles[y+dy]; ok { // y+dy is valid
						if _, ok := seenTileIds[t2.ID]; !ok { // tile not yet seen
							gps.addSeenTileIds(x+dx, y+dy, t2, seenTileIds) // recursive call
						}
					}
				}
			}
		}
	}
}

// Debug function
func (gps gamePlayerState) getUsedTilesRows() []string {
	// compute bounds
	var xSeen, ySeen bool
	var xMin, xMax, yMin, yMax int
	for x, yTiles := range gps.usedTileLocs {
		switch {
		case !xSeen:
			xSeen = true
			xMin = x
			xMax = x
		case x < xMin:
			xMin = x
		case x > xMax:
			xMax = x
		}
		for y := range yTiles {
			switch {
			case !ySeen:
				ySeen = true
				yMin = y
				yMax = y
			case y < yMin:
				yMin = y
			case y > yMax:
				yMax = y
			}
		}
	}
	// computer array
	rows := make([]string, yMax-yMin+1)
	for r := yMin; r <= yMax; r++ {
		var buffer bytes.Buffer
		for c := xMin; c <= xMax; c++ {
			var tileFound bool
			if yTiles, ok := gps.usedTileLocs[c]; ok {
				if t, ok := yTiles[r]; ok {
					buffer.WriteRune(rune(t.Ch))
					tileFound = true
				}
			}
			if !tileFound {
				buffer.WriteRune(' ')
			}
		}
		rows[r-yMin] = buffer.String()
	}
	return rows
}
