package socket

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/jacobpatterson1549/selene-bananas/game"
	"github.com/jacobpatterson1549/selene-bananas/game/message"
	"github.com/jacobpatterson1549/selene-bananas/game/player"
	"github.com/jacobpatterson1549/selene-bananas/server/runner"
)

type (
	// Manager handles sending messages to different sockets,
	Manager struct {
		runner.Runner
		upgrader      Upgrader
		playerSockets map[player.Name]map[net.Addr]chan<- message.Message
		playerGames   map[player.Name]map[game.ID]net.Addr
		socketOut     chan message.Message
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
		playerSockets: make(map[player.Name]map[net.Addr]chan<- message.Message, cfg.MaxSockets),
		playerGames:   make(map[player.Name]map[game.ID]net.Addr),
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

// Run consumes messages from the message channel.  This channel is used to create sockets and send messages from games to them.
// The messages recieved from sockets are sent on an "out" channel to be read by games.
func (sm *Manager) Run(ctx context.Context, in <-chan message.Message) (<-chan message.Message, error) {
	if err := sm.Runner.Run(); err != nil {
		return nil, fmt.Errorf("running socket manager: %v", err)
	}
	sm.socketOut = make(chan message.Message)
	out := make(chan message.Message)
	go func() {
		defer close(out)
		defer sm.Runner.Finish()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-in:
				if !ok {
					return
				}
				sm.handleGameMessage(ctx, m)
			case m := <-sm.socketOut:
				sm.handleSocketMessage(ctx, m, out)
			}
		}
	}()
	return out, nil
}

// AddSocket runs and adds a socket for the player to the manager.
func (sm *Manager) AddSocket(ctx context.Context, pn player.Name, w http.ResponseWriter, r *http.Request) error {
	if !sm.Runner.IsRunning() {
		return fmt.Errorf("socket manager not running")
	}
	if sm.numSockets() >= sm.MaxSockets {
		return fmt.Errorf("no room for another socket")
	}
	if len(sm.playerSockets[pn]) >= sm.MaxPlayerSockets {
		return fmt.Errorf("player has reached quota of sockets, close an existing one")
	}
	conn, err := sm.upgrader.Upgrade(w, r)
	if err != nil {
		return fmt.Errorf("upgrading to websocket connection: %w", err)
	}
	s, err := sm.SocketConfig.NewSocket(conn)
	if err != nil {
		return fmt.Errorf("creating socket in manager: %v", err)
	}
	socketIn := make(chan message.Message)
	if err := s.Run(ctx, socketIn, sm.socketOut); err != nil {
		return fmt.Errorf("running socket")
	}
	a := conn.RemoteAddr()
	if sm.hasSocket(a) {
		return fmt.Errorf("socket already exists with address of %v", a)
	}
	playerSockets, ok := sm.playerSockets[pn]
	switch {
	case ok:
		playerSockets[a] = socketIn
	default:
		sm.playerSockets[pn] = map[net.Addr]chan<- message.Message{
			a: socketIn,
		}
	}
	return nil
}

// numSockets sums the number of sockets for each player.  Not thread safe.
func (sm *Manager) numSockets() int {
	numSockets := 0
	for _, sockets := range sm.playerSockets {
		numSockets += len(sockets)
	}
	return numSockets
}

// hasSocket determines if a socket exists in the manager with the same address.  Not thread safe.
func (sm *Manager) hasSocket(a net.Addr) bool {
	for _, sockets := range sm.playerSockets {
		for a0 := range sockets {
			if a0 == a {
				return true
			}
		}
	}
	return false
}

func (sm *Manager) removeSocket(a net.Addr) {

}

// handleGameMessage writes the message to the appropriate sockets in the manager.
func (sm *Manager) handleGameMessage(ctx context.Context, m message.Message) {
	// TODO
}

// handleSocketMessage writes the socket message to to the out channel, possibly taking action.
func (sm *Manager) handleSocketMessage(ctx context.Context, m message.Message, out chan<- message.Message) {
	// TODO
}
