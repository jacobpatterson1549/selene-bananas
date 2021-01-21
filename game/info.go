package game

import "github.com/jacobpatterson1549/selene-bananas/game/board"

// Info contains information about a game.
type Info struct {
	// ID is unique among the other games that currently exist.
	ID ID `json:"id,omitempty"`
	// Status is the state of the game.
	Status Status `json:"status,omitempty"`
	// Board is the board of the game for the player.  Also used to send new movements of tiles.
	Board *board.Board `json:"board,omitempty"`
	// TilesLeft is the number of tiles left that players do not have.
	TilesLeft int `json:"tilesLeft,omitempty"`
	// Players is a list of the names of players in the game.
	Players []string `json:"players,omitempty"`
	// CreatedAt is the game's creation time in seconds since the unix epoch.
	CreatedAt int64 `json:"createdAt,omitempty"`
	// Config is the specific options used to create the game.
	Config *Config `json:"config,omitempty"`
	// FinalBoards is used to describe the state of all player's boards when a game is finished.
	FinalBoards map[string]board.Board `json:"finalBoards,omitempty"`
}

// CanJoin indicates whether or not a player can join the game.
// Players can only join games that are not started or that they were previously a part of,
func (i Info) CanJoin(playerName string) bool {
	if i.Status == NotStarted {
		return true
	}
	for _, n := range i.Players {
		if n == playerName {
			return true
		}
	}
	return false
}
