package socket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// gorillaSocketManager handles sending messages to different sockets,
	gorillaWebSocketManager struct {
		upgrader      *websocket.Upgrader
		playerSockets map[player.Name][]Socket
		playerGames   map[player.Name]map[game.ID]Socket
		Config        ManagerConfig
	}

	// ManagerConfig is used to create a socket Manager.
	ManagerConfig struct {
		// Log is used to log errors and other information
		Log *log.Logger // TODO: use this
		// The maximum number of sockets.
		MaxSockets int
		// The maximum number of sockets each player can open.  Must be no more than maxSockets.
		MaxPlayerSockets int
		// The config for creating new sockets
		SocketConfig Config
	}

	// Manager handles sending messages to different sockets,
	Manager interface {
		AddSocket(playerName player.Name, w http.ResponseWriter, r *http.Request) error
		SendMessage(m message.Message)
		SendGameMessage(m message.Message, id game.ID)
	}
)

// NewManager creates a new socket manager from the config.
func (cfg ManagerConfig) NewManager() (Manager, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating lobby: validation: %w", err)
	}
	u := new(websocket.Upgrader)
	g := gorillaWebSocketManager{
		upgrader:      u,
		playerSockets: make(map[player.Name][]Socket, cfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]Socket),
		Config:        cfg,
	}
	return &g, nil
}

// validate ensures the configuration has no errors.
func (cfg ManagerConfig) validate() error {
	switch {
	case cfg.Log == nil:
		return fmt.Errorf("log required")
	case cfg.MaxPlayerSockets < 1:
		return fmt.Errorf("each player must be able to open at least one socket")
	case cfg.MaxSockets < cfg.MaxPlayerSockets:
		return fmt.Errorf("players cannot create more sockets than the manager allows")
	}
	return nil
}

func (g *gorillaWebSocketManager) AddSocket(pn player.Name, w http.ResponseWriter, r *http.Request) error {
	if g.numSockets() >= g.Config.MaxSockets {
		return fmt.Errorf("no room for another socket")
	}
	if len(g.playerSockets[pn]) >= g.Config.MaxPlayerSockets {
		return fmt.Errorf("player has reached quota of sockets, close an existing one")
	}
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s, err := g.Config.SocketConfig.NewSocket(conn)
	if err != nil {
		return fmt.Errorf("creating socket in manager: %v", err)
	}
	g.playerSockets[pn] = append(g.playerSockets[pn], s)
	return nil
}

func (g *gorillaWebSocketManager) SendMessage(m message.Message) {
	// TODO
	// TODO: add player name to message
	// TODO: log if messages is to close socket
}

func (g *gorillaWebSocketManager) SendGameMessage(m message.Message, id game.ID) {
	// TODO
}

// numSockets sums the number of sockets for each player.  Not thread safe.
func (g gorillaWebSocketManager) numSockets() int {
	numSockets := 0
	for _, sockets := range g.playerSockets {
		numSockets += len(sockets)
	}
	return numSockets
}
