package game

type (
	// Status is the state of the game
	Status int

	// Info contains information about a game
	Info struct {
		ID        ID       `json:"id"`
		Status    Status   `json:"status"`
		Players   []string `json:"players"`
		CanJoin   bool     `json:"canJoin"`
		CreatedAt string   `json:"createdAt"`
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
