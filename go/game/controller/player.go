package controller

import (
	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
)

type (
	player struct {
		winPoints
		board.Board
	}

	winPoints int
)

func (p *player) decrementWinPoints() {
	if p.winPoints > 2 {
		p.winPoints--
	}
}
