package game

type (
	// messageType represents what the purpose of a message is
	messageType int

	// message contains information to or from a player for a game/lobby
	message struct {
		Type          messageType    `json:"type"`
		Info          string         `json:"info,omitempty"`
		Tiles         []tile         `json:"tiles,omitempty"`
		TilePositions []tilePosition `json:"tilePositions,omitempty"`
		GameInfos     []gameInfo     `json:"gameInfos,omitempty"`
		GameID        int            `json:"gameID,omitempty"`
		// pointers for inter-goroutine communication:
		Player       *player         `json:"-"`
		Game         *game           `json:"-"`
		GameInfoChan <-chan gameInfo `json:"-"`
	}
)

const (
	// not using iota because messageTypes are switched on on in javascript
	gameCreate        messageType = 1
	gameJoin          messageType = 2
	gameLeave         messageType = 3
	gameDelete        messageType = 4
	gameStart         messageType = 5
	gameSnag          messageType = 6
	gameSwap          messageType = 7
	gameFinish        messageType = 8
	gameTilePositions messageType = 9
	gameInfos         messageType = 10
	playerCreate      messageType = 11
	playerDelete      messageType = 12
	socketInfo        messageType = 13
	socketError       messageType = 14
)
