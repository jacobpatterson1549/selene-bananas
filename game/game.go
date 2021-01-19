// Package game contains communication structures for the game controller, lobby, and socket to use.
package game

type (
	// ID is the id of a game.
	ID int

	// Config is used when checking player words on a snag or game finish request.
	Config struct {
		// CheckOnSnag is a flag to check the board when a player wants to snag to ensure their board has on group of valid words.
		CheckOnSnag bool `json:"checkOnSnag,omitempty"`
		// Penalize is a flag to decrement a player's points if they try to snag a tile when their board is invalid.
		Penalize bool `json:"penalize,omitempty"`
		// MinLength is the minimum allowed word length for each word on the board.
		MinLength int `json:"minLength,omitempty"`
		// AllowDuplicates is a flag for whether or not to allow duplicate words when checking the board
		AllowDuplicates bool `json:"allowDuplicates,omitempty"`
	}
)
