package socket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
)

type (
	// Manager handles sending messages to different sockets,
	Manager struct {
		upgrader      Upgrader
		playerSockets map[player.Name][]Socket
		playerGames   map[player.Name]map[game.ID]Socket
		ManagerConfig
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

	// Upgrader turns a http request into a websocket.
	Upgrader interface {
		// Upgrade creates a Conn from the HTTP request.
		Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error)
	}
)

// NewManager creates a new socket manager from the config.
func (cfg ManagerConfig) NewManager() (*Manager, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("creating socket manager: validation: %w", err)
	}
	u := newGorillaUpgrader()
	sm := Manager{
		upgrader:      u,
		playerSockets: make(map[player.Name][]Socket, cfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]Socket),
		ManagerConfig: cfg,
	}
	return &sm, nil
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

// Run consumes messages from the message channel.  This channel is used to create sockets and send messages to them.
// The messages recieved from sockets are send on an "out" channel to be read games.
func (sm *Manager) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	// TODO
	return nil
}

// AddSocket adds a socket for the player to the manager.
func (sm *Manager) AddSocket(pn player.Name, w http.ResponseWriter, r *http.Request) error {
	if sm.numSockets() >= sm.MaxSockets {
		return fmt.Errorf("no room for another socket")
	}
	if len(sm.playerSockets[pn]) >= sm.MaxPlayerSockets {
		return fmt.Errorf("player has reached quota of sockets, close an existing one")
	}
	c, err := sm.upgrader.Upgrade(w, r)
	conn, ok := c.(Conn)
	if !ok {
		return fmt.Errorf("%T is an invalid Conn", c)
	}
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s, err := sm.SocketConfig.NewSocket(conn)
	if err != nil {
		return fmt.Errorf("creating socket in manager: %v", err)
	}
	sm.playerSockets[pn] = append(sm.playerSockets[pn], *s)
	return nil
}

// SendMessage delivers a message to the socket for a player in the specified game.
func (sm *Manager) SendMessage(m message.Message) {
	// TODO
	// TODO: add player name to message
	// TODO: log if messages is to close socket
}

// SendGameMessage delivers a message to all sockets in a particular game.
func (sm *Manager) SendGameMessage(m message.Message, id game.ID) {
	// TODO
}

// numSockets sums the number of sockets for each player.  Not thread safe.
func (sm Manager) numSockets() int {
	numSockets := 0
	for _, sockets := range sm.playerSockets {
		numSockets += len(sockets)
	}
	return numSockets
}
