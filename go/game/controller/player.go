package controller

import (
	"time"

	"github.com/jacobpatterson1549/selene-bananas/go/game/board"
)

type (
	player struct {
		*time.Ticker
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

func (p *player) stopBoardRefresh() {
	p.Ticker.Stop()
}
