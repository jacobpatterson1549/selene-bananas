// Package game contains communication structures for the game controller, lobby, and socket to use
package game

// TODO: Rename file to info.go
type (
	// Status is the state of the game
	Status int

	// Info contains information about a game
	Info struct {
		// ID is unique among the other games that currently exist.
		ID ID `json:"id"`
		// Status is the state of the game.
		Status Status `json:"status"`
		// Players is a list of the names of players in the game.
		Players []string `json:"players"`
		// CreatedAt is the game's creation time in seconds since the unix epoch.
		CreatedAt int64 `json:"createdAt"`
	}

	// ID is the id of a game
	ID int

	// PlayerName is the name of a player
	PlayerName string
)

const (
	_ Status = iota
	// NotStarted is the status of a game that is waiting for mor players to join before allowing them to rearrange their tiles.
	NotStarted
	// InProgress is the status of a game that has been started but is not finished.
	InProgress
	// Finished is the status of a game that has no tiles left and has a winner that has used all his tiles to form one group of interconnected words.
	Finished
)

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

// String returns the display value for the status.
func (s Status) String() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case InProgress:
		return "In Progress"
	case Finished:
		return "Finished"
	}
	return "?"
}
