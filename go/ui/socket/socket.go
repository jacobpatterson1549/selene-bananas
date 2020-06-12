// +build js

// Package socket contains the logic to communicate with the server for the game via websocket communication
package socket

import (
	"strconv"
	"strings"

	"github.com/jacobpatterson1549/selene-bananas/go/game"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/controller"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/dom"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/lobby"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/log"
	"github.com/jacobpatterson1549/selene-bananas/go/ui/user"
)

// OnMessage is called when the websocket opens.
func OnOpen() {
	dom.SetChecked("has-websocket", true)
}

// OnMessage is called when the websocket is closing.
func OnClose() {
	dom.SetChecked("has-websocket", false)
	dom.SetChecked("has-game", false)
	dom.SetChecked("tab-4", true) // lobby tab
}

// OnMessage is called when the websocket encounters an unexpected error.
func OnError() {
	log.Error("lobby closed")
	user.Logout()
}

// OnMessage is called when the websocket receives a message.
func OnMessage(m game.Message, g controller.Game) {
	switch m.Type {
	case game.Leave:
		g.Leave()
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.BoardRefresh:
		g.ReplaceGameTiles(m.Tiles, m.TilePositions, false)
	case game.Infos:
		lobby.SetGameInfos(m.GameInfos)
	case game.PlayerDelete:
		dom.CloseWebsocket()
		g.Leave()
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.Join, game.SocketInfo:
		if m.GameStatus != 0 {
			g.SetStatus(m.GameStatus)
		}
		if m.TilesLeft != 0 {
			dom.SetValue("game-tiles-left", strconv.Itoa(m.TilesLeft))
		}
		if len(m.GamePlayers) > 0 {
			players := strings.Join(m.GamePlayers, ",")
			dom.SetValue("game-players", players)
		}
		switch {
		case len(m.TilePositions) > 0:
			silent := m.Type == game.Join
			g.ReplaceGameTiles(m.Tiles, m.TilePositions, silent)
		case len(m.Tiles) > 0:
			silent := m.Type == game.Join
			g.AddUnusedTiles(m.Tiles, silent)
		}
		if len(m.Info) > 0 {
			log.Info(m.Info)
		}
	case game.SocketError:
		log.Error(m.Info)
	case game.SocketWarning:
		log.Warning(m.Info)
	case game.SocketHTTPPing:
		dom.SocketHTTPPing()
	case game.Chat:
		log.Chat(m.Info)
	default:
		log.Error("unknown message type received")
	}
}
