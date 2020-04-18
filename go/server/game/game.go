package game

import "github.com/jacobpatterson1549/selene-bananas/go/server/db"

type (
	// Game represents each game that multiple Players can participate in
	Game interface { // TODO: make struct
		Add(p player)
		Has(u db.User) bool
		Remove(u db.User)
		Start()
		Snag(p player)
		Swap(p player)
		Finish(p player)
	}
)

// TODO
