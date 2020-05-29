// +build js

// Package lobby contains code to view available games and to close the websocket.
// TODO: investigate if this is still needed. It is currently just a callthrough to the js package.
package lobby

import (
	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/js"
)

// SetGameInfos updates the game-infos table with the specified game infos.
func SetGameInfos(gameInfos []game.Info) {
	js.SetGameInfos(gameInfos)
}
