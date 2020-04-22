package game

import (
	"fmt"
	"math/rand"

	"github.com/jacobpatterson1549/selene-bananas/go/server/db"
)

type (
	// Game represents each game that multiple Players can participate in
	Game interface {
		Join(p player) error
		Remove(u db.Username)
		Has(u db.Username) bool
		IsEmpty() bool
		IsStarted() bool
		Start()
		Snag(p player)
		Swap(p player, r rune)
		Finish(p player)
	}
	// TODO: track tile movements

	game struct {
		words   map[string]bool
		players map[db.Username]player
		started bool
		tiles   []rune
	}
)

// func NewGame(words map[string]bool, p player) Game {
// 	players := make(map[db.Username]player, 2)
// 	return game {
// 		words: words,
// 		players: players,
// 		started: false,
// 		tiles: createTiles(),
// 	}

// }

func createTiles() []rune {
	var tiles []rune
	add := func(s string, n int) {
		for i := 0; i < len(s); i++ {
			r := s[i]
			for j := 0; j < n; j++ {
				tiles = append(tiles, rune(r))
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
	rand.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
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
	return nil
}

func (g game) Snag(p player) {
	// TODO: use channel
	// TODO: test to ensure specified player gets a tile.
}

func (g game) Swap(p player, r rune) {

}

func (g game) Finish(p player) {
	if len(g.tiles) != 0 {
		// TODO: lower points for player
		return
	}
	// TODO
}
