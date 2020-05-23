package game

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
		// CanJoin is a flag that indicates whether or not a player can join the game.
		// Players can only join games that are not started or that they  were previously a part of,
		CanJoin bool `json:"canJoin"`
		// Created at is the time since the unix expoch in seconds.
		CreatedAt int64 `json:"createdAt"`
	}

	// ID is the id of a game
	ID int

	// PlayerName is the name of a player
	PlayerName string
)

// not using iota because gameStates hardcoded in ui javascript
const (
	InProgress Status = 1
	Finished   Status = 2
	NotStarted Status = 3
)
