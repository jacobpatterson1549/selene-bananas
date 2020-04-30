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
		TilesLeft     int            `json:"tilesLeft,omitempty"`
		GameInfos     []gameInfo     `json:"gameInfos,omitempty"`
		GameID        int            `json:"gameID,omitempty"`
		GameState     gameState      `json:"gameState,omitempty"`
		GamePlayers   []string       `json:"gamePlayers,omitempty"`
		// pointers for inter-goroutine communication:
		Player       *player         `json:"-"`
		Game         *game           `json:"-"`
		GameInfoChan chan<- gameInfo `json:"-"`
	}
)

const (
	// not using iota because messageTypes are switched on on in javascript
	gameCreate        messageType = 1
	gameJoin          messageType = 2
	gameLeave         messageType = 3
	gameDelete        messageType = 4 // TODO: remove this, add gameState = delete
	gameStateChange   messageType = 5
	gameSnag          messageType = 7
	gameSwap          messageType = 8
	gameTileMoved     messageType = 9
	gameTilePositions messageType = 10 // TODO: is this needed anymore?
	gameInfos         messageType = 11
	playerCreate      messageType = 12
	playerDelete      messageType = 13
	socketInfo        messageType = 14
	socketError       messageType = 15
	socketClosed      messageType = 16
	socketHTTPPing    messageType = 17 // the socket should send a http ping every so often  This is  explicitly for heroku, which will shut down the server if 30 minutes passes between http requests
	gameChatRecv      messageType = 18
	gameChatSend      messageType = 19
)
