// Package tile contains structures that players interact with on game boards.
package tile

type (
	// Tile is a piece in the game.
	Tile struct {
		ID ID     `json:"id"`
		Ch letter `json:"ch"`
	}

	// Position represents a tile and its location.
	Position struct {
		Tile Tile `json:"tile"`
		X    X    `json:"x"`
		Y    Y    `json:"y"`
	}

	// ID is the id of a tile.
	ID int
	// X is the x position of a tile (column).
	X int
	// Y is the y position of a tile (row).
	Y int
)

// New creates a new Tile, throwing an error if the letter is not uppercase in the A-Z range.
func New(id ID, r rune) (*Tile, error) {
	ch, err := newLetter(r)
	if err != nil {
		return nil, err
	}
	t := Tile{
		ID: id,
		Ch: ch,
	}
	return &t, nil
}
