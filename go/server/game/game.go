package game

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// TODO: track tile movements

	game interface {
		handleRequest(m message)
		infoRequest(r gameInfoRequest)
	}

	gameImpl struct {
		log        *log.Logger
		createdAt  string
		words      map[string]bool
		players    map[db.Username]player
		started    bool
		maxPlayers int
		tiles      []tile
		messages   chan message
		gameInfos  chan gameInfoRequest
		// the shuffle functions shuffles the slices my mutating them
		shuffleTilesFunc   func(tiles []tile)
		shufflePlayersFunc func(players []player)
	}

	gameInfoRequest struct {
		u db.Username
		c chan gameInfo
	}
)

// newGame creates a new game with randomly shuffled tiles and players
func newGame(log *log.Logger, words map[string]bool, p player) game {
	players := make(map[db.Username]player, 2)
	g := gameImpl{
		log:        log,
		createdAt:  time.Now().String(),
		words:      words,
		players:    players,
		started:    false,
		maxPlayers: 8,
		messages:   make(chan message, 64),
		shuffleTilesFunc: func(tiles []tile) {
			rand.Shuffle(len(tiles), func(i, j int) {
				tiles[i], tiles[j] = tiles[j], tiles[i]
			})
		},
		shufflePlayersFunc: func(players []player) {
			rand.Shuffle(len(players), func(i, j int) {
				players[i], players[j] = players[j], players[i]
			})
		},
	}
	g.tiles = g.createTiles()
	go g.run()
	return g
}

func (g gameImpl) handleRequest(m message) {
	g.messages <- m
}

func (g gameImpl) handleProcess(m message) {
	// TODO: switch on messageType, call correct function
}

func (g gameImpl) infoRequest(r gameInfoRequest) {
	g.gameInfos <- r
}
func (g gameImpl) infoProcess(u db.Username) gameInfo {
	players := make([]db.Username, len(g.players))
	i := 0
	_, canJoin := g.players[u]
	for u := range g.players {
		players[i] = u
		i++
	}
	return gameInfo{
		Players:   players,
		CanJoin:   canJoin,
		CreatedAt: g.createdAt,
	}
}

func (g gameImpl) createTiles() []tile {
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

func (g gameImpl) run() {
	for {
		select {
		case m, ok := <-g.messages:
			if !ok {
				g.close()
			}
			g.handleProcess(m)
		case r, ok := <-g.gameInfos:
			if !ok {
				g.close()
			}
			gameInfo := g.infoProcess(r.u)
			r.c <- gameInfo
		}
	}
}

func (g gameImpl) close() {
	for _, p := range g.players {
		p.sendMessage(infoMessage{
			Type: gameClose,
			Info: "game closing",
		})
	}
	close(g.messages)
	close(g.gameInfos)
}

func (g gameImpl) add(p player) {
	if !g.started {
		return
	}
	if _, ok := g.players[p.username()]; ok {
		return
	}
	if len(g.players) >= g.maxPlayers {
		return
	}
	g.players[p.username()] = p
}

func (g gameImpl) remove(u db.Username) {
	delete(g.players, u)
}

func (g gameImpl) isEmpty() bool {
	return len(g.players) == 0
}

// TODO: where is this used?  is it needed?
func (g gameImpl) isStarted() bool {
	return g.started
}

func (g gameImpl) start() error {
	// TODO: use chanel to start to prevent race conditions.
	if g.started {
		return fmt.Errorf("game already started")
	}
	g.started = true
	newTiles := make(map[db.Username][]tile, len(g.players))
	for t := 0; t < 21; t++ {
		for u := range g.players {
			if len(g.tiles) == 0 {
				return fmt.Errorf("could not start game because there ar not enough tiles")
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
		g.addTiles(fmt.Sprintf("startingGame with tiles: %v", newTiles[u]), p, newTiles[u]...)
	}
	return nil
}

func (g gameImpl) snag(p player) {
	// TODO: test to ensure specified player gets a tile.
	if len(g.tiles) == 0 {
		return
	}
	message := fmt.Sprintf("%v snagged a tile, adding %v", p.username(), g.tiles[0])
	g.addTiles(message, p, g.tiles[0])
	g.tiles = g.tiles[1:]
	otherPlayers := make([]player, len(g.players)-1)
	i := 0
	for u2, p2 := range g.players {
		if p.username() != u2 {
			otherPlayers[i] = p2
			i++
		}
	}
	g.shufflePlayersFunc(otherPlayers)
	for i := 0; i < len(otherPlayers) && len(g.tiles) > 0; i++ {
		message := fmt.Sprintf("%s snagged a tile, adding %v to your tiles", p.username(), g.tiles[0])
		g.addTiles(message, otherPlayers[i], g.tiles[0])
		g.tiles = g.tiles[1:]
	}
}

func (g gameImpl) swap(p player, t tile) {
	// TODO: ensure player had the specified tile
	g.tiles = append(g.tiles, t)
	g.shuffleTilesFunc(g.tiles)
	newTiles := make([]tile, 1)
	for i := 0; i < 3 && len(g.tiles) > 0; i++ {
		newTiles = append(newTiles, g.tiles[0])
		g.tiles = g.tiles[1:]
	}
	g.addTiles(fmt.Sprintf("swapping %v tile for %v", t, newTiles), p, newTiles...)
}

func (g gameImpl) finish(p player) {
	if len(g.tiles) != 0 {
		// TODO: lower points for player
		return
	}
	// TODO
}

func (gameImpl) addTiles(info string, p player, tiles ...tile) {
	p.sendMessage(tilesMessage{
		Info:  info,
		Tiles: tiles,
	})
}
