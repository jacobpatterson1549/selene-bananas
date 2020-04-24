package game

import (
	"fmt"
	"log"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// TODO: track tile movements

	game struct {
		log        *log.Logger
		createdAt  string
		words      map[string]bool
		players    map[db.Username]*player // TODO: refactor: replace db.Username with string to avoid imports in this package
		lobby      *lobby
		started    bool
		maxPlayers int
		tiles      []tile
		messages   chan message
		// the shuffle functions shuffles the slices my mutating them
		shuffleTilesFunc   func(tiles []tile)
		shufflePlayersFunc func(players []*player)
	}

	gameInfoRequest struct { // TODO: DELETEME
		p player
		c chan gameInfo
	}

	gameInfo struct {
		Players   []db.Username `json:"players"`
		CanJoin   bool          `json:"canJoin"`
		CreatedAt string        `json:"createdAt"`
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

// func (g game) close() {
// 	for _, p := range g.players {
// 		p.sendMessage(message{
// 			Type: gameDelete,
// 			Info: "game closing",
// 		})
// 	}
// }

func (g game) run() {
	messageHandlers := map[messageType]func(message){
		gameJoin:          g.handleGameJoin,
		gameLeave:         g.handleGameLeave,
		gameDelete:        g.handleGameDelete,
		gameStart:         g.handleGameStart,
		gameSnag:          g.handleGameSnag,
		gameFinish:        g.handleGameFinish,
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
	g.players[m.Player.username] = m.Player
}

func (g game) handleGameLeave(m message) {
	// TODO
}

func (g game) handleGameDelete(m message) {
	// TODO
}

func (g game) handleGameStart(m message) {
	if g.started {
		m.Player.socket.messages <- message{Type: socketError, Info: "game already started"}
		return
	}
	// TODO: what if no players exist?
	g.started = true
	newTiles := make(map[db.Username][]tile, len(g.players))
	for t := 0; t < 21; t++ {
		for u := range g.players {
			if len(g.tiles) == 0 {
				g.lobby.messages <- message{
					Type: gameDelete,
					Info: "deleting game because it can not start because there are not enough tiles",
				}
				return
			}
			t := g.tiles[0]
			g.tiles = g.tiles[1:]
			pt, ok := newTiles[u]
			if ok {
				newTiles[u] = append(pt, t)
			} else {
				// TODO debug if this condition ever occurs
				newTiles[u] = []tile{t}
			}
		}
	}
	for u, p := range g.players {
		p.socket.messages <- message{
			Type:  gameSnag,
			Info:  fmt.Sprintf("starting game with tiles: %v", newTiles[u]),
			Tiles: newTiles[u],
		}
	}
}

func (g game) handleGameSnag(m message) {
	if len(g.tiles) == 0 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tiles left to snag, use what you have to finish"}
		return
	}
	m.Player.socket.messages <- message{
		Type:  gameSwap,
		Info:  fmt.Sprintf("snagged a tile: %v", g.tiles[0]),
		Tiles: g.tiles[:1],
	}
	g.tiles = g.tiles[1:]
	otherPlayers := make([]*player, len(g.players)-1)
	i := 0
	for u2, p2 := range g.players {
		if m.Player.username != u2 {
			otherPlayers[i] = p2
			i++
		}
	}
	g.shufflePlayersFunc(otherPlayers)
	for i := 0; i < len(otherPlayers) && len(g.tiles) > 0; i++ {
		otherPlayers[i].socket.messages <- message{
			Type:  gameSwap,
			Info:  fmt.Sprintf("%v snagged a tile, adding %v to your tiles", m.Player.username, g.tiles[0]),
			Tiles: g.tiles[:1],
		}
		g.tiles = g.tiles[1:]
	}
}

// TODO: add tile moved action (inbound) differintate from forced player tile refresh

func (g game) handleGameSwap(m message) {
	if len(m.Tiles) != 1 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tile specified for swap"}
		return
	}
	t := m.Tiles[0]
	if len(g.tiles) == 0 {
		m.Player.socket.messages <- message{Type: socketInfo, Info: "no tiles left to swap, user what you have to finish"}
		g.messages <- message{Type: gameTilePositions, Player: m.Player}
		return
	}
	// TODO: ensure player had the specified tile
	g.tiles = append(g.tiles, t)
	g.shuffleTilesFunc(g.tiles)
	newTiles := make([]tile, 1)
	for i := 0; i < 3 && len(g.tiles) > 0; i++ {
		newTiles = append(newTiles, g.tiles[0])
		g.tiles = g.tiles[1:]
	}
	m.Player.socket.messages <- message{
		Type:  gameSwap,
		Info:  fmt.Sprintf("swapping %v tile for %v", t, newTiles),
		Tiles: newTiles}
	for u, p2 := range g.players {
		if u != m.Player.username {
			p2.socket.messages <- message{
				Type: socketInfo,
				Info: fmt.Sprintf("%v swapped a tile", m.Player.username),
			}
		}
	}
}

func (g game) handleGameFinish(m message) {
	if len(g.tiles) != 0 {
		// TODO: lower points for player
		return
	}
	// TODO
}

func (g game) handleGameTilePositions(m message) {
	// TODO
}

func (g game) handleGameInfos(m message) {
	// TODO
}

func (g game) handlePlayerDelete(m message) {
	_, ok := g.players[m.Player.username]
	if !ok {
		m.Player.socket.messages <- message{Type: socketError, Info: "cannot leave game player is not a part of"}
		return
	}
	delete(g.players, m.Player.username)
	if len(g.players) == 0 {
		g.lobby.messages <- message{Type: gameDelete, Info: "automatically deleting game because it has no players"}
	}
}
