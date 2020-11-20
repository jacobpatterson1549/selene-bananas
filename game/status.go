package game

// Status is the state of the game.
type Status int

const (
	_ Status = iota
	// NotStarted is the status of a game that is waiting for mor players to join before allowing them to rearrange their tiles.
	NotStarted
	// InProgress is the status of a game that has been started but is not finished.
	InProgress
	// Finished is the status of a game that has no tiles left and has a winner that has used all his tiles to form one group of interconnected words.
	Finished
	// FinishedAllowMove is the status of a game that is finished, and but allows tiles to be moved.
	FinishedAllowMove
)

// String returns the display value for the status.
func (s Status) String() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case InProgress:
		return "In Progress"
	case Finished:
		return "Finished"
	case FinishedAllowMove:
		return "Finished, tile movement allowed"
	}
	return "?"
}
