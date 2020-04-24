package game

import (
	"fmt"
	"log"
	"sort"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
	// TODO: refactor: replace db.Username with string to avoid imports in this package
	// possibly remove imports on fmt, log
)

type (
	game struct {
		log         *log.Logger
		createdAt   string
		words       map[string]bool
		players     map[db.Username]gamePlayerState
		unusedTiles []tile
		started     bool
		maxPlayers  int
		messages    chan message
		// the shuffle functions shuffles the slices my mutating them
		shuffleTilesFunc   func(tiles []tile)
		shufflePlayersFunc func(players []*player)
	}

	gameInfo struct {
		Players   []db.Username `json:"players"`
		CanJoin   bool          `json:"canJoin"`
		CreatedAt string        `json:"createdAt"`
	}

	gamePlayerState struct {
		player       *player
		unusedTiles  map[int]tile
		usedTiles    map[int]tilePosition
		usedTileLocs map[int]map[int]tile // X -> Y -> tile
	}
)

func (g game) createTiles() []tile {
	var tiles []tile
	add := func(s string, n int) {
		for i := 0; i < len(s); i++ {
			ch := s[i]
			for j := 0; j < n; j++ {
				tiles = append(tiles, tile{Ch: letter(ch)})
			}
		}
	}
	add("JKQXZ", 2)
	add("BCFHMPVWY", 3)
	add("G", 4)
	add("L", 5)
	add("DSU", 6)
	add("N", 8)
	add("TR", 9)
	add("O", 11)
	add("I", 12)
	add("A", 13)
	add("E", 18)
	g.shuffleTilesFunc(tiles)
	for i := range tiles {
		// t.ID = i + 1
		tiles[i].ID = i + 1
	}
	return tiles
}

func (g game) run() {
	messageHandlers := map[messageType]func(message){
		gameJoin:          g.handleGameJoin,
		gameLeave:         g.handleGameLeave,
		gameDelete:        g.handleGameDelete,
		gameStart:         g.handleGameStart,
		gameSnag:          g.handleGameSnag,
		gameFinish:        g.handleGameFinish,
		gameTileMoved:     g.handleGameTileMoved,
		gameTilePositions: g.handleGameTilePositions,
		gameInfos:         g.handleGameInfos,
		playerDelete:      g.handlePlayerDelete,
	}
	for m := range g.messages {
		mh, ok := messageHandlers[m.Type]
		if !ok {
			g.log.Printf("game does not know how to handle messageType %v", m.Type)
			continue
		}
		// TODO: validate Player, Tiles, TilePositions, ensure certain fields not set, ...
		mh(m)
	}
}

func (g game) handleGameJoin(m message) {
	if g.started {
		m.Player.socket.messages <- message{Type: socketError, Info: "game already started"}
		return
	}
	_, ok := g.players[m.Player.username]
	if ok {
		m.Player.socket.messages <- message{Type: socketError, Info: "user already a part of game"}
		return
	}
	if len(g.players) >= g.maxPlayers {
		m.Player.socket.messages <- message{Type: socketError, Info: "no room for another player in game"}
		return
	}
	g.players[m.Player.username] = gamePlayerState{
		player:       m.Player,
		unusedTiles:  make(map[int]tile),
		usedTiles:    make(map[int]tilePosition),
		usedTileLocs: make(map[int]map[int]tile),
	}
}

func (g game) handleGameLeave(m message) {
	g.messages <- message{
		Type:   playerDelete,
		Player: m.Player,
	}
}

func (g game) handleGameDelete(m message) {
	for _, gps := range g.players {
		g.messages <- message{
			Type:   playerDelete,
			Player: gps.player,
		}
	}
}

func (g game) handleGameStart(m message) {
	if g.started {
		m.Player.socket.messages <- message{Type: socketError, Info: "game already started"}
		return
	}
	g.started = true
	newTiles := make(map[db.Username][]tile, len(g.players))
	for t := 0; t < 21; t++ {
		for u := range g.players {
			if len(g.unusedTiles) == 0 {
				m.Player.messages <- message{
					Type: gameDelete,
					Info: "deleting game because it can not start because there are not enough tiles",
				}
				return
			}
			t := g.unusedTiles[0]
			g.unusedTiles = g.unusedTiles[1:]
			pt, ok := newTiles[u]
			if ok {
				newTiles[u] = append(pt, t)
			} else {
				// TODO debug if this condition ever occurs
				newTiles[u] = []tile{t}
			}
		}
	}
	for u, gps := range g.players {
		gps.player.socket.messages <- message{
			Type:  gameSnag,
			Info:  fmt.Sprintf("starting game with tiles: %v", newTiles[u]),
			Tiles: newTiles[u],
		}
		for _, t := range newTiles[u] {
			gps.unusedTiles[t.ID] = t
		}
	}
}

func (g game) handleGameSnag(m message) {
	if len(g.unusedTiles) == 0 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tiles left to snag, use what you have to finish"}
		return
	}
	m.Player.socket.messages <- message{
		Type:  gameSwap,
		Info:  fmt.Sprintf("snagged a tile: %v", g.unusedTiles[0]),
		Tiles: g.unusedTiles[:1],
	}
	g.players[m.Player.username].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
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
		otherPlayers[i].socket.messages <- message{
			Type:  gameSwap,
			Info:  fmt.Sprintf("%v snagged a tile, adding %v to your tiles", m.Player.username, g.unusedTiles[0]),
			Tiles: g.unusedTiles[:1],
		}
		g.players[otherPlayers[i].username].unusedTiles[g.unusedTiles[0].ID] = g.unusedTiles[0]
		g.unusedTiles = g.unusedTiles[1:]
	}
}

func (g game) handleGameSwap(m message) {
	if len(m.Tiles) != 1 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tile specified for swap"}
		return
	}
	if len(g.unusedTiles) == 0 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tiles left to swap, user what you have to finish"}
		g.messages <- message{Type: gameTilePositions, Player: m.Player}
		return
	}
	t := m.Tiles[0]
	gps := g.players[m.Player.username]
	g.unusedTiles = append(g.unusedTiles, t)
	if _, ok := gps.unusedTiles[t.ID]; ok {
		delete(gps.unusedTiles, t.ID)
	} else {
		tp := gps.usedTiles[t.ID]
		delete(gps.usedTiles, t.ID)
		delete(gps.usedTileLocs[tp.X], tp.Y)
	}
	g.shuffleTilesFunc(g.unusedTiles)

	newTiles := make([]tile, 1)
	for i := 0; i < 3 && len(g.unusedTiles) > 0; i++ {
		newTiles = append(newTiles, g.unusedTiles[0])
		g.unusedTiles = g.unusedTiles[1:]
	}
	m.Player.socket.messages <- message{
		Type:  gameSwap,
		Info:  fmt.Sprintf("swapping %v tile for %v", t, newTiles),
		Tiles: newTiles}
	for u, p2 := range g.players {
		if u != m.Player.username {
			p2.player.socket.messages <- message{
				Type: socketInfo,
				Info: fmt.Sprintf("%v swapped a tile", m.Player.username),
			}
		}
	}
}

func (g game) handleGameFinish(m message) {
	if len(g.unusedTiles) != 0 || len(g.players[m.Player.username].unusedTiles) != 0 {
		// TODO: lower points for player
		return
	}
	// TODO
}

func (g game) handleGameTileMoved(m message) {
	gps := g.players[m.Player.username]
	tp := m.TilePositions[0]
	switch len(m.TilePositions) {
	case 1:
		if _, ok := gps.unusedTiles[tp.Tile.ID]; !ok {
			m.Player.socket.messages <- message{
				Type: socketError,
				Info: fmt.Sprintf("trying to add tile %v to the words area, but it is not in the unused pile", tp.Tile),
			}
			return
		}
		delete(gps.unusedTiles, tp.Tile.ID)
	case 2:
		if gps.unusedTiles[tp.Tile.ID] != tp.Tile {
			m.Player.socket.messages <- message{Type: socketError, Info: "trying move tile from location of other tile"}
			return
		}
		tp = m.TilePositions[1]
	}
	if _, ok := gps.usedTileLocs[tp.X]; !ok {
		gps.usedTileLocs[tp.X] = make(map[int]tile)
	}
	if _, ok := gps.usedTileLocs[tp.X][tp.Y]; ok {
		m.Player.socket.messages <- message{Type: socketError, Info: "trying move tile to used location"}
		return
	}
	gps.usedTiles[tp.Tile.ID] = tp
	gps.usedTileLocs[tp.X][tp.Y] = tp.Tile
}

func (g game) handleGameTilePositions(m message) {
	gps := g.players[m.Player.username]
	unusedTiles := make([]tile, len(gps.unusedTiles))
	usedTilePositions := make([]tilePosition, len(gps.usedTiles))
	i := 0
	for _, t := range gps.unusedTiles {
		unusedTiles[i] = t
		i++
	}
	i = 0
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
	m.Player.socket.messages <- message{
		Type:          gameTilePositions,
		Tiles:         unusedTiles,
		TilePositions: usedTilePositions,
	}
}

func (g game) handleGameInfos(m message) {
	usernames := make([]db.Username, len(g.players))
	i := 0
	for u := range g.players {
		usernames[i] = u
		i++
	}
	_, canJoin := g.players[m.Player.username]
	gi := gameInfo{
		Players:   usernames,
		CanJoin:   canJoin,
		CreatedAt: g.createdAt,
	}
	m.GameInfoChan <- gi
}

func (g game) handlePlayerDelete(m message) {
	_, ok := g.players[m.Player.username]
	if !ok {
		m.Player.socket.messages <- message{Type: socketError, Info: "cannot leave game player is not a part of"}
		return
	}
	// Note that this makes the player's tiles disappear
	delete(g.players, m.Player.username)
	if len(g.players) == 0 {
		m.Player.lobby.messages <- message{Type: gameDelete, Info: "deleting game because it has no players"}
	}
}
