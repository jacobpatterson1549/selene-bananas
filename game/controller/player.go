package controller

import (
	"github.com/jacobpatterson1549/selene-bananas/game/board"
)

type (
	// player stores the board and other player-specific data for each player in the game.
	player struct {
		winPoints int
		board     *board.Board
	}
)

// decrementWinPoints decreases the win points by 1.  The winPoints are never dreased to below 2.
func (p *player) decrementWinPoints() {
	if p.winPoints > 2 {
		p.winPoints--
	}
}
