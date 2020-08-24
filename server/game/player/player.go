// Package player controls the game for each player
package player

import (
	"fmt"

	"github.com/jacobpatterson1549/selene-bananas/game/board"
)

type (
	// Player stores the board and other player-specific data for each player in the game.
	Player struct {
		WinPoints int
		Board     *board.Board
	}

	// Config can be used to create new players.
	Config struct {
		// WinPoints are the amount of points a player gets if they win a game.
		// A player's win points are decremented each time he attempts to unsuccessfully finish/win a game.
		WinPoints int
	}
)

// New creates a player with the winPoints defined by config and a board to modify.
func (cfg Config) New(b *board.Board) (*Player, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating player: validation: %w", err)
	}
	p := Player{
		WinPoints: cfg.WinPoints,
		Board:     b,
	}
	return &p, nil
}

// validate ensures the configuration has no errors.
func (cfg Config) validate() error {
	switch {
	case cfg.WinPoints <= 1:
		return fmt.Errorf("winPoints must be over 1")
	}
	return nil
}
