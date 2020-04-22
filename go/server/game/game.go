package game

import (
	"fmt"
	"math/rand"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// TODO: track tile movements

	tile rune

	game struct {
		words   map[string]bool
		players map[db.Username]player
		started bool
		tiles   []tile
		lobby   lobby
		message
		// the shuffle functions shuffles the slices my mutating them
		shuffleTilesFunc   func(tiles []tile)
		shufflePlayersFunc func(players []player)
	}
)

// Run starts the lobby
func Run() {
	// for {
	// 	select m, ok :=
	// }
}

// newGame creates a new game with randomly shuffled tiles and players
func newGame (words map[string]bool, p player) game {
	players := make(map[db.Username]player, 2)
	g := game{
		words:   words,
		players: players,
		started: false,
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
	return g
}

func (g game) createTiles() []tile {
	var tiles []tile
	add := func(s string, n int) {
		for i := 0; i < len(s); i++ {
			r := s[i]
			for j := 0; j < n; j++ {
				tiles = append(tiles, tile(r))
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
	return tiles
}

func (g game) Join(p player) error {
	if !g.started {
		return fmt.Errorf("game is not started")
	}
	if g.Has(p.username) {
		return fmt.Errorf("user already in current game: %v", p.username)
	}
	g.players[p.username] = p
	return nil
}

func (g game) Remove(u db.Username) {
	delete(g.players, u)
}

func (g game) Has(u db.Username) bool {
	_, ok := g.players[u]
	return ok
}

func (g game) IsEmpty() bool {
	return len(g.players) == 0
}

func (g game) IsStarted() bool {
	return g.started
}

func (g game) Start() error {
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
		p.addTiles(newTiles[u]...)
	}
	return nil
}

func (g game) Snag(p player) {
	// TODO: use channel
	// TODO: test to ensure specified player gets a tile.
	if len(g.tiles) == 0 {
		return
	}
	p.addTiles(g.tiles[0])
	g.tiles = g.tiles[1:]
	otherPlayers := make([]player, len(g.players)-1)
	i := 0
	for u2, p2 := range g.players {
		if p.username != u2 {
			otherPlayers[i] = p2
			i++
		}
	}
	g.shufflePlayersFunc(otherPlayers)
	for i := 0; i < len(otherPlayers) && len(g.tiles) > 0; i++ {
		otherPlayers[i].addTiles(g.tiles[0])
		g.tiles = g.tiles[1:]
	}
}

func (g game) Swap(p player, t tile) {
	// TODO: ensure player had the specified tile
	g.tiles = append(g.tiles, t)
	g.shuffleTilesFunc(g.tiles)
	newTiles := make([]tile, 1)
	for i := 0; i < 3 && len(g.tiles) > 0; i++ {
		newTiles = append(newTiles, g.tiles[0])
		g.tiles = g.tiles[1:]
	}
	p.addTiles(newTiles...)
}

func (g game) Finish(p player) {
	if len(g.tiles) != 0 {
		// TODO: lower points for player
		return
	}
	// TODO
}
