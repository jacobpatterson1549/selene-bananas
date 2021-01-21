// Package game contains communication structures for the game controller, lobby, and socket to use.
package game

import "fmt"

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

// Rules gets the rules for the game.  Extra rules are added for customized configurations.
func (cfg Config) Rules() []string {
	rules := []string{
		"Create or join a game from the Lobby after refreshing the games list.",
		"Any player can join a game that is not started, but active games can only be joined by players who started in them.",
		"After all players have joined the game, click the Start button to start the game.",
		"Arrange unused tiles in the game area form vertical and horizontal English words.",
		"Click the Snag button to get a new tile if all tiles are used in words. This also gives other players a new tile.",
		"Click the Swap button and then a tile to exchange it for three others.",
		"Click the Finish button to run the scoring function when there are no tiles left to use.  The scoring function determines if all of the player's tiles are used and form a continuous block of English words.  If successful, the player wins. Otherwise, the player's potential winning score is decremented and play continues.",
	}
	if cfg.CheckOnSnag {
		rules = append(rules, "Words are checked to be valid when a player tries to snag a new letter.")
	}
	if cfg.Penalize {
		rules = append(rules, "If a player tries to snag unsuccessfully, the amount potential of win points is decremented")
	}
	if cfg.MinLength > 2 {
		rules = append(rules, fmt.Sprintf("All words must be at least %d letters long", cfg.MinLength))
	}
	if !cfg.AllowDuplicates {
		rules = append(rules, "Duplicate words are not allowed.")
	}
	return rules
}
