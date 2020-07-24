package game

type (
	// Info contains information about a game.
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
